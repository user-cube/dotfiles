"""sofka: the config loads without warnings, plugin keys aren't one of sofka's
own table keys, and the plugins that change the cluster say so (hidden in
read-only, on production)."""

import shutil
import subprocess
from collections import defaultdict

import pytest
import yaml

from conftest import CONFIG

CONF = yaml.safe_load((CONFIG / "sofka" / "conf.d" / "00-k9s.yaml").read_text())
PLUGINS = {p["name"]: p for p in CONF.get("plugins", [])}

# sofka's own keys in a table (docs/keys.md): a plugin there is shadowed when
# the action is available (ctrl-f is page down), and `sofka info` doesn't warn
BUILT_IN = {"ctrl-b", "ctrl-c", "ctrl-d", "ctrl-e", "ctrl-f", "ctrl-k", "ctrl-r", "ctrl-s", "ctrl-space", "ctrl-z"}


@pytest.mark.parametrize("name", PLUGINS)
def test_plugin_key_is_not_a_sofka_key(name):
    assert PLUGINS[name]["key"] not in BUILT_IN


def test_plugin_keys_are_unique_per_view():
    by_view = defaultdict(list)
    for name, plugin in PLUGINS.items():
        for scope in plugin.get("scopes", ["all"]):
            by_view[(scope, plugin["key"])].append(name)
    assert not {key: names for key, names in by_view.items() if len(names) > 1}


@pytest.mark.parametrize("name", PLUGINS)
def test_plugin_says_whether_it_changes_the_cluster(name):
    assert isinstance(PLUGINS[name].get("mutating"), bool), "mutating: true hides it in read-only"


@pytest.mark.skipif(not shutil.which("sofka"), reason="sofka not installed")
def test_config_loads_without_warnings(tmp_path):
    # the repo's .config as sofka's, offline: no cluster reached
    out = subprocess.run(
        [shutil.which("sofka"), "info", "--offline"],
        env={"HOME": str(tmp_path), "PATH": "/usr/bin:/bin", "XDG_CONFIG_HOME": str(CONFIG)},
        capture_output=True, text=True, timeout=30,
    ).stdout
    assert f"{CONFIG}/sofka/conf.d/00-k9s.yaml (loaded)" in out
    assert "No config validation warnings." in out, out
