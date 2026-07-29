#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建丘陵／高山的常態與冬季 64×56 試片。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

for family in hills mountains; do
  case "$family" in
    hills) tiles=(0e 2b) ;;
    mountains) tiles=(0f 10) ;;
  esac
  for season in normal winter; do
    specs=()
    for tile in "${tiles[@]}"; do
      source="artwork/modern-icon/m1/masters/$season-$family-$tile.png"
      for variant in 0 1 2 3; do
        case "$variant" in
          0) offset="+0+0" ;;
          1) offset="+454+0" ;;
          2) offset="+0+454" ;;
          3) offset="+454+454" ;;
        esac
        crop="/tmp/$season-$family-$tile-v$variant.png"
        convert "$source" -crop "800x800$offset" +repage "$crop"
        specs+=("$crop:artwork/modern-icon/m1/trial/$season-$family-$tile-v$variant.png")
      done
    done
    python3 tools/seamless_set.py --width 64 --height 56 --blend 8 "${specs[@]}"
  done
done
