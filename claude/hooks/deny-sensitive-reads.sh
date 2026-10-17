#!/usr/bin/env bash
# PreToolUse hook (matchers: Bash, Read).
#
# Bash: the Read(...) deny rules in settings.json do not cover Bash: a plain
# `cat file` still goes through. This hook inspects the command line and
# blocks it when it touches sensitive paths.
#
# secrets/ and credentials/ directories: files there may be read only when
# they ARE encrypted, judged by content with `sops filestatus` (which also
# rejects a hand-written fake `sops:` block), not by file name. They are
# guarded here rather than with Read(...) deny rules because a permission
# rule can't say "except encrypted files".
#
# LIMIT: for Bash the paths are taken from the command text. Paths built at
# runtime (variables, cd then a relative path, base64) are not resolved; when
# a secrets/ path can't be resolved to a file, the old name rule applies
# (.enc. / .encrypted. / .sops. in the command). A safety net, not a sandbox.

set -uo pipefail

input=$(cat)
tool=$(jq -r '.tool_name // ""' <<<"$input" 2>/dev/null) || exit 0
cwd=$(jq -r '.cwd // ""' <<<"$input" 2>/dev/null)
# Relative paths in commands resolve from the session's directory. If that
# fails they don't resolve, and the stricter name rule applies.
if [ -n "$cwd" ]; then cd "$cwd" 2>/dev/null || true; fi

deny () {
  jq -nc --arg r "$1" \
    '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'
  exit 0
}

SECRET_DIRS='(^|[^a-z0-9_-])(secrets|credentials)/'
ENCRYPTED_NAME='\.(enc|encrypted|sops)\.(ya?ml|json|hcl|env|ini)'

# True only when sops itself reports the file as encrypted. A cleartext file
# returns false; a fake or broken sops block makes sops error out: both deny.
is_encrypted () {
  [ -f "$1" ] || return 1
  if command -v sops >/dev/null 2>&1; then
    sops filestatus "$1" 2>/dev/null | jq -e '.encrypted == true' >/dev/null 2>&1
  else
    # Without sops: require the metadata block and at least one ENC[ value.
    grep -Eq '^sops:|"sops"[[:space:]]*:' "$1" && grep -q 'ENC\[' "$1"
  fi
}

if [ "$tool" = "Read" ]; then
  path=$(jq -r '.tool_input.file_path // ""' <<<"$input")
  [ -z "$path" ] && exit 0
  if printf '%s' "$path" | grep -Eiq "$SECRET_DIRS"; then
    # Nothing to leak from a missing file: let Read report "not found".
    [ -e "$path" ] || exit 0
    is_encrypted "$path" || deny "File under secrets/ or credentials/ is not SOPS-encrypted (sops filestatus; local rule in ~/.claude/hooks/deny-sensitive-reads.sh)."
  fi
  exit 0
fi

cmd=$(jq -r '.tool_input.command // ""' <<<"$input" 2>/dev/null) || exit 0
[ -z "$cmd" ] && exit 0

# Cleartext credentials: blocked unconditionally.
# `.key` must look like a path, not an expression: `each.key` and `local.key`
# are ordinary HCL/Python and must not trip this.
if printf '%s' "$cmd" | grep -Eiq 'age-key|\.agekey|AGE-SECRET-KEY|\.pem([^a-z]|$)|[/~][a-z0-9_.-]*\.key([^a-z]|$)|(^|[^a-z0-9_-])\.aws/|(^|[^a-z0-9_-])\.ssh/|(^|[^a-z0-9_-])\.env|(^|/)env\.php|config/database\.yml'; then
  deny "Command touches a cleartext credential file (local rule in ~/.claude/hooks/deny-sensitive-reads.sh). Ask the user to do it themselves."
fi

if printf '%s' "$cmd" | grep -Eiq 'docker-compose[^ ]*\.ya?ml'; then
  deny "Command touches a docker-compose file (local rule). Ask the user to do it themselves."
fi

# secrets/ and credentials/: check every path the command names. Each one
# must resolve to files that are all encrypted; a path that resolves to no
# file falls back to the name rule.
while IFS= read -r token; do
  [ -z "$token" ] && continue
  p=${token/#\~/$HOME}
  shopt -s nullglob
  # Unquoted on purpose: expands globs such as github*.encrypted.yaml.
  # shellcheck disable=SC2206
  files=( $p )
  shopt -u nullglob
  existing=()
  for f in "${files[@]}"; do [ -f "$f" ] && existing+=("$f"); done
  if [ ${#existing[@]} -gt 0 ]; then
    for f in "${existing[@]}"; do
      is_encrypted "$f" || deny "Command touches $f, which is not SOPS-encrypted (sops filestatus; local rule)."
    done
  elif ! printf '%s' "$token" | grep -Eq "$ENCRYPTED_NAME"; then
    deny "Command touches secrets/ or credentials/ ($token) and it can't be checked as an encrypted file (local rule)."
  fi
done < <(printf '%s' "$cmd" | grep -Eoi "[^[:space:]\"'=<>|;&()]*(secrets|credentials)/[^[:space:]\"'<>|;&()]*" \
  | grep -Ei '(^|[/~])(secrets|credentials)/')   # a whole path segment: not external-secrets/

exit 0
