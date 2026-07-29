#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建最後七個世界地圖特殊索引。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

python3 tools/terrain_grid.py \
  --sheet artwork/modern-icon/m1/masters/world-specials-15-55.png \
  --indices 15,16,2d,2f,35,36,55 \
  --out artwork/modern-icon/m1/trial
