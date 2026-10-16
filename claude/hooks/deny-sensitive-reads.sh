#!/usr/bin/env bash
# PreToolUse hook (matcher: Bash).
#
# The Read(...) deny rules in settings.json do not cover Bash: a plain
# `cat file` still goes through. This hook inspects the command line and
# blocks it when it touches sensitive paths.
#
# LIMIT: this is a text match over the command line. It catches accidents
# (cat, grep, less, cp). It does NOT catch deliberate obfuscation --
# variables, base64, globs that only expand at runtime. It is a safety net,
# not a sandbox.

set -uo pipefail

cmd=$(jq -r '.tool_input.command // ""' 2>/dev/null) || exit 0
[ -z "$cmd" ] && exit 0

deny () {
  jq -nc --arg r "$1" \
    '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'
  exit 0
}

# Already-encrypted SOPS files are safe to read, and the workflow depends on
# it (sops -d secrets/<name>.enc.yaml).
is_encrypted_only=0
if printf '%s' "$cmd" | grep -Eq '\.enc\.(yaml|json|hcl)'; then
  is_encrypted_only=1
fi

# Cleartext credentials: blocked unconditionally.
# `.key` must look like a path, not an expression: `each.key` and `local.key`
# are ordinary HCL/Python and must not trip this.
if printf '%s' "$cmd" | grep -Eiq 'age-key|\.agekey|AGE-SECRET-KEY|\.pem([^a-z]|$)|[/~][a-z0-9_.-]*\.key([^a-z]|$)|(^|[^a-z0-9_-])\.aws/|(^|[^a-z0-9_-])\.ssh/|(^|[^a-z0-9_-])\.env|(^|/)env\.php|config/database\.yml'; then
  deny "Command touches a cleartext credential file (local rule in ~/.claude/hooks/deny-sensitive-reads.sh). Ask the user to do it themselves."
fi

# Secret directories: only blocked when not referring to a .enc.* file.
if [ "$is_encrypted_only" -eq 0 ]; then
  if printf '%s' "$cmd" | grep -Eiq '(^|[^a-z0-9_-])secrets/|(^|[^a-z0-9_-])credentials/|docker-compose[^ ]*\.ya?ml'; then
    deny "Command touches secrets/ or credentials/ without naming a .enc.* file (local rule). Encrypted files are allowed."
  fi
fi

exit 0