"""plugins-outdated.zsh: the plugins the playbook pins to a commit that master
has moved past, and the background check every 14 days. A sample defaults
file stands in for the playbook, a fake git for `git ls-remote`, a fake
osascript for the notification."""

import os
import time

import pytest

from conftest import CONFIG, zsh

PLUGINS_OUTDATED = CONFIG / "zsh" / "plugins-outdated.zsh"
SAME = "a" * 40
OLD, NEW = "b" * 40, "c" * 40
DEFAULTS = f"""---
zsh_theme_repo: "https://github.com/romkatv/powerlevel10k.git"
zsh_plugins:
  - name: up-to-date # a comment
    repo: https://github.com/x/up-to-date
    version: {SAME}
  - name: behind
    repo: https://github.com/x/behind.git
    version: {OLD}
"""


@pytest.fixture
def setup(stubs, tmp_path):
    playbook = tmp_path / "playbook" / "roles" / "terminal" / "defaults"
    playbook.mkdir(parents=True)
    (playbook / "main.yml").write_text(DEFAULTS)
    # master of each fake repo: up-to-date's is its pin, behind's has moved
    stubs.add("git", f"""case "$2" in
  *up-to-date*) echo "{SAME}	refs/heads/master" ;;
  *behind*) echo "{NEW}	refs/heads/master" ;;
esac
""")
    notified = tmp_path / "notified"
    stubs.add("osascript", f'echo "$*" >> "{notified}"\n')
    home = tmp_path / "home"
    home.mkdir()
    env = stubs.env(HOME=str(home), DAILY_DRIVER_DIR=str(tmp_path / "playbook"))
    return env, home, notified


def test_pins_are_read_from_the_playbook(setup):
    env, _, _ = setup
    out = zsh(f"source {PLUGINS_OUTDATED}; _plugins_pins", env)
    assert out.stdout.splitlines() == [
        f"up-to-date https://github.com/x/up-to-date {SAME}",
        f"behind https://github.com/x/behind.git {OLD}",
    ]


def test_lists_only_the_plugins_behind(setup):
    env, _, _ = setup
    out = zsh(f"source {PLUGINS_OUTDATED}; plugins-outdated", env)
    assert out.returncode == 1
    assert f"behind                   {OLD[:7]} -> {NEW[:7]}" in out.stdout
    assert "up-to-date" not in out.stdout


def test_without_the_playbook_says_so(setup):
    env, _, _ = setup
    out = zsh(f"source {PLUGINS_OUTDATED}; plugins-outdated", dict(env, DAILY_DRIVER_DIR="/nonexistent"))
    assert out.returncode == 2 and "no playbook" in out.stdout


def wait_for(path, seconds=5):
    for _ in range(seconds * 10):
        if path.exists():
            return True
        time.sleep(0.1)
    return False


def test_first_shell_notifies_then_waits_14_days(setup):
    env, home, notified = setup
    stamp = home / ".cache" / "zsh" / "plugins-outdated.stamp"

    zsh(f"source {PLUGINS_OUTDATED}; sleep 2", env)  # the check runs in the background
    assert wait_for(notified), "no notification"
    assert "1 behind master" in notified.read_text()

    notified.unlink()
    zsh(f"source {PLUGINS_OUTDATED}; sleep 2", env)  # checked a moment ago
    assert not notified.exists()

    old = time.time() - 15 * 86400
    os.utime(stamp, (old, old))  # 15 days later
    zsh(f"source {PLUGINS_OUTDATED}; sleep 2", env)
    assert wait_for(notified), "no notification after 15 days"
