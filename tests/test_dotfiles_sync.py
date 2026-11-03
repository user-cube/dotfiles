"""dotfiles-sync (.config/zsh/dotfiles.zsh): the function a shell loaded when it
started may be older than what the pull brings, so after the pull the rest runs
from the clone's own dotfiles.zsh. A throwaway origin and clone, git without the
user's config (no global hooks, no signing)."""

import os
import subprocess

from conftest import CONFIG, zsh

DOTFILES_ZSH = CONFIG / "zsh" / "dotfiles.zsh"


def git(*args, cwd):
    env = dict(os.environ, GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1",
               GIT_AUTHOR_NAME="t", GIT_AUTHOR_EMAIL="t@t", GIT_COMMITTER_NAME="t", GIT_COMMITTER_EMAIL="t@t")
    subprocess.run(["git", *args], cwd=cwd, env=env, check=True, capture_output=True)


def test_the_rest_runs_from_the_pulled_dotfiles_zsh(tmp_path):
    origin, clone = tmp_path / "origin", tmp_path / "clone"
    target = origin / ".config" / "zsh" / "dotfiles.zsh"
    target.parent.mkdir(parents=True)
    target.write_text(DOTFILES_ZSH.read_text())
    git("init", "-q", "-b", "main", str(origin), cwd=tmp_path)
    git("add", "-A", cwd=origin)
    git("commit", "-qm", "v1", cwd=origin)
    git("clone", "-q", str(origin), str(clone), cwd=tmp_path)

    # What the pull brings: a newer install step
    target.write_text(DOTFILES_ZSH.read_text() + '\n_dotfiles_sync_install() { echo "pulled install of $1" }\n')
    git("commit", "-qam", "v2", cwd=origin)

    env = dict(os.environ, HOME=str(tmp_path), DOTFILES_DIR=str(clone),
               GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1")
    # the shell's dotfiles-sync is this checkout's, as loaded at startup
    out = zsh(f'source "{DOTFILES_ZSH}" && dotfiles-sync', env)
    assert out.returncode == 0, out.stderr
    assert f"pulled install of {clone}" in out.stdout
