#!/usr/bin/env bash
# 在 headless X（Xvfb）底下跑 cmd/demonwinter 並截一張圖。
#
#   tools/screenshot.sh out.png [-- 額外的程式參數…]
#
# 用途是驗收：本專案的硬規則是視覺產物一律 dump PNG 肉眼比對，
# 不接受「編譯過了 / 測試綠」當作畫面正確的證據。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE=demonwinter-go
CACHE_VOL=dw-gomod
BUILD_VOL=dw-gobuild

OUT="${1:?用法: tools/screenshot.sh <輸出.png> [KEYS=按鍵序列] [程式參數…]}"
shift || true

# KEYS=Up,Up,Right 這種形式會在截圖前用 xdotool 依序送出，
# 用來驗證「按了會怎樣」而不只是「開得起來」。
KEYS=""
if [[ "${1:-}" == KEYS=* ]]; then
    KEYS="${1#KEYS=}"
    shift
fi

mkdir -p "$(dirname "$OUT")"
OUT_ABS="$(cd "$(dirname "$OUT")" && pwd)/$(basename "$OUT")"

docker run --rm \
    -v "$REPO_ROOT:/src" \
    -v "$(dirname "$OUT_ABS"):/out" \
    -v "$CACHE_VOL:/gomod" \
    -v "$BUILD_VOL:/gocache" \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -e GOCACHE=/gocache \
    -e GOMODCACHE=/gomod \
    -e LIBGL_ALWAYS_SOFTWARE=1 \
    -w /src \
    "$IMAGE" bash -c "
set -e
go build -o /tmp/demonwinter ./cmd/demonwinter
Xvfb :99 -screen 0 1600x900x24 >/tmp/xvfb.log 2>&1 &
XVFB_PID=\$!
for i in \$(seq 1 50); do xdpyinfo -display :99 >/dev/null 2>&1 && break; sleep 0.1; done
DISPLAY=:99 /tmp/demonwinter $* >/tmp/app.log 2>&1 &
APP_PID=\$!
sleep 3
if [ -n '$KEYS' ]; then
    WID=\$(DISPLAY=:99 xdotool search --sync --onlyvisible --name "冬之魔" | head -1)
    DISPLAY=:99 xdotool windowactivate --sync \$WID 2>/dev/null || true
    sleep 1
    for k in \$(echo '$KEYS' | tr ',' ' '); do
        DISPLAY=:99 xdotool key --window \$WID --clearmodifiers \$k
        sleep 0.25
    done
    sleep 0.6
fi
DISPLAY=:99 import -window root /out/$(basename "$OUT_ABS")
# 一律 KILL：TERM 有機會被卡在裝置 I/O 的執行緒忽略，
# 那時 wait 會永遠等下去，整個截圖流程掛住。
kill -9 \$APP_PID \$XVFB_PID 2>/dev/null || true
wait \$APP_PID 2>/dev/null || true
echo '--- app log ---'; cat /tmp/app.log || true
"
echo "截圖 -> $OUT_ABS"
