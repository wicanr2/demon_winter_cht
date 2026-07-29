#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建怪物與船艦的 B 相位透明素材。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

python3 tools/sprite_phase.py \
  artwork/modern-icon/m1/trial/monster-*-south.png \
  artwork/modern-icon/m1/trial/monster-*-west.png \
  artwork/modern-icon/m1/trial/monster-*-east.png \
  artwork/modern-icon/m1/trial/monster-*-north.png \
  artwork/modern-icon/m1/trial/ship-south.png \
  artwork/modern-icon/m1/trial/ship-west.png \
  artwork/modern-icon/m1/trial/ship-east.png \
  artwork/modern-icon/m1/trial/ship-north.png \
  artwork/modern-icon/m1/trial/sea-pirate-south.png \
  artwork/modern-icon/m1/trial/sea-pirate-west.png \
  artwork/modern-icon/m1/trial/sea-pirate-east.png \
  artwork/modern-icon/m1/trial/sea-pirate-north.png \
  artwork/modern-icon/m1/trial/sea-monster-south.png \
  artwork/modern-icon/m1/trial/sea-monster-west.png \
  artwork/modern-icon/m1/trial/sea-monster-east.png \
  artwork/modern-icon/m1/trial/sea-monster-north.png
