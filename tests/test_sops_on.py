"""sops.zsh at login (_sops_login, run by the LaunchAgent): the keys sops-on left
when the session ended are removed, also from a SOPS_KEYS_FILE only
~/.zshrc.local set; a keys file sops-on didn't create stays. The keys file holds
placeholder text: only whether it is removed matters here."""

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
