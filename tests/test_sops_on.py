"""sops.zsh at login (_sops_login, run by the LaunchAgent): the keys sops-on left
when the session ended are removed, also from a SOPS_KEYS_FILE only
~/.zshrc.local set; a keys file sops-on didn't create stays. sops-login turns
that LaunchAgent on or off (a fake launchctl keeps the state in a file). The
keys file holds placeholder text: only whether it is removed matters here."""

import pytest

from conftest import CONFIG, zsh

SOPS = CONFIG / "zsh" / "sops.zsh"
KEYS = "placeholder keys\n"


def login(home):
    """_sops_login as launchd runs it: a bare zsh, HOME and nothing of ~/.zshrc.local."""
    return zsh(f'source "{SOPS}" && _sops_login', {"HOME": str(home), "PATH": "/usr/bin:/bin"})


def sops_on_state(home, keys_file, expiry="9999999999"):
    """What sops-on leaves: the deadline, the keys file and its path."""
    state = home / ".cache" / "sops-on"
    state.mkdir(parents=True)
    (state / "expiry").write_text(expiry + "\n")
    (state / "file").write_text(f"{keys_file}\n")
    keys_file.parent.mkdir(parents=True, exist_ok=True)
    keys_file.write_text(KEYS)
    return state


def test_keys_with_time_left_are_removed_at_login(tmp_path):
    keys = tmp_path / "Library" / "Application Support" / "sops" / "age" / "keys.txt"
    state = sops_on_state(tmp_path, keys)
    assert login(tmp_path).returncode == 0
    assert not keys.exists()
    assert not (state / "expiry").exists() and not (state / "file").exists()
    assert "session ended" in (state / "notice").read_text()  # said at the next prompt


def test_the_keys_file_sops_on_wrote_not_the_default(tmp_path):
    """SOPS_KEYS_FILE set in ~/.zshrc.local: launchd never sees it, the state does."""
    keys = tmp_path / "elsewhere" / "keys.txt"
    sops_on_state(tmp_path, keys)
    login(tmp_path)
    assert not keys.exists()


def test_a_keys_file_sops_on_did_not_create_stays(tmp_path):
    keys = tmp_path / "Library" / "Application Support" / "sops" / "age" / "keys.txt"
    keys.parent.mkdir(parents=True)
    keys.write_text(KEYS)
    assert login(tmp_path).returncode == 0
    assert keys.read_text() == KEYS


# A fake launchctl: enable / disable write the state, print-disabled prints it as macOS does
LAUNCHCTL = r"""
state="$HOME/launchctl-state"
case "$1" in
  disable) echo "\"${2##*/}\" => disabled" >"$state" ;;
  enable) echo "\"${2##*/}\" => enabled" >"$state" ;;
  print-disabled) echo "disabled services = {"; cat "$state" 2>/dev/null; echo "}" ;;
esac
"""


@pytest.fixture
def launchctl(stubs, tmp_path):
    stubs.add("launchctl", LAUNCHCTL)

    def run(*commands):
        script = f'source "{SOPS}"' + "".join(f" && {c}" for c in commands)
        return zsh(script, stubs.env(HOME=str(tmp_path)))
    return run


def test_sops_login_off_and_on(launchctl, tmp_path):
    plist = tmp_path / "Library" / "LaunchAgents" / "com.user-cube.sops-on-login.plist"
    assert "not installed" in launchctl("sops-login").stdout
    plist.parent.mkdir(parents=True)
    plist.write_text("<plist/>")
    assert "sops-login: on" in launchctl("sops-login").stdout

    out = launchctl("sops-login off", "sops-login")
    assert out.returncode == 0
    assert out.stdout.splitlines()[-1].startswith("sops-login: off on this machine")
    assert '"com.user-cube.sops-on-login" => disabled' in (tmp_path / "launchctl-state").read_text()

    out = launchctl("sops-login on", "sops-login")
    assert out.stdout.splitlines()[-1].startswith("sops-login: on")


def test_sops_login_usage(launchctl):
    out = launchctl("sops-login maybe")
    assert out.returncode == 1 and "usage: sops-login [on|off]" in out.stderr
