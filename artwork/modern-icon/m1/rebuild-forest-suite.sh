#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建 0x01–0x0c 密林索引的正常／冬季試片。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

tiles=(01 02 03 05 06 08 09 0a 0c)
offsets=(
  +0+0 +227+0 +454+0
  +0+227 +227+227 +454+227
  +0+454 +227+454 +454+454
)

for season in normal winter; do
  specs=()
  source="artwork/modern-icon/m1/masters/$season-forest-suite.png"
  for i in "${!tiles[@]}"; do
    tile="${tiles[$i]}"
    crop="/tmp/$season-forest-$tile.png"
    convert "$source" -crop "800x800${offsets[$i]}" +repage "$crop"
    specs+=("$crop:artwork/modern-icon/m1/trial/$season-forest-$tile.png")
  done
  python3 tools/seamless_set.py --width 64 --height 56 --blend 8 "${specs[@]}"
done
