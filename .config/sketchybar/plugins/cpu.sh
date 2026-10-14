#!/bin/bash
# Total CPU, normalised by core count
CORES=$(sysctl -n hw.ncpu)
TOTAL=$(ps -A -o %cpu= | awk -v c="$CORES" '{s+=$1} END {printf "%.0f", s/c}')
sketchybar --push "$NAME" "$(awk -v t="$TOTAL" 'BEGIN{printf "%.2f", t/100}')" \
           --set "$NAME" label="$TOTAL%"
