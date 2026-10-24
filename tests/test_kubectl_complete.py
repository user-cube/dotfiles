"""The kubectl_complete-<plugin> scripts (Tab completion for the krew
plugins): kubectl runs them with the words after the plugin name and reads one
completion per line, then :<directive>. A fake kubectl prints the
`kubectl __complete` call the scripts hand over to."""

import subprocess

import pytest

from conftest import CONFIG

COMPLETE = CONFIG / "kubectl" / "complete"


@pytest.fixture
def env(stubs, tmp_path):
    stubs.add("kubectl", 'echo "kubectl $*"\n')
    krew = tmp_path / "krew"
    for folder, plugins in (("index/default/plugins", ["neat", "tree", "ctx"]), ("receipts", ["neat"])):
        (krew / folder).mkdir(parents=True)
        for plugin in plugins:
            (krew / folder / f"{plugin}.yaml").write_text("")
    return stubs.env(KREW_ROOT=str(krew))


def complete(env, plugin, *words):
    out = subprocess.run([str(COMPLETE / f"kubectl_complete-{plugin}"), *words],
                         env=env, capture_output=True, text=True, check=True)
    return out.stdout.splitlines()


def test_krew_commands(env):
    lines = complete(env, "krew", "")
    assert "install" in lines and "uninstall" in lines and lines[-1] == ":4"


def test_krew_install_lists_the_index(env):
    assert complete(env, "krew", "install", "") == ["ctx", "neat", "tree", ":4"]


def test_krew_uninstall_lists_the_installed(env):
    assert complete(env, "krew", "uninstall", "") == ["neat", ":4"]


@pytest.mark.parametrize("plugin,words,handed_over", [
    ("tree", ["deploy", ""], "kubectl __complete get deploy "),
    ("view_secret", ["-n", "api", ""], "kubectl __complete get secret -n api "),
    ("who_can", ["delete", "pods", ""], "kubectl __complete get pods "),
    ("neat", ["get", "--", "pod", ""], "kubectl __complete get pod "),
])
def test_plugins_hand_over_to_kubectl_get(env, plugin, words, handed_over):
    assert complete(env, plugin, *words) == [handed_over]


def test_who_can_starts_with_a_verb(env):
    lines = complete(env, "who_can", "")
    assert "delete" in lines and lines[-1] == ":4"
