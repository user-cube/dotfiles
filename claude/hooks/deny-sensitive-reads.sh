#!/usr/bin/env bash
# PreToolUse hook (matchers: Bash, Read).
#
# Bash: the Read(...) deny rules in settings.json do not cover Bash: a plain
# `cat file` still goes through. This hook inspects the command line and
# blocks it when it touches sensitive paths.
#
# Read: secrets/ and credentials/ directories are guarded here instead of
# with Read(...) deny rules, because a permission rule can't say "except
# encrypted files" (a `!` exception can't carve into a `//`-anchored rule).
# Files there are readable only when they are SOPS-encrypted: the name says
# so (.enc. / .encrypted. / .sops.) AND the content carries SOPS metadata.
#
# LIMIT: for Bash this is a text match over the command line. It catches
# accidents (cat, grep, less, cp). It does NOT catch deliberate obfuscation --
# variables, base64, globs that only expand at runtime. It is a safety net,
# not a sandbox.

set -uo pipefail

input=$(cat)
tool=$(jq -r '.tool_name // ""' <<<"$input" 2>/dev/null) || exit 0

deny () {
  jq -nc --arg r "$1" \
    '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'
  exit 0
}

ENCRYPTED_NAME='\.(enc|encrypted|sops)\.(ya?ml|json|hcl|env)'
SECRET_DIRS='(^|[^a-z0-9_-])(secrets|credentials)/'

# A file is SOPS-encrypted when it carries the sops metadata block:
# a top-level `sops:` key (YAML) or a "sops" object (JSON).
is_sops_file () {
  [ -f "$1" ] || return 1
  grep -Eq '^sops:|"sops"[[:space:]]*:' "$1" 2>/dev/null
}

if [ "$tool" = "Read" ]; then
  path=$(jq -r '.tool_input.file_path // ""' <<<"$input")
  [ -z "$path" ] && exit 0
  if printf '%s' "$path" | grep -Eiq "$SECRET_DIRS"; then
    # Nothing to leak from a missing file: let Read report "not found".
    [ -e "$path" ] || exit 0
    if ! printf '%s' "$path" | grep -Eiq "${ENCRYPTED_NAME}\$"; then
      deny "Read of a file under secrets/ or credentials/ that isn't named as SOPS-encrypted (local rule in ~/.claude/hooks/deny-sensitive-reads.sh)."
    fi
    if ! is_sops_file "$path"; then
      deny "File under secrets/ or credentials/ is named as encrypted but has no SOPS metadata (local rule). Ask the user to check it."
    fi
  fi
  exit 0
fi

cmd=$(jq -r '.tool_input.command // ""' <<<"$input" 2>/dev/null) || exit 0
[ -z "$cmd" ] && exit 0

# Already-encrypted SOPS files are safe to read, and the workflow depends on
# it (sops -d secrets/<name>.enc.yaml, common/secrets/<name>.encrypted.yaml).
is_encrypted_only=0
if printf '%s' "$cmd" | grep -Eq "$ENCRYPTED_NAME"; then
  is_encrypted_only=1
fi

# Cleartext credentials: blocked unconditionally.
# `.key` must look like a path, not an expression: `each.key` and `local.key`
# are ordinary HCL/Python and must not trip this.
if printf '%s' "$cmd" | grep -Eiq 'age-key|\.agekey|AGE-SECRET-KEY|\.pem([^a-z]|$)|[/~][a-z0-9_.-]*\.key([^a-z]|$)|(^|[^a-z0-9_-])\.aws/|(^|[^a-z0-9_-])\.ssh/|(^|[^a-z0-9_-])\.env|(^|/)env\.php|config/database\.yml'; then
  deny "Command touches a cleartext credential file (local rule in ~/.claude/hooks/deny-sensitive-reads.sh). Ask the user to do it themselves."
fi

# Secret directories: only blocked when not referring to an encrypted file.
if [ "$is_encrypted_only" -eq 0 ]; then
  if printf '%s' "$cmd" | grep -Eiq "${SECRET_DIRS}|docker-compose[^ ]*\.ya?ml"; then
    deny "Command touches secrets/ or credentials/ without naming an encrypted (.enc. / .encrypted.) file (local rule). Encrypted files are allowed."
  fi
fi

exit 0
