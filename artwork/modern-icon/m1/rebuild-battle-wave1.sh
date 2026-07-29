#!/usr/bin/env bash
# 在 demonwinter-art 容器內重建第一批八組怪物三視圖。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

python3 tools/chroma_spritesheet.py \
  --south artwork/modern-icon/m1/masters/battle-wave1-south.png \
  --north artwork/modern-icon/m1/masters/battle-wave1-north.png \
  --east artwork/modern-icon/m1/masters/battle-wave1-east.png \
  --names 00,01,02,03,04,07,08,12 \
  --out artwork/modern-icon/m1/trial
