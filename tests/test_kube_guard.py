"""kube-guard.zsh: on a production context kubectl / kubecolor / helm ask
before changing the cluster, and k9s and sofka open read-only. Fake commands stand in
for the real ones: they print what they were asked to run, and kubectl's
`config current-context` answers $FAKE_CTX. The prompt reads the terminal,
so pexpect runs zsh in one and answers it."""

import subprocess

import pexpect
import pytest

from conftest import CONFIG, zsh

GUARD = CONFIG / "zsh" / "kube-guard.zsh"
FAKE = r"""
if [ "$1 $2" = "config current-context" ]; then echo "$FAKE_CTX"; exit 0; fi
if [ ! -t 0 ]; then echo "RAN {name} $* stdin=$(cat)"; else echo "RAN {name} $*"; fi
"""


@pytest.fixture
def fakes(stubs):
    for name in ("kubectl", "kubecolor", "helm", "k9s", "sofka"):
        stubs.add(name, FAKE.replace("{name}", name))
    return stubs


def run(fakes, ctx: str, command: str, answer=None):
    """Run command in zsh with kube-guard sourced; returns (asked?, output)."""
    child = pexpect.spawn(
        "zsh", ["-f", "-c", f"source {GUARD}; {command}"],
        env=fakes.env(FAKE_CTX=ctx), encoding="utf-8", timeout=10,
    )
    asked = child.expect([r"\? \[y/N\] ", pexpect.EOF]) == 0
    if asked:
        child.send(answer)
        child.expect(pexpect.EOF)
    child.close()
    return asked, child.before or ""


# (current context, command, asks?, runs?) answering n
CASES = [
    ("dev-cluster", "kubectl delete pod x", False, True),
    ("prod-eu", "kubectl get pods -A", False, True),
    ("prod-eu", "kubectl -n api delete pod x", True, False),
    ("PROD-EU", "kubectl delete pod x", True, False),  # any case, like grep -i
    ("dev", "kubectl --context PRD-us -n a apply -f x.yaml", True, False),
    ("prod-eu", "kubectl --context dev delete pod x", False, True),
    ("prod-eu", "kubectl rollout status deploy/api", False, True),
    ("prod-eu", "kubectl rollout restart deploy/api", True, False),
    ("prod-eu", "helm template ./chart", False, True),
    ("prod-eu", "helm --kube-context dev upgrade api ./chart", False, True),
    ("dev", "helm upgrade api ./chart --kube-context=prod-eu", True, False),
    ("prod-eu", "kubecolor delete ns test", True, False),
    ("prod-eu", "KUBE_GUARD=0 kubectl delete pod x", False, True),
]


@pytest.mark.parametrize("ctx,command,asks,runs", CASES)
def test_guard_answering_no(fakes, ctx, command, asks, runs):
    asked, out = run(fakes, ctx, command, answer="n")
    assert asked == asks
    assert ("RAN " in out) == runs, out


def test_yes_runs_it(fakes):
    asked, out = run(fakes, "prod-eu", "kubectl delete pod x", answer="y")
    assert asked and "RAN kubectl delete pod x" in out


def test_a_piped_manifest_reaches_kubectl(fakes):
    """The answer comes from the terminal, so stdin (the manifest) is left for kubectl."""
    asked, out = run(fakes, "prod-eu", "echo manifest-content | kubectl apply -f -", answer="y")
    assert asked and "stdin=manifest-content" in out


@pytest.mark.parametrize("ctx,command,expected", [
    ("prod-eu", "k9s", "RAN k9s --readonly"),
    ("prod-eu", "k9s --write", "RAN k9s --write"),
    ("dev", "k9s --context prod-us", "RAN k9s --readonly --context prod-us"),
    ("dev", "k9s", "RAN k9s"),
])
def test_k9s_is_read_only_on_production(fakes, ctx, command, expected):
    _, out = run(fakes, ctx, command)
    ran = next(line for line in out.splitlines() if line.startswith("RAN"))
    assert ran.strip() == expected


@pytest.mark.parametrize("ctx,command,expected", [
    ("prod-eu", "sofka", "RAN sofka --readonly"),
    ("prod-eu", "sofka --write", "RAN sofka --write"),
    ("dev", "sofka --context prod-us", "RAN sofka --readonly --context prod-us"),
    ("dev", "sofka", "RAN sofka"),
    # subcommands don't open the TUI
    ("prod-eu", "sofka info --offline", "RAN sofka info --offline"),
])
def test_sofka_is_read_only_on_production(fakes, ctx, command, expected):
    _, out = run(fakes, ctx, command)
    ran = next(line for line in out.splitlines() if line.startswith("RAN"))
    assert ran.strip() == expected


# ⌘K follows the pane: each interactive shell writes its KUBECONFIG to a file
# per pane (a `ktx -s` shell has one of its own), and puts back its parent's
# when it exits; the popup (_kube_popup) reads it.
def test_a_shell_writes_its_kubeconfig_for_the_pane_and_restores_the_parents(stubs, tmp_path):
    pane = tmp_path / "kube-pane" / "tmux-%7"
    pane.parent.mkdir()
    pane.write_text("/parent.yaml\n")
    # exit: zsh -c would exec its last command and never run the exit hook
    script = f"source {GUARD}; _kube_pane_publish; cat {pane}; exit"
    out = zsh_i(script, stubs.env(KUBE_PANE_DIR=str(pane.parent), TMUX_PANE="%7", KUBECONFIG="/child.yaml"))
    assert out == "/child.yaml"
    assert pane.read_text() == "/parent.yaml\n"


@pytest.mark.parametrize("exists", [True, False])
def test_the_popup_opens_on_the_panes_kubeconfig(fakes, tmp_path, exists):
    kubeconfig = tmp_path / "kubectx-shell.yaml"
    if exists:
        kubeconfig.write_text("")
    pane = tmp_path / "kube-pane" / "rex-block:1"
    pane.parent.mkdir()
    pane.write_text(f"{kubeconfig}\n")
    fakes.add("sofka", 'echo "RAN sofka KUBECONFIG=${KUBECONFIG-unset}"')
    env = fakes.env(KUBE_PANE_DIR=str(pane.parent), FAKE_CTX="dev")
    env.pop("KUBECONFIG", None)
    out = zsh(f"source {GUARD}; _kube_popup sofka rex-block:1", env).stdout.strip()
    # a file naming a kubeconfig that is gone (its shell didn't exit cleanly): the default
    assert out == f"RAN sofka KUBECONFIG={kubeconfig if exists else 'unset'}"


def zsh_i(script: str, env: dict) -> str:
    """An interactive zsh (no .zshrc), as kube-guard only writes the pane file there."""
    return subprocess.run(["zsh", "-f", "-i", "-c", script], env=env, capture_output=True, text=True, timeout=30).stdout.strip()
