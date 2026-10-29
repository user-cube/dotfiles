"""git_hooks/shellcheck, git_hooks/hadolint and git_hooks/sops-guard: the staged
files, as staged, in a throwaway repo."""

import os
import shutil
import subprocess

import pytest

from conftest import REPO

HOOKS = REPO / "git_hooks"

needs_shellcheck = pytest.mark.skipif(not shutil.which("shellcheck"), reason="needs shellcheck")
needs_hadolint = pytest.mark.skipif(not shutil.which("hadolint"), reason="needs hadolint")

BAD_SH = '#!/bin/bash\nrm -rf "$dir"/ ; cd $1\n'  # SC2164: cd without || exit
GOOD_SH = '#!/bin/bash\ncd "$1" || exit\n'
BAD_DOCKERFILE = "FROM debian\nRUN cd /tmp && echo hi\n"  # DL3006 untagged, DL3003 cd
GOOD_DOCKERFILE = "FROM debian:12\nWORKDIR /tmp\n"
ENCRYPTED_YAML = (
    "password: ENC[AES256_GCM,data:abc,iv:x,tag:y,type:str]\n"
    "sops:\n    mac: ENC[AES256_GCM,data:def,iv:x,tag:y,type:str]\n    version: 3.9.0\n"
)


@pytest.fixture
def repo(tmp_path):
    env = dict(os.environ, GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1")
    subprocess.run(["git", "init", "-q", str(tmp_path)], check=True, env=env)

    def stage(name, text):
        (tmp_path / name).parent.mkdir(parents=True, exist_ok=True)
        (tmp_path / name).write_text(text)
        subprocess.run(["git", "add", name], cwd=tmp_path, check=True, env=env)

    def hook(name):
        return subprocess.run(
            [str(HOOKS / name / "pre-commit")], cwd=tmp_path, capture_output=True, text=True, env=env
        )

    return tmp_path, stage, hook


@needs_shellcheck
def test_shellcheck_warning_stops_the_commit_with_the_file_name(repo):
    _, stage, hook = repo
    stage("bin/deploy.sh", BAD_SH)
    out = hook("shellcheck")
    assert out.returncode == 1
    assert "In bin/deploy.sh line 2" in out.stdout and "SC2164" in out.stdout


@needs_shellcheck
def test_shellcheck_finds_scripts_by_shebang_and_checks_the_staged_version(repo):
    root, stage, hook = repo
    stage("deploy", BAD_SH)
    (root / "deploy").write_text(GOOD_SH)  # fixed on disk, not staged
    assert hook("shellcheck").returncode == 1


@needs_shellcheck
def test_shellcheck_sh_without_shebang_is_read_as_bash(repo):
    _, stage, hook = repo
    stage("lib.sh", 'cd $1\n')
    out = hook("shellcheck")
    assert out.returncode == 1 and "SC2164" in out.stdout


@needs_shellcheck
def test_shellcheck_leaves_clean_scripts_zsh_and_other_files_alone(repo):
    _, stage, hook = repo
    stage("ok.sh", GOOD_SH)
    stage("init.zsh", "#!/bin/zsh\nfor f in *(N); cd $f\n")
    stage("tool", "#!/usr/bin/env python3\nprint(1)\n")
    stage("README.md", "cd $1\n")
    out = hook("shellcheck")
    assert out.returncode == 0 and out.stdout == ""


@needs_hadolint
def test_hadolint_warning_stops_the_commit_with_the_file_name(repo):
    _, stage, hook = repo
    stage("api/Dockerfile", BAD_DOCKERFILE)
    out = hook("hadolint")
    assert out.returncode == 1
    assert "api/Dockerfile:1 DL3006" in out.stdout


@needs_hadolint
def test_hadolint_style_only_is_shown_but_passes(repo):
    _, stage, hook = repo
    stage("Containerfile", "FROM debian:12\nRUN apt-get update && apt-get install -y --no-install-recommends curl=7.88.1-10\n")
    out = hook("hadolint")
    assert out.returncode == 0 and "DL3009" in out.stdout  # info: apt lists not deleted


@needs_hadolint
def test_hadolint_checks_named_variants_and_the_staged_version(repo):
    root, stage, hook = repo
    stage("build/app.Dockerfile", BAD_DOCKERFILE)
    (root / "build/app.Dockerfile").write_text(GOOD_DOCKERFILE)
    assert hook("hadolint").returncode == 1


@needs_hadolint
def test_hadolint_clean_or_no_dockerfile_passes(repo):
    root, stage, hook = repo
    stage("Dockerfile", GOOD_DOCKERFILE)
    (root / "Dockerfile.dev").write_text(BAD_DOCKERFILE)  # not staged
    out = hook("hadolint")
    assert out.returncode == 0 and out.stdout == ""


def test_sops_guard_cleartext_sops_file_stops_the_commit(repo):
    _, stage, hook = repo
    stage("k8s/secrets.sops.yaml", "password: hunter2\n")
    out = hook("sops-guard")
    assert out.returncode == 1 and "k8s/secrets.sops.yaml" in out.stderr


def test_sops_guard_encrypted_formats_pass(repo):
    _, stage, hook = repo
    stage("secrets.sops.yaml", ENCRYPTED_YAML)
    stage("app.enc.json", '{"a": "ENC[x]", "sops": {"mac": "ENC[AES256_GCM,data:z]"}}\n')
    stage("prod.sops.env", "A=ENC[x]\nsops_mac=ENC[AES256_GCM,data:z]\n")
    assert hook("sops-guard").returncode == 0


def test_sops_guard_follows_the_path_regex_of_sops_yaml(repo):
    _, stage, hook = repo
    stage(".sops.yaml", "creation_rules:\n  - path_regex: 'secrets/.*\\.yaml$'\n    age: age1x\n"
                        '  - path_regex: "values-prod\\.yaml$"  # prod only\n    age: age1x\n')
    stage("secrets/db.yaml", "password: hunter2\n")
    stage("chart/values-prod.yaml", "token: x\n")
    stage("chart/values-dev.yaml", "token: x\n")
    out = hook("sops-guard")
    assert out.returncode == 1
    assert "secrets/db.yaml" in out.stderr and "chart/values-prod.yaml" in out.stderr
    assert "values-dev" not in out.stderr


def test_sops_guard_reads_the_staged_sops_yaml(repo):
    root, stage, hook = repo
    stage(".sops.yaml", "creation_rules:\n  - path_regex: nothing$\n")
    (root / ".sops.yaml").write_text("creation_rules:\n  - path_regex: \\.yaml$\n")  # not staged
    stage("config.yaml", "a: 1\n")
    assert hook("sops-guard").returncode == 0


def test_sops_guard_ignores_other_files(repo):
    _, stage, hook = repo
    stage("values.yaml", "password: x\n")
    out = hook("sops-guard")
    assert out.returncode == 0 and out.stderr == ""
