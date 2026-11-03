"""The install path end to end: dotfiles-sync (and so install.sh) into an empty
fake HOME, from a clone of this checkout with its uncommitted changes, with a
bare origin of its own. One scenario, in order (the tests share the module's
HOME): a first run installs everything, a second changes nothing, the warnings,
Hue, the VS Code links, a changed .go file rebuilds only its tool, the built
binaries work, dotfiles-diff, and a broken build fails the sync.

Safe to run on this Mac: the apps a sync touches are fakes first on PATH
(code, defaults for Rex's Settings, tmux, aerospace, sketchybar, brew, pgrep),
so nothing reaches the real ones. The SketchyBar icon map is downloaded (needs
the network). Needs go, zsh, rsync."""

import json
import os
import shutil
import subprocess
from pathlib import Path

import pytest

from conftest import REPO

pytestmark = pytest.mark.skipif(
    not all(shutil.which(c) for c in ("go", "zsh", "rsync")), reason="needs go, zsh and rsync")

TOOLS = [
    "git_hooks/go/commit-msg/commit-msg",
    "git_hooks/go/sops-guard/sops-guard",
    ".config/dotfiles/local-files/local-files",
    ".config/kube-guard/kube-guard",
    ".config/sketchybar/plugins/check/check",
    ".claude/hooks/deny-sensitive-reads/deny-sensitive-reads",
]
LAUNCH_AGENT = "Library/LaunchAgents/com.user-cube.sops-on-login.plist"
# defaults, for Rex's Settings (settings.py): what is written is exported back,
# as the real one does, so a second apply finds nothing to change
DEFAULTS = """#!/usr/bin/env python3
import plistlib, sys
from pathlib import Path
store = Path(sys.argv[0]).with_suffix(".plist")
prefs = plistlib.loads(store.read_bytes()) if store.exists() else {}
cmd, args = sys.argv[1], sys.argv[3:]
if cmd == "export":
    sys.stdout.buffer.write(plistlib.dumps(prefs))
elif cmd == "write":
    key, kind, value = args
    prefs[key] = {"-bool": lambda v: v == "true", "-int": int, "-float": float,
                  "-string": str, "-data": bytes.fromhex}[kind](value)
    store.write_bytes(plistlib.dumps(prefs))
"""


def listed(path: Path) -> list[str]:
    """The entries of a list file: no comments, no blank lines."""
    return [line.strip() for line in path.read_text().splitlines()
            if line.strip() and not line.strip().startswith("#")]


class Install:
    """The fake HOME, the clone and its origin, and how to run things in them."""

    def __init__(self, t: Path):
        self.t, self.home, self.clone, self.origin = t, t / "home", t / "dotfiles", t / "origin.git"
        self.bin = t / "bin"
        self.home.mkdir()
        self.bin.mkdir()
        go_env = {k: subprocess.run(["go", "env", k], capture_output=True, text=True).stdout.strip()
                  for k in ("GOCACHE", "GOMODCACHE")}
        self.env = dict(os.environ, HOME=str(self.home), PATH=f"{self.bin}:{os.environ['PATH']}",
                        DOTFILES_DIR=str(self.clone), **go_env)
        self.env.pop("KUBECONFIG", None)

    def fake(self, name: str, script: str):
        path = self.bin / name
        path.write_text(script if script.startswith("#!") else "#!/bin/sh\n" + script)
        path.chmod(0o755)

    def git(self, *args, cwd=None) -> str:
        """git without the user's config: no global hooks, no signing."""
        env = dict(os.environ, HOME=str(self.t), GIT_CONFIG_NOSYSTEM="1")
        out = subprocess.run(["git", "-c", "user.name=test", "-c", "user.email=test@example.com", *args],
                             cwd=cwd or self.clone, env=env, capture_output=True, text=True, check=True)
        return out.stdout.strip()

    def zsh(self, script: str, *args) -> subprocess.CompletedProcess:
        return subprocess.run(["zsh", "-f", "-c", f'source "$DOTFILES_DIR/.config/zsh/dotfiles.zsh" && {script}',
                               "_", *args], env=self.env, capture_output=True, text=True, timeout=600)

    def sync(self) -> subprocess.CompletedProcess:
        """dotfiles-sync; stdout without its own "Already up to date." and "Done." lines."""
        out = self.zsh("dotfiles-sync")
        out.stdout = "\n".join(line for line in out.stdout.splitlines()
                               if not line.startswith(("Already up to date.", "Done. "))).strip()
        return out

    def diff(self, *args) -> str:
        out = self.zsh('dotfiles-diff "$@"', *args)
        return out.stdout + out.stderr

    def state(self) -> dict:
        """Every file under HOME and its content: to tell that nothing changed."""
        return {str(p.relative_to(self.home)): p.read_bytes()
                for p in self.home.rglob("*") if p.is_file() and not p.is_symlink()}


