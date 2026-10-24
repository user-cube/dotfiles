"""Every config file parses, so a typo fails here and not in the app (which
often just ignores the file, or the whole config)."""

import json
import subprocess
import tomllib
import uuid

import pytest
import yaml

from conftest import CONFIG, REPO
from test_shortcuts import tmux_conf_without_plugins


def tracked(pattern: str) -> list:
    out = subprocess.run(["git", "ls-files", pattern], cwd=REPO, capture_output=True, text=True, check=True)
    return out.stdout.split()


@pytest.mark.parametrize("path", tracked("*.toml"))
def test_toml_parses(path):
    tomllib.loads((REPO / path).read_text())


@pytest.mark.parametrize("path", tracked("*.yaml") + tracked("*.yml"))
def test_yaml_parses(path):
    list(yaml.safe_load_all((REPO / path).read_text()))


@pytest.mark.parametrize("path", tracked("*.json"))
def test_json_parses(path):
    json.loads((REPO / path).read_text())


def test_tmux_conf_loads_without_errors(tmp_path):
    """tmux keeps going after a bad line and only says so on stderr."""
    socket = f"pytest-{uuid.uuid4().hex[:8]}"
    out = subprocess.run(
        ["tmux", "-L", socket, "-f", tmux_conf_without_plugins(tmp_path), "start-server", ";", "kill-server"],
        capture_output=True, text=True, timeout=30,
    )
    assert out.returncode == 0 and not out.stderr.strip(), out.stderr


def test_bat_config_names_a_real_theme():
    themes = subprocess.run(["bat", "--list-themes", "--color=never"], capture_output=True, text=True).stdout
    config = (CONFIG / "bat" / "config").read_text()
    theme = config.split('--theme="', 1)[1].split('"', 1)[0]
    assert theme in themes.splitlines()
