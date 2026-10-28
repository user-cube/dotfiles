"""The tmux sessions VS Code's terminal makes (vscode/tmux.sh) and what
tmux-resurrect saves of them (tmux/resurrect-drop.sh). Each test has a tmux
server of its own (TMUX_TMPDIR) and an empty HOME, so it never touches the
real one; tmux.sh attaches, so pexpect gives it a terminal."""

import shutil
import tempfile
from pathlib import Path
import subprocess
import time

import pexpect
import pytest

from conftest import CONFIG

TMUX_SH = CONFIG / "vscode" / "tmux.sh"
DROP = CONFIG / "tmux" / "resurrect-drop.sh"


@pytest.fixture
def tmux(tmp_path):
    """run(args) runs tmux on the test's server; open(cwd) is a VS Code
    terminal in cwd (closed at the end)."""
    home = tmp_path / "home"
    (home / ".config" / "tmux").mkdir(parents=True)
    for script in ("start-server.sh", "preload-sessions.sh"):
        shutil.copy(CONFIG / "tmux" / script, home / ".config" / "tmux" / script)
    # A unix socket path has ~104 characters at most: pytest's tmp_path is too long
    sockets = Path(tempfile.mkdtemp(prefix="tmux-test-", dir="/tmp"))
    env = {"HOME": str(home), "TMUX_TMPDIR": str(sockets), "PATH": "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin",
           "SHELL": "/bin/sh", "TERM": "xterm-256color"}
    terminals = []

    def run(*args):
        return subprocess.run(["tmux", *args], env=env, capture_output=True, text=True, timeout=10).stdout

    def sessions():
        # ':' between the fields: tmux prints a tab as _, and a session name can't hold a ':'
        out = run("list-sessions", "-F", "#{session_name}:#{@vscode}:#{destroy-unattached}:#{session_attached}")
        return {name: (vscode, destroy, attached) for name, vscode, destroy, attached in
                (line.split(":") for line in out.splitlines())}

    def clients():
        return len(run("list-clients").splitlines())

    def open_terminal(cwd):
        before = clients()
        child = pexpect.spawn("/bin/bash", [str(TMUX_SH)], cwd=str(cwd), env=env, timeout=10, dimensions=(24, 80))
        terminals.append(child)
        for _ in range(50):  # until it is attached: one more client
            if clients() > before:
                return child
            time.sleep(0.1)
        raise AssertionError(f"no client for {cwd}: {sessions()}")

    run.sessions = sessions
    run.open = open_terminal
    run.home = home
    run.sockets = sockets
    yield run
    for child in terminals:
        child.terminate(force=True)
    run("kill-server")
    shutil.rmtree(sockets, ignore_errors=True)


def test_a_folder_gets_its_own_session_marked_vscode(tmux):
    project = tmux.home / "work" / "my.proj"
    project.mkdir(parents=True)
    tmux.open(project)
    vscode, destroy, attached = tmux.sessions()["my_proj"]
    assert (vscode, destroy, attached) == ("1", "off", "1")
    assert "main" in tmux.sessions(), "start-server.sh started the server with main"


def test_the_home_directory_gets_a_throwaway_session(tmux):
    tmux.open(tmux.home)
    made = {name: v for name, v in tmux.sessions().items() if name != "main"}
    assert len(made) == 1, made
    (name, (vscode, destroy, _)), = made.items()
    assert name != "home", "not a session named after the home directory"
    assert (vscode, destroy) == ("", "on")


def test_a_second_terminal_on_a_shown_folder_is_throwaway_and_a_closed_one_comes_back(tmux):
    project = tmux.home / "app"
    project.mkdir()
    first = tmux.open(project)
    tmux.open(project)
    throwaway = [n for n, (_, destroy, _) in tmux.sessions().items() if destroy == "on"]
    assert len(throwaway) == 1 and throwaway[0] != "app"
    first.terminate(force=True)  # the VS Code window reloads: its terminal goes
    time.sleep(0.5)
    assert tmux.sessions()["app"][2] == "0", "app outlives its terminal"
    count = len(tmux.sessions())
    tmux.open(project)
    assert tmux.sessions()["app"][2] == "1", "the new terminal attaches it again"
    assert len(tmux.sessions()) == count, "no new session for it"


def test_resurrect_drops_vscode_and_throwaway_sessions_from_the_save(tmux, tmp_path):
    tmux("new-session", "-d", "-s", "keep")
    tmux("new-session", "-d", "-s", "vs", ";", "set-option", "@vscode", "1")
    tmux.open(tmux.home)  # a throwaway session only lives while a terminal shows it
    tmp = next(n for n, (_, destroy, _) in tmux.sessions().items() if destroy == "on")
    tmux("new-session", "-d", "-s", "vs-from-o")  # same kind of name, not marked: kept
    save = tmp_path / "tmux_resurrect_20261028.txt"
    lines = [
        "pane\tkeep\t1\t1\t:*\t1\ttitle\t:/w/keep\t1\tzsh\t:",
        "pane\tvs\t1\t1\t:*\t1\ttitle\t:/w/vs\t1\tzsh\t:",
        f"pane\t{tmp}\t1\t1\t:*\t1\ttitle\t:/w\t1\tzsh\t:",
        "pane\tvs-from-o\t1\t1\t:*\t1\ttitle\t:/w/vs\t1\tzsh\t:",
        "window\tkeep\t1\t:zsh\t1\t:*\tlayout\t:",
        "window\tvs\t1\t:zsh\t1\t:*\tlayout\t:",
        f"window\t{tmp}\t1\t:zsh\t1\t:*\tlayout\t:",
        "grouped_session\tvs\tkeep\t:1\t:1",
        "state\tkeep\tvs",
    ]
    save.write_text("\n".join(lines) + "\n")
    env = {"HOME": str(tmux.home), "TMUX_TMPDIR": str(tmux.sockets), "PATH": "/opt/homebrew/bin:/usr/bin:/bin"}
    subprocess.run(["/bin/bash", str(DROP), str(save)], env=env, check=True, timeout=10)
    kept = save.read_text().splitlines()
    assert kept == [lines[0], lines[3], lines[4], lines[8]]


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
