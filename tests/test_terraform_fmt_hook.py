"""git_hooks/terraform-fmt: the staged Terraform files, as staged, through
tofu fmt (terraform fmt without tofu), in a throwaway repo."""

import os
import shutil
import subprocess

import pytest

from conftest import REPO

HOOK = REPO / "git_hooks" / "terraform-fmt" / "pre-commit"
UNFORMATTED = 'resource "x" "y" {\na=1\n  bb = 2\n}\n'
FORMATTED = 'resource "x" "y" {\n  a  = 1\n  bb = 2\n}\n'

pytestmark = pytest.mark.skipif(
    not (shutil.which("tofu") or shutil.which("terraform")), reason="needs tofu or terraform"
)


@pytest.fixture
def repo(tmp_path):
    env = dict(os.environ, GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1")
    subprocess.run(["git", "init", "-q", str(tmp_path)], check=True, env=env)

    def git(*args):
        subprocess.run(["git", *args], cwd=tmp_path, check=True, env=env)

    def hook():
        return subprocess.run([str(HOOK)], cwd=tmp_path, capture_output=True, text=True, env=env)

    return tmp_path, git, hook


def write(root, name, text):
    (root / name).parent.mkdir(parents=True, exist_ok=True)
    (root / name).write_text(text)


def test_unformatted_file_stops_the_commit_with_its_diff(repo):
    root, git, hook = repo
    write(root, "mod/main.tf", UNFORMATTED)
    git("add", "mod/main.tf")
    out = hook()
    assert out.returncode == 1
    assert "+++ new/mod/main.tf" in out.stdout and "+  a  = 1" in out.stdout


def test_the_staged_version_is_checked_not_the_working_tree(repo):
    root, git, hook = repo
    write(root, "main.tf", UNFORMATTED)
    git("add", "main.tf")
    write(root, "main.tf", FORMATTED)  # fixed on disk, not staged
    assert hook().returncode == 1


def test_formatted_files_pass(repo):
    root, git, hook = repo
    write(root, "main.tf", FORMATTED)
    git("add", "main.tf")
    assert hook().returncode == 0


def test_tfvars_are_checked(repo):
    root, git, hook = repo
    write(root, "prod.tfvars", 'name  = "a"\n')
    git("add", "prod.tfvars")
    assert hook().returncode == 1


def test_invalid_hcl_stops_the_commit(repo):
    root, git, hook = repo
    write(root, "bad.tf", 'resource "x" { a = \n')
    git("add", "bad.tf")
    out = hook()
    assert out.returncode == 1 and "Error" in out.stdout


def test_nothing_terraform_staged_does_nothing(repo):
    root, git, hook = repo
    write(root, "README.md", "x\n")
    write(root, "main.tf", UNFORMATTED)  # not staged
    git("add", "README.md")
    out = hook()
    assert out.returncode == 0 and out.stdout == ""