@pytest.fixture(scope="module")
def install(tmp_path_factory):
    i = Install(tmp_path_factory.mktemp("install"))
    t = i.t
    # VS Code: lists rui.hue once Hue's install.sh installed its .vsix, and the
    # marketplace ids installed since (install-extensions.sh)
    i.fake("code", f'''case "$1" in
  --list-extensions) [ -f "{t}/hue-installed" ] && echo rui.hue; cat "{t}/extensions" 2>/dev/null; exit 0 ;;
  --install-extension) case "$2" in *.vsix) touch "{t}/hue-installed" ;; *) echo "$2" >>"{t}/extensions" ;; esac ;;
esac
''')
    i.fake("defaults", DEFAULTS)
    # The running apps a sync reloads: none is running here
    i.fake("pgrep", "exit 1\n")
    i.fake("tmux", "exit 1\n")
    for app in ("aerospace", "brew"):
        i.fake(app, f'echo "$*" >>"{t}/{app}"\n')
    i.fake("sketchybar", f'echo "$*" >>"{t}/bar"\n')

    # The repo the sync pulls: this checkout with its working tree, committed
    i.git("clone", "-q", str(REPO), str(i.clone), cwd=t)
    subprocess.run(["rsync", "-a", "--exclude", ".git", f"{REPO}/", f"{i.clone}/"], check=True)
    i.git("add", "-A")
    i.git("commit", "-q", "--allow-empty", "-m", "working tree")
    i.git("clone", "-q", "--bare", str(i.clone), str(i.origin), cwd=t)
    i.git("remote", "set-url", "origin", str(i.origin))
    i.git("fetch", "-q")
    i.git("branch", "-q", "-u", f"origin/{i.git('branch', '--show-current')}")

    # An obsolete file, as an old install would have left it
    (i.home / ".config/cmux").mkdir(parents=True)
    (i.home / ".config/cmux/settings.json").write_text("old\n")

    i.first = i.sync()
    return i


# --- a first run: an empty HOME ---

def test_first_run_succeeds(install):
    assert install.first.returncode == 0, install.first.stderr


@pytest.mark.parametrize("tool", TOOLS)
def test_go_tools_built(install, tool):
    assert os.access(install.home / tool, os.X_OK)


def test_home_files_copied(install):
    files = listed(install.clone / ".config/dotfiles/home-files") + [".gitconfig-work", ".claude/settings.json"]
    assert [f for f in files if not (install.home / f).is_file()] == []


def test_icon_map_downloaded(install):
    assert "__icon_map()" in (install.home / ".config/sketchybar/plugins/icon_map.sh").read_text()


def test_hue_and_extensions_installed(install):
    assert (install.t / "hue-installed").exists()
    want = sorted(listed(install.clone / ".config/vscode/extensions.txt"))
    assert sorted((install.t / "extensions").read_text().split()) == want


def test_obsolete_file_removed(install):
    assert not (install.home / ".config/cmux/settings.json").exists()


@pytest.mark.parametrize("name", ["settings.json", "keybindings.json"])
def test_vscode_linked_into_the_clone(install, name):
    link = install.home / "Library/Application Support/Code/User" / name
    assert os.readlink(link) == str(install.clone / ".config/vscode" / name)


def test_lazygit_config_linked(install):
    link = install.home / "Library/Application Support/lazygit/config.yml"
    assert os.readlink(link) == str(install.home / ".config/lazygit/config.yml")


def test_launch_agent_installed(install):
    agent = install.home / LAUNCH_AGENT
    assert agent.read_bytes() == (install.clone / ".config/dotfiles/launchd" / agent.name).read_bytes()


def test_rex_settings_applied_to_the_fake_defaults(install):
    assert "Rex: " in install.first.stdout and (install.bin / "defaults.plist").exists()


def test_no_running_app_touched(install):
    """pgrep finds none, so no reload: AeroSpace, SketchyBar and borders untouched."""
    assert not (install.t / "aerospace").exists() and not (install.t / "brew").exists()


# --- a second run: nothing to do ---

def test_second_run_changes_nothing(install):
    out = install.sync()
    assert out.returncode == 0 and out.stdout == "", out.stdout


# --- the clone has a branch or uncommitted changes: said on stderr ---

def test_no_warning_on_a_clean_clone(install):
    out = install.sync()
    assert "installing branch" not in out.stderr and "uncommitted" not in out.stderr, out.stderr


def test_warns_about_uncommitted_changes(install):
    scratch = install.clone / "scratch.txt"
    scratch.write_text("scratch\n")
    try:
        assert "has uncommitted changes" in install.sync().stderr
    finally:
        scratch.unlink()


def test_warns_about_a_branch(install):
    upstream = install.git("rev-parse", "--abbrev-ref", "@{u}")
    install.git("switch", "-q", "-c", "topic", "--track", upstream)  # the pull needs an upstream
    try:
        assert "installing branch 'topic'" in install.sync().stderr
    finally:
        install.git("switch", "-q", "-")
        install.git("branch", "-q", "-D", "topic")


# --- Hue changed without a version bump: said on stderr ---

