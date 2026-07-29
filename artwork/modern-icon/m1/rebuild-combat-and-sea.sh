#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建三組隊員與兩組敵方海戰四方向素材。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

python3 tools/chroma_direction_grid.py \
  --sheet artwork/modern-icon/m1/masters/combat-class-directions.png \
  --names martial,arcane,agile \
  --prefix combat \
  --out artwork/modern-icon/m1/trial

python3 tools/chroma_direction_grid.py \
  --sheet artwork/modern-icon/m1/masters/sea-enemy-directions.png \
  --names pirate,monster \
  --prefix sea \
  --out artwork/modern-icon/m1/trial
