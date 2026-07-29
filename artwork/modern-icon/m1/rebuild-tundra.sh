#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建 0x5a 凍土的常態／冬季四變體。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

tmp_dir="$(mktemp -d /tmp/dw-tundra-XXXXXX)"
cleanup() { rm -rf "$tmp_dir"; }
trap cleanup EXIT INT TERM

for season in normal winter; do
  source="artwork/modern-icon/m1/masters/$season-tundra-5a.png"
  canvas="$tmp_dir/$season-canvas.png"
  convert "$source" -resize '1200x1000^' -gravity center -extent 1200x1000 "$canvas"

  specs=()
  for variant in 0 1 2 3 4 5 6 7; do
    x=$((variant % 4 * 300))
    y=$((variant / 4 * 500))
    offset="+$x+$y"
    crop="$tmp_dir/$season-tundra-5a-v$variant.png"
    convert "$canvas" -crop "300x500$offset" +repage "$crop"
    specs+=("$crop:artwork/modern-icon/m1/trial/$season-tundra-5a-v$variant.png")
  done
  python3 tools/seamless_set.py --width 64 --height 56 --blend 8 "${specs[@]}"
done
