"""What tmux-resurrect saves of the tmux sessions (tmux/resurrect-drop.sh):
throwaway ones stay out. Each test has a tmux server of its own (TMUX_TMPDIR)
and an empty HOME, so it never touches the real one; a throwaway session only
lives while a client shows it, so pexpect gives that client a terminal."""

import shutil
import tempfile
from pathlib import Path
import subprocess
import time

import pexpect
import pytest

from conftest import CONFIG

DROP = CONFIG / "tmux" / "resurrect-drop.sh"


@pytest.fixture
def tmux(tmp_path):
    """run(args) runs tmux on the test's server; throwaway() is a session
    like a second Ghostty window's (ghostty/tmux-start.sh), shown in a
    terminal closed at the end."""
    home = tmp_path / "home"
    home.mkdir()
    # A unix socket path has ~104 characters at most: pytest's tmp_path is too long
    sockets = Path(tempfile.mkdtemp(prefix="tmux-test-", dir="/tmp"))
    env = {"HOME": str(home), "TMUX_TMPDIR": str(sockets), "PATH": "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin",
           "SHELL": "/bin/sh", "TERM": "xterm-256color"}
    terminals = []

    def run(*args):
        return subprocess.run(["tmux", *args], env=env, capture_output=True, text=True, timeout=10).stdout

    def sessions():
        # ':' between the fields: tmux prints a tab as _, and a session name can't hold a ':'
        out = run("list-sessions", "-F", "#{session_name}:#{destroy-unattached}")
        return dict(line.split(":") for line in out.splitlines())

    def clients():
        return len(run("list-clients").splitlines())

    def throwaway():
        before = clients()
        child = pexpect.spawn("tmux", ["new-session", ";", "set-option", "destroy-unattached", "on"],
                              cwd=str(home), env=env, timeout=10, dimensions=(24, 80))
        terminals.append(child)
        for _ in range(50):  # until it is attached: one more client
            if clients() > before:
                return next(n for n, destroy in sessions().items() if destroy == "on")
            time.sleep(0.1)
        raise AssertionError(f"no client: {sessions()}")

    run.sessions = sessions
    run.throwaway = throwaway
    run.home = home
    run.sockets = sockets
    yield run
    for child in terminals:
        child.terminate(force=True)
    run("kill-server")
    shutil.rmtree(sockets, ignore_errors=True)


def test_resurrect_drops_throwaway_sessions_from_the_save(tmux, tmp_path):
    tmux("new-session", "-d", "-s", "keep")
    tmp = tmux.throwaway()
    save = tmp_path / "tmux_resurrect_20261028.txt"
    lines = [
        "pane\tkeep\t1\t1\t:*\t1\ttitle\t:/w/keep\t1\tzsh\t:",
        f"pane\t{tmp}\t1\t1\t:*\t1\ttitle\t:/w\t1\tzsh\t:",
        "window\tkeep\t1\t:zsh\t1\t:*\tlayout\t:",
        f"window\t{tmp}\t1\t:zsh\t1\t:*\tlayout\t:",
        f"grouped_session\t{tmp}\tkeep\t:1\t:1",
        "state\tkeep\t",
    ]
    save.write_text("\n".join(lines) + "\n")
    env = {"HOME": str(tmux.home), "TMUX_TMPDIR": str(tmux.sockets), "PATH": "/opt/homebrew/bin:/usr/bin:/bin"}
    subprocess.run(["/bin/bash", str(DROP), str(save)], env=env, check=True, timeout=10)
    kept = save.read_text().splitlines()
    assert kept == [lines[0], lines[2], lines[5]]


def test_resurrect_drop_with_nothing_to_drop_leaves_the_save_alone(tmux, tmp_path):
    tmux("new-session", "-d", "-s", "keep")
    save = tmp_path / "save.txt"
    save.write_text("pane\tkeep\t1\n")
    env = {"HOME": str(tmux.home), "TMUX_TMPDIR": str(tmux.sockets), "PATH": "/opt/homebrew/bin:/usr/bin:/bin"}
    subprocess.run(["/bin/bash", str(DROP), str(save)], env=env, check=True, timeout=10)
    assert save.read_text() == "pane\tkeep\t1\n"
    assert sorted(p.name for p in tmp_path.iterdir() if p.name.startswith("save")) == ["save.txt"], "no temp file left"


def test_tmux_conf_runs_the_drop_hook(tmux):
    conf = (CONFIG / "tmux" / "tmux.conf").read_text()
    assert "set -g @resurrect-hook-post-save-layout '~/.config/tmux/resurrect-drop.sh'" in conf
    assert conf.index("@resurrect-hook-post-save-layout") < conf.index("tmux-resurrect/resurrect.tmux"), "set before the plugin loads"
