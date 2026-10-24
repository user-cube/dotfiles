"""k9s plugins and hotkeys: keys k9s can actually use, and the plugins that
change the cluster switched off in read-only (production)."""

import re
from collections import defaultdict

import pytest
import yaml

from conftest import CONFIG

K9S = CONFIG / "k9s"
PLUGINS = yaml.safe_load((K9S / "plugins.yaml").read_text())["plugins"]
HOTKEYS = yaml.safe_load((K9S / "hotkeys.yaml").read_text())["hotKeys"]

# A terminal already sends these as something else (Backspace, Tab, Enter): k9s
# rejects the key and then loads no plugin at all. Ctrl-P never reaches a plugin.
FORBIDDEN = {"Ctrl-H", "Ctrl-I", "Ctrl-J", "Ctrl-M", "Ctrl-P"}
# Shift + digit types the same character on the Portuguese and US layouts only
# for these; k9s reads Shift-2 as @, which a Portuguese keyboard doesn't send
SAFE_SHIFT_DIGITS = {"Shift-1", "Shift-3", "Shift-4", "Shift-5"}
# What a plugin runs that changes the cluster
WRITES = re.compile(r"\b(debug|annotate|patch|apply|delete|edit|scale|sync|rollout)\b")


@pytest.mark.parametrize("name", PLUGINS)
def test_plugin_key_is_usable(name):
    assert PLUGINS[name]["shortCut"] not in FORBIDDEN


def test_plugin_keys_are_unique_per_view():
    by_view = defaultdict(list)
    for name, plugin in PLUGINS.items():
        for scope in plugin["scopes"]:
            by_view[(scope, plugin["shortCut"])].append(name)
    clashes = {key: names for key, names in by_view.items() if len(names) > 1}
    assert not clashes


@pytest.mark.parametrize("name", PLUGINS)
def test_plugin_that_changes_the_cluster_is_dangerous(name):
    plugin = PLUGINS[name]
    runs = " ".join([plugin["command"], *map(str, plugin.get("args", []))])
    if WRITES.search(runs):
        assert plugin.get("dangerous") is True, f"{name} changes the cluster: dangerous: true hides it in read-only"


@pytest.mark.parametrize("name", HOTKEYS)
def test_hotkey_works_on_a_portuguese_keyboard(name):
    assert HOTKEYS[name]["shortCut"] in SAFE_SHIFT_DIGITS


def test_skin_exists():
    config = yaml.safe_load((K9S / "config.yaml").read_text())
    assert (K9S / "skins" / f"{config['k9s']['ui']['skin']}.yaml").is_file()
