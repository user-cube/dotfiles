#!/bin/bash
# Prints the AWS profiles to show in the bar, one per line, in display order:
# $CONFIG_DIR/aws_profiles (per machine, not in dotfiles) or every configured
# profile. Prints nothing without the aws CLI.
source "$CONFIG_DIR/plugins/helpers.sh"
command -v aws >/dev/null || exit 0

f="$CONFIG_DIR/aws_profiles"
if [ -s "$f" ]; then
  grep -vE '^\s*(#|$)' "$f" | tr -d '[:blank:]'
  exit 0
fi
run aws configure list-profiles 2>/dev/null
