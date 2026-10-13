#!/bin/bash
# Shared helpers for plugins.
export PATH="/opt/homebrew/bin:$PATH"

# SketchyBar ignores SIGCHLD and children inherit it, which breaks tools that
# wait on their own subprocesses (brew, Go CLIs). Run them with it reset.
run() { perl -e '$SIG{CHLD} = "DEFAULT"; exec @ARGV' "$@"; }
