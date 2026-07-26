#!/usr/bin/env bash
# 全程試玩驗收（CONTEXT.md §7 的 A4）。
#
#   tools/playthrough.sh <腳本檔> <輸出目錄> [額外的程式參數…]
#
# 腳本檔一行一個指令：
#
#   key Return          送一個按鍵（xdotool 的鍵名）
#   rep 5 Up            同一個鍵送 5 次（每一步前後都等到隊伍可行動）
#   wait 1.5            等幾秒（讓動畫／載入跑完）
#   shot 名稱           截一張圖到 <輸出目錄>/名稱.png
#   #  註解
#
# **`rep` 會等模態狀態結束**（文字框、戰鬥），因為不等就走不下去。
# 這是真的跑全程試玩才發現的兩件事：
#
#   1. 踩到事件格會彈出文字框，它把後面所有方向鍵全部吃掉。
#      腳本照樣送鍵，軌跡卻停在那一格 —— 看起來像卡住，其實是模態畫面。
#   2. 事件格會帶遭遇。仗在打的時候方向鍵一樣全部落空，
#      等回到野外時路線已經歪掉 —— 而這個**完全沒有錯誤訊息**，
#      軌跡只是顯示隊伍走到了別的地方。
#
# **單點截圖驗收永遠遇不到這兩件事**，因為它只按幾下就拍照。
#
# 產出：`<輸出目錄>/trace.txt`（每一次狀態變化一行）＋ 沿路的截圖。
#
# **為什麼不用 tools/screenshot.sh 就好。**
# 那支是單點驗收：開一個畫面、拍一張。長路徑上它會騙人 ——
# xdotool 偶爾漏鍵，隊伍少走一步，最後那張圖照樣是張合理的畫面，
# 只是走到了別的地方，而肉眼沒有對照看不出來。
# 軌跡檔補的就是這個對照：走了幾步、到了哪、觸發了什麼，逐行可查。
#
# **這裡不給任何 debug 旗標。** A4 的定義就是「不用捷徑從新遊戲走完」
# （`docs/re/64` §3）。要驗單一畫面請用 screenshot.sh。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE=demonwinter-go
CACHE_VOL=dw-gomod
BUILD_VOL=dw-gobuild

SCRIPT="${1:?用法: tools/playthrough.sh <腳本檔> <輸出目錄> [程式參數…]}"
OUTDIR="${2:?用法: tools/playthrough.sh <腳本檔> <輸出目錄> [程式參數…]}"
shift 2

mkdir -p "$OUTDIR"
OUT_ABS="$(cd "$OUTDIR" && pwd)"
SCRIPT_ABS="$(cd "$(dirname "$SCRIPT")" && pwd)/$(basename "$SCRIPT")"

docker run --rm \
    -v "$REPO_ROOT:/src" \
    -v "$OUT_ABS:/out" \
    -v "$SCRIPT_ABS:/script.txt:ro" \
    -v "$CACHE_VOL:/gomod" \
    -v "$BUILD_VOL:/gocache" \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -e GOCACHE=/gocache \
    -e GOMODCACHE=/gomod \
    -e LIBGL_ALWAYS_SOFTWARE=1 \
    -w /src \
    "$IMAGE" bash -c '
set -e
go build -o /tmp/demonwinter ./cmd/demonwinter
Xvfb :99 -screen 0 1600x900x24 >/tmp/xvfb.log 2>&1 &
XVFB_PID=$!
for i in $(seq 1 50); do xdpyinfo -display :99 >/dev/null 2>&1 && break; sleep 0.1; done

# 存檔寫到暫存目錄：試玩會存檔，絕不可寫回 workplace/orig（唯讀）。
mkdir -p /tmp/pt-save
DISPLAY=:99 /tmp/demonwinter -trace /out/trace.txt -save /tmp/pt-save/PARTY.DAT '"$*"' \
    >/tmp/app.log 2>&1 &
APP_PID=$!
sleep 3
WID=$(DISPLAY=:99 xdotool search --sync --onlyvisible --name "冬之魔" | head -1)
DISPLAY=:99 xdotool windowactivate --sync $WID 2>/dev/null || true
sleep 1

send() { DISPLAY=:99 xdotool key --window $WID --clearmodifiers "$1"; sleep 0.18; }

# 目前狀態 = 軌跡檔的最後一行（軌跡只在狀態改變時寫，所以末行就是現況）。
now() { tail -1 /out/trace.txt 2>/dev/null || true; }

# 等到隊伍又走得動為止：關掉文字框、等戰鬥打完。
#
# 兩件事都是**吃掉方向鍵**的模態狀態。不等的話腳本會照樣一路送鍵，
# 那些鍵全部落空，等回到野外時路線已經對不上了 ——
# 而軌跡看起來只是「走到別的地方」，沒有任何錯誤。
# 第一次接上自動戰鬥就是這樣：仗打贏了，後面的路線全歪。
#
# 上限存在的理由：關不掉／打不完就是真的卡住，要讓它留在軌跡裡，
# 不要用無窮迴圈把問題藏成「跑很久」。
settle() {
    for i in $(seq 1 400); do
        case "$(now)" in
            *文字框*) send Return ;;
            *戰鬥*)   sleep 0.25 ;;
            *)        return 0 ;;
        esac
    done
    echo "!! 等不到隊伍恢復可行動，停在：$(now)"
}

while read -r cmd arg rest; do
    case "$cmd" in
        ""|"#") ;;
        key)  send "$arg" ;;
        rep)  for i in $(seq 1 "$arg"); do settle; send "$rest"; settle; done ;;
        wait) sleep "$arg" ;;
        shot) sleep 0.4; DISPLAY=:99 import -window root "/out/$arg.png" ;;
        *)    echo "!! 看不懂的指令：$cmd $arg" ;;
    esac
    # 程式掛了就別再送鍵 —— 後面的按鍵會全部落空，
    # 軌跡看起來像「走到一半停住」，實際是崩潰。要分得出來。
    if ! kill -0 $APP_PID 2>/dev/null; then
        echo "!! 程式已結束，腳本在 [$cmd $arg $rest] 之後中止"
        break
    fi
done < /script.txt

sleep 0.6
kill -9 $APP_PID $XVFB_PID 2>/dev/null || true
wait $APP_PID 2>/dev/null || true
echo "--- app log ---"; cat /tmp/app.log || true
'
echo "軌跡 -> $OUT_ABS/trace.txt"
