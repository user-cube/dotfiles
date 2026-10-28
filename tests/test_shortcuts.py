"""The ⌘ shortcuts live in three places that must agree: Ghostty's config (the
key, sending the tmux prefix C-b plus a key), tmux.conf (what that key does)
and the ⌘⇧H cheat sheet (the list, which also runs the picked one)."""

import re
import subprocess
import uuid

import pytest

from conftest import CONFIG

GHOSTTY = CONFIG / "ghostty" / "config"
TMUX_CONF = CONFIG / "tmux" / "tmux.conf"
CHEATSHEET = CONFIG / "tmux" / "cheatsheet.sh"

# Ghostty keys with no cheat sheet line of their own: ⌘⌥ arrows are listed as
# one line (C-b q), and ⌘⇧H is the cheat sheet itself
NOT_IN_CHEATSHEET = {"h", "j", "k", "l", "?"}


def ghostty_keybinds() -> dict:
    """Ghostty trigger (cmd+shift+c) -> the tmux key it sends after C-b."""
    binds = {}
    for line in GHOSTTY.read_text().splitlines():
        m = re.match(r"keybind = (\S+?)=text:\\x02(.+)$", line.strip())
        if m:
            binds[m.group(1)] = m.group(2)
    return binds


def tmux_conf_without_plugins(tmp_dir) -> str:
    """tmux.conf without its plugin lines (the playbook clones the plugins, not
    the repo), as a file: tmux doesn't read a config from /dev/stdin."""
    conf = tmp_dir / "tmux.conf"
    conf.write_text("\n".join(
        line for line in TMUX_CONF.read_text().splitlines() if not re.match(r"\s*run(-shell)?\s", line)
    ))
    return str(conf)


def plugin_keys() -> set:
    """The prefix keys the plugins bind, from the options tmux.conf sets for
    them (tmux-fingers: @fingers-key, tmux-fzf-url: @fzf-url-bind)."""
    return set(re.findall(r"set -g @(?:fingers-key|fzf-url-bind) (\S+)", TMUX_CONF.read_text()))


def tmux_keys(table: str, tmp_dir) -> set:
    """The keys bound in a tmux key table once tmux.conf is loaded (tmux's own
    defaults included), in a throwaway server; for prefix, also the plugins'."""
    socket = f"pytest-{uuid.uuid4().hex[:8]}"
    out = subprocess.run(
        ["tmux", "-L", socket, "-f", tmux_conf_without_plugins(tmp_dir),
         "start-server", ";", "list-keys", "-T", table, ";", "kill-server"],
        capture_output=True, text=True, timeout=30,
    )
    keys = set()
    for line in out.stdout.splitlines():
        tokens = line.split()
        key = tokens[tokens.index(table) + 1]
        keys.add(key[1:] if len(key) > 1 and key.startswith("\\") else key)
    return keys | plugin_keys() if table == "prefix" else keys


def cheatsheet_entries() -> list:
    """(tmux keys, ⌘ shortcut) of each cheat sheet line."""
    body = CHEATSHEET.read_text().split("entries=(", 1)[1].split("\n)", 1)[0]
    entries = []
    for line in body.splitlines():
        line = line.strip()
        if line.startswith("'"):
            keys, _section, shortcut, *_ = line.strip("'").split("\t")
            entries.append((keys, shortcut))
    return entries


def to_ghostty_trigger(shortcut: str):
    """⌘⇧C -> cmd+shift+c; None for the ones that stand for several keys (⌘1-9)."""
    mods = {"⌘": "cmd", "⌥": "alt", "⇧": "shift"}
    names = {"←": "left", "→": "right", "↑": "up", "↓": "down"}
    parts, rest = [], shortcut
    while rest and rest[0] in mods:
        parts.append(mods[rest[0]])
        rest = rest[1:]
    if not parts or len(rest) != 1:
        return None
    return "+".join(parts + [names.get(rest, rest.lower())])


