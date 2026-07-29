#!/usr/bin/env bash
# 在 demonwinter-go 容器內由核准母稿重建世界地標的 64×56 執行期素材。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

for season in normal winter; do
  for site in temple-25 college-26 town-2e asaht-64; do
    convert "artwork/modern-icon/m1/masters/$season-$site.png" \
      -filter Lanczos -resize 64x56\! -alpha off \
      "artwork/modern-icon/m1/trial/$season-$site.png"
  done
done
