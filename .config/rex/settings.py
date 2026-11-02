#!/usr/bin/env python3
"""Rex's Settings (theme, tabs, panes, cursor) as code: init.lua has no theme
or font, Rex keeps them in the com.superlogical.rex defaults (Workspace.*).

  settings.py export > settings.json   the app's current ones, after a change in Settings
  settings.py apply [settings.json]    write the ones that differ (install.sh)

Workspace.* keys only, less the app's own state (window, last session, what
was seen). Values Rex stores as JSON data (the custom themes) are kept as
JSON, so a diff shows the colors. Undocumented keys of a beta: a key Rex
renames is simply ignored by it. Prints a line when it changed something.
"""
import json
import os
import plistlib
import subprocess
import sys

DOMAIN = "com.superlogical.rex"
STATE = {"Workspace.lastFocusedSession", "Workspace.programActivitySeen"}


def current():
    # `defaults export` and not the file: cfprefsd may not have written it yet
    r = subprocess.run(["defaults", "export", DOMAIN, "-"], capture_output=True, check=True)
    return plistlib.loads(r.stdout)


def to_json(value):
    if isinstance(value, bytes):
        return json.loads(value)
    return value


def export():
    prefs = current()
    out = {k: to_json(v) for k, v in sorted(prefs.items())
           if k.startswith("Workspace.") and k not in STATE}
    json.dump(out, sys.stdout, indent=2, ensure_ascii=False)
    sys.stdout.write("\n")


def write(key, value):
    if isinstance(value, bool):
        args = ["-bool", "true" if value else "false"]
    elif isinstance(value, int):
        args = ["-int", str(value)]
    elif isinstance(value, float):
        args = ["-float", repr(value)]
    elif isinstance(value, str):
        args = ["-string", value]
    else:  # list or dict: JSON data, as Rex stores it
        args = ["-data", json.dumps(value, separators=(",", ":")).encode().hex()]
    subprocess.run(["defaults", "write", DOMAIN, key, *args], check=True)


def apply(path):
    with open(path) as f:
        wanted = json.load(f)
    prefs = current()
    changed = [k for k, v in wanted.items() if to_json(prefs.get(k)) != v]
    for k in changed:
        write(k, wanted[k])
    if changed:
        print(f"Rex: {len(changed)} setting(s) applied, ⌘Q Rex to load them")


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else ""
    if cmd == "export":
        export()
    elif cmd == "apply":
        here = os.path.dirname(os.path.abspath(__file__))
        apply(sys.argv[2] if len(sys.argv) > 2 else os.path.join(here, "settings.json"))
    else:
        sys.exit(__doc__)
