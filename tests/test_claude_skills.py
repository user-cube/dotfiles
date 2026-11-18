"""The skills and subagents in claude/ are read by Claude Code, which skips one
with a broken header without saying so: a typo fails here instead."""

import re

import pytest
import yaml

from conftest import REPO

SKILLS = sorted((REPO / "claude/skills").glob("*/SKILL.md"))
AGENTS = sorted((REPO / "claude/agents").glob("*.md"))


def frontmatter(path):
    match = re.match(r"^---\n(.*?)\n---\n", path.read_text(), re.S)
    assert match, f"{path.relative_to(REPO)} has no --- header ---"
    meta = yaml.safe_load(match.group(1))
    assert isinstance(meta, dict), f"{path.relative_to(REPO)}: the header is not a mapping"
    return meta


@pytest.mark.parametrize("path", SKILLS, ids=lambda p: p.parent.name)
def test_skill_header(path):
    meta = frontmatter(path)
    assert meta.get("name") == path.parent.name, "name must be the directory's"
    assert str(meta.get("description", "")).strip(), "a skill needs a description: it is how Claude picks it"
    assert path.read_text().split("---\n", 2)[2].strip(), "a skill needs a body"


@pytest.mark.parametrize("path", AGENTS, ids=lambda p: p.stem)
def test_agent_header(path):
    meta = frontmatter(path)
    assert meta.get("name") == path.stem, "name must be the file's"
    assert str(meta.get("description", "")).strip(), "an agent needs a description: it is how Claude picks it"
    assert path.read_text().split("---\n", 2)[2].strip(), "an agent needs a prompt"
    # least privilege is the point of an agent here: tools are listed, never inherited from everything
    assert meta.get("tools"), "list the tools the agent may use"


@pytest.mark.parametrize("path", SKILLS + AGENTS, ids=lambda p: p.parent.name if p.name == "SKILL.md" else p.stem)
def test_no_attribution_in_what_claude_will_write(path):
    """A skill that tells Claude to sign its commits would fight git-guard."""
    text = path.read_text()
    for line in text.splitlines():
        lowered = line.lower()
        if "co-authored-by" in lowered or "generated with" in lowered:
            assert any(w in lowered for w in ("no ", "never", "deny", "denies", "not ")), \
                f"{path.relative_to(REPO)}: attribution mentioned without forbidding it: {line!r}"