def test_ghostty_keys_are_bound_in_tmux(tmp_path):
    bound = tmux_keys("prefix", tmp_path)
    missing = {t: k for t, k in ghostty_keybinds().items() if k not in bound}
    assert not missing, f"Ghostty sends C-b + a key tmux doesn't bind: {missing}"


def test_ghostty_keys_are_in_the_cheatsheet():
    listed = {keys.removeprefix("C-b ") for keys, _ in cheatsheet_entries()}
    missing = {
        t: k for t, k in ghostty_keybinds().items()
        if not k.isdigit() and k not in NOT_IN_CHEATSHEET and k not in listed
    }
    assert not missing, f"⌘ shortcuts missing from the ⌘⇧H cheat sheet: {missing}"


def test_cheatsheet_keys_exist(tmp_path):
    prefix, root = tmux_keys("prefix", tmp_path), tmux_keys("root", tmp_path)
    for keys, _ in cheatsheet_entries():
        if keys.startswith("C-b "):
            assert keys[4:] in prefix, f"cheat sheet line for {keys}, which tmux doesn't bind"
        else:
            assert keys in root, f"cheat sheet line for {keys}, which tmux doesn't bind"


@pytest.mark.parametrize("keys,shortcut", [e for e in cheatsheet_entries() if to_ghostty_trigger(e[1])])
def test_cheatsheet_shortcut_matches_ghostty(keys, shortcut):
    trigger = to_ghostty_trigger(shortcut)
    assert ghostty_keybinds().get(trigger) == keys.removeprefix("C-b "), (
        f"the cheat sheet says {shortcut} is {keys}; Ghostty's {trigger} sends "
        f"{ghostty_keybinds().get(trigger)!r}"
    )


VSCODE_KEYBINDINGS = CONFIG / "vscode" / "keybindings.json"


def test_vscode_keys_match_ghostty():
    """VS Code's terminal runs tmux too (vscode/tmux.sh): its ⌘ shortcuts send
    the same C-b + key as Ghostty's, never a different one."""
    import json
    ghostty = ghostty_keybinds()
    for b in json.loads(VSCODE_KEYBINDINGS.read_text()):
        if b["command"] != "workbench.action.terminal.sendSequence" or not b["args"]["text"].startswith("\x02"):
            continue
        assert b.get("when") == "terminalFocus", f"{b['key']} would also fire outside the terminal"
        assert ghostty.get(b["key"]) == b["args"]["text"].removeprefix("\x02"), (
            f"VS Code's {b['key']} sends {b['args']['text']!r}; Ghostty's sends {ghostty.get(b['key'])!r}"
        )


def test_vscode_shift_enter_is_a_newline():
    """VS Code's terminal sends Shift+Enter as a plain Enter. As ESC + Enter
    (Alt+Enter) it reaches the app through tmux, and Claude Code takes it as
    a newline, as it takes Ghostty's extended Shift+Enter."""
    import json
    binds = {b["key"]: b for b in json.loads(VSCODE_KEYBINDINGS.read_text())}
    b = binds.get("shift+enter")
    assert b and b["args"]["text"] == "\x1b\r" and b["when"] == "terminalFocus", b


def test_vscode_cmd_k_starts_chords_in_the_terminal():
    """VS Code binds ⌘K to Terminal: Clear, which ate the chords that start with
    it (⌘K H Hue: Search, ⌘K ⌘S Go to Folder) while the terminal had focus; in
    tmux that clear did nothing anyway (C-l or `clear` there). Taken out, and
    nothing else takes a bare ⌘K."""
    import json
    binds = json.loads(VSCODE_KEYBINDINGS.read_text())
    assert any(b["key"] == "cmd+k" and b["command"] == "-workbench.action.terminal.clear" and "when" not in b for b in binds)
    assert not [b for b in binds if b["key"] == "cmd+k" and not b["command"].startswith("-")]