def test_hue_changed_without_a_bump(install):
    version = json.loads((REPO / ".config/vscode/extensions/hue/package.json").read_text())["version"]
    stamp = install.home / ".cache/hue.sha"
    stamp.write_text(f"an-older-sha {version}\n")
    out = install.sync()
    assert out.stdout.startswith(f"hue: installed {version}"), out.stdout
    assert f"hue: changed but still {version}" in out.stderr
    stamp.write_text("an-older-sha\n")  # a stamp from before it held the version
    assert "bump" not in install.sync().stderr


# --- VS Code linked to the ~/.config copy (an older install): moved into the clone ---

def test_vscode_links_moved_into_the_clone(install):
    user = install.home / "Library/Application Support/Code/User"
    for name in ("settings.json", "keybindings.json"):
        (user / name).unlink()
        (user / name).symlink_to(install.home / ".config/vscode" / name)
    out = install.sync().stdout
    assert f"linked VS Code settings.json to {install.clone}/.config/vscode" in out
    assert "linked VS Code keybindings.json" in out
    assert ".bak" not in out  # a link isn't kept as .bak
    # a setting VS Code writes lands in the clone, in its git status
    (user / "settings.json").write_text('{"from": "the VS Code UI"}\n')
    try:
        assert install.git("status", "--porcelain", "--", ".config/vscode/settings.json").startswith("M")
    finally:
        install.git("checkout", "-q", "--", ".config/vscode/settings.json")


# --- a .go file changes: only its tool is rebuilt ---

def test_only_the_changed_tool_is_rebuilt(install):
    (install.clone / ".config/sketchybar/plugins/check/main.go").touch()
    assert install.sync().stdout == "built check"


# --- the binaries work ---

@pytest.mark.parametrize("command,denied", [("cat .env", True), ("cat README.md", False)])
def test_claude_hook(install, command, denied):
    event = json.dumps({"tool_name": "Bash", "cwd": "/tmp", "tool_input": {"command": command}})
    out = subprocess.run([install.home / TOOLS[-1]], input=event, capture_output=True, text=True).stdout
    assert ('"deny"' in out) == denied, out


def test_sketchybar_check_toggles_its_popup(install):
    env = dict(install.env, NAME="checks", SENDER="mouse.clicked",
               CONFIG_DIR=str(install.home / ".config/sketchybar"))
    subprocess.run([install.home / ".config/sketchybar/plugins/check/check"], env=env, check=True)
    assert "popup.drawing=toggle" in (install.t / "bar").read_text()


def test_local_files_prints_its_usage(install):
    out = subprocess.run([install.home / ".config/dotfiles/local-files/local-files"], capture_output=True, text=True)
    assert (out.stdout + out.stderr).startswith("usage:")


def test_kube_guard_context(install):
    kube = install.home / ".kube"
    kube.mkdir(exist_ok=True)
    (kube / "config").write_text("current-context: prod-eu\n")
    out = subprocess.run([install.home / ".config/kube-guard/kube-guard", "context"],
                         env=install.env, capture_output=True, text=True)
    assert out.stdout == "prod-eu\n\nprod\n"


# --- dotfiles-diff: what a sync would change, changing nothing ---

def test_diff_matches_right_after_a_sync(install):
    shutil.rmtree(install.home / ".kube")  # left by the test above, not the sync's
    assert "This machine matches it." in install.diff()


def test_diff_lists_changes_and_changes_nothing(install):
    h = install.home
    with (h / ".zshrc").open("a") as f:
        f.write("# local edit\n")
    (h / ".config/sketchybar/colors.sh").unlink()
    (h / ".config/cmux").mkdir(exist_ok=True)
    (h / ".config/cmux/settings.json").write_text("old\n")
    before = install.state()
    out = install.diff()
    assert "changed  ~/.zshrc" in out
    assert "new      ~/.config/sketchybar/colors.sh" in out
    assert "removed  ~/.config/cmux/settings.json" in out
    out = install.diff("-p")
    assert "/.zshrc" in out and "-# local edit" in out  # - is this machine
    assert install.state() == before


def test_diff_compares_with_the_upstream_the_sync_pulls(install):
    other = install.t / "other"
    install.git("clone", "-q", str(install.origin), str(other), cwd=install.t)
    with (other / ".config/git/ignore").open("a") as f:
        f.write("# from upstream\n")
    install.git("commit", "-qam", "upstream", cwd=other)
    install.git("push", "-q", cwd=other)
    out = install.diff()
    assert "1 commit(s) ahead of the clone" in out and "changed  ~/.config/git/ignore" in out


def test_diff_compares_with_head_when_the_clone_is_ahead(install):
    install.git("pull", "-q", "--ff-only")  # back in step with the upstream
    with (install.clone / ".config/git/ignore").open("a") as f:
        f.write("# not pushed\n")
    install.git("commit", "-qam", "not pushed")
    out = install.diff()
    assert f"Compared with {install.git('branch', '--show-current')} " in out
    assert "changed  ~/.config/git/ignore" in out


# --- last: a broken build fails the sync ---

def test_a_broken_build_fails_the_sync(install):
    with (install.clone / ".config/dotfiles/local-files/main.go").open("a") as f:
        f.write("not go\n")
    assert install.sync().returncode != 0
