"""What the tests share: the repo's paths, and fake commands (kubectl, helm,
git...) put first on PATH so a test never reaches a real cluster or network.

Run them: python3 -m venv .venv && .venv/bin/pip install -r tests/requirements.txt
          .venv/bin/pytest tests/
"""

import os
import stat
import subprocess
from pathlib import Path

import pytest

REPO = Path(__file__).resolve().parents[1]
CONFIG = REPO / ".config"


class Stubs:
    """A bin folder of fake commands, first on PATH in env()."""

    def __init__(self, root: Path):
        self.bin = root / "bin"
        self.bin.mkdir()

    def add(self, name: str, script: str) -> Path:
        path = self.bin / name
        path.write_text("#!/bin/sh\n" + script)
        path.chmod(path.stat().st_mode | stat.S_IXUSR)
        return path

    def env(self, **extra: str) -> dict:
        env = dict(os.environ, PATH=f"{self.bin}:{os.environ['PATH']}")
        env.update(extra)
        return env


@pytest.fixture
def stubs(tmp_path) -> Stubs:
    return Stubs(tmp_path)


def zsh(script: str, env: dict, cwd=None) -> subprocess.CompletedProcess:
    """Run a script in a bare zsh (no .zshrc): the file under test is sourced by it."""
    return subprocess.run(
        ["zsh", "-f", "-c", script], env=env, cwd=cwd, capture_output=True, text=True, timeout=60
    )
