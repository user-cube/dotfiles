"""The history block of .zshrc: ↑ and Ctrl+R see what another shell ran since
this one's last prompt, and ↑ keeps its prefix search going. Two interactive
zsh in terminals (pexpect) share a throwaway history file; they load only
.zshrc's history options and its ↑ / Ctrl+R widgets, not Oh My Zsh or p10k,
with ↑ bound the way Oh My Zsh binds it."""

import os
import re
import shutil
import time

import pexpect
import pytest

from conftest import REPO

ZSHRC = (REPO / ".zshrc").read_text()
PROMPT = "PROMPT> "
UP = "\x1b[A"        # normal cursor mode: up-line-or-beginning-search
UP_APP = "\x1bOA"    # application cursor mode, bound to up-line-or-history here
CTRL_R = "\x12"


def history_block() -> str:
    """.zshrc's history options and the widgets that read the other shells' commands."""
    options = re.findall(r"^setopt HIST\w*|^setopt SHARE_HISTORY|^setopt EXTENDED_HISTORY", ZSHRC, re.M)
    start = ZSHRC.index("# ↑ and Ctrl+R first read")
    end = ZSHRC.index("\n", ZSHRC.index("zle -N fzf-history-widget _history_fzf"))
    return "\n".join(options) + "\n" + ZSHRC[start:end] + "\n"


@pytest.fixture
def shells(tmp_path):
    """shell() starts an interactive zsh on the shared history; all closed at the end."""
    rc = tmp_path / "zdotdir" / ".zshrc"
    rc.parent.mkdir()
    fzf = 'source <(fzf --zsh)\n' if shutil.which("fzf") else ""
    rc.write_text(
        f"HISTFILE={tmp_path / 'history'}\nHISTSIZE=1000\nSAVEHIST=1000\n"
        "autoload -U up-line-or-beginning-search\nzle -N up-line-or-beginning-search\n"
        f"bindkey '^[[A' up-line-or-beginning-search\nbindkey '^[OA' up-line-or-history\n{fzf}"
        + history_block() + f"PS1='{PROMPT}'\n"
    )
    env = {k: v for k, v in os.environ.items() if not k.startswith(("FZF_", "TMUX"))}
    env.update(ZDOTDIR=str(rc.parent), TERM="xterm-256color")
    started = []

    def shell():
        child = pexpect.spawn("zsh", ["-i"], env=env, encoding="utf-8", timeout=10, dimensions=(30, 120))
        child.expect(PROMPT)
        started.append(child)
        return child

    yield shell
    for child in started:
        child.terminate(force=True)


def run(child, command: str):
    child.sendline(command)
    child.expect(PROMPT)
    time.sleep(0.2)  # the history file is written as the command starts; let it land


def recall(child, typed: str, *keys: str) -> str:
    """Type `typed`, press keys, then Enter: what the recalled command printed.
    The commands print res<N> only when run ($((N)) on the line, N in the output)."""
    child.send(typed)
    for key in keys:
        child.send(key)
        time.sleep(0.3)
    child.send("\r")
    child.expect(PROMPT)
    found = re.findall(r"res\d", re.sub(r"res\$\(\(\d\)\)", "", child.before))
    return found[-1] if found else ""


@pytest.mark.parametrize("key", [UP, UP_APP], ids=["up", "up-application-mode"])
def test_up_sees_what_another_shell_ran(shells, key):
    here, other = shells(), shells()
    run(other, "echo res$((1))")
    assert recall(here, "", key) == "res1"


def test_up_keeps_the_prefix_search_going(shells):
    here, other = shells(), shells()
    run(here, "echo p res$((1))")
    run(here, "echo zz")
    run(other, "echo p res$((2))")
    run(other, "echo zz2")
    assert recall(here, "echo p ", UP) == "res2"
    assert recall(here, "echo p ", UP, UP) == "res1"


def test_up_prefix_search_with_nothing_new(shells):
    """No other shell wrote anything: the history isn't read again, which would
    break the next ↑'s prefix search."""
    here = shells()
    for command in ("echo p res$((1))", "echo zz", "echo p res$((2))", "echo zz2"):
        run(here, command)
    assert recall(here, "echo p ", UP, UP) == "res1"


@pytest.mark.skipif(not shutil.which("fzf"), reason="needs fzf")
def test_ctrl_r_sees_what_another_shell_ran(shells):
    here, other = shells(), shells()
    run(other, "echo res$((3))")
    here.send(CTRL_R)
    here.expect(r"\d+/\d+")  # fzf is up and reading keys
    for ch in "res":
        here.send(ch)
        time.sleep(0.1)
    here.expect(r"[1-9]\d*/\d+")  # it matched
    here.send("\r")
    here.expect(PROMPT)
    assert recall(here, "") == "res3"
