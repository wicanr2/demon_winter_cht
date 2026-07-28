#!/usr/bin/env bash
# 全程試玩驗收（CONTEXT.md §7 的 A4）。
#
#   tools/playthrough.sh <腳本檔> <輸出目錄> [額外的程式參數…]
#
# 腳本檔一行一個指令：
#
#   key Return          送一個按鍵（xdotool 的鍵名）
#   rep 5 Up            同一個鍵送 5 次（每一步前後都等到畫面回到野外）
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

# **具名 + trap**：外層被 timeout 砍掉時，`docker run --rm` 的容器會活下來
# （殺的是 shell，不是容器）。踩過一次 —— 三個孤兒容器同時跑，
# 互相搶 CPU，於是後續每一次試玩都慢到像卡住，而 log 上什麼都看不到。
CONTAINER="dw-playthrough-$$"
cleanup() { docker kill "$CONTAINER" >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM

docker run --rm --name "$CONTAINER" \
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

# **存檔寫到輸出目錄**，不是容器內的暫存目錄 ——
# 試玩產生的存檔本身就是驗收對象（人數對不對、背包清空了沒），
# 寫在容器裡等於跑完就丟。絕不可寫回 workplace/orig（唯讀）。
DISPLAY=:99 /tmp/demonwinter -trace /out/trace.txt -save /out/PARTY.DAT '"$*"' \
    >/tmp/app.log 2>&1 &
APP_PID=$!
sleep 3
WID=$(DISPLAY=:99 xdotool search --sync --onlyvisible --name "冬之魔" | head -1)
DISPLAY=:99 xdotool windowactivate --sync $WID 2>/dev/null || true
sleep 1

# **按住再放，不要用 `xdotool key`。**
#
# `xdotool key` 的 press 與 release 是連著送的。ebiten 的
# `inpututil.IsKeyJustPressed` 要那個鍵在**某一幀的邊界上是按下的**才算數 ——
# press／release 落在同兩幀之間的話這一下就完全消失。
#
# 症狀是**偶爾漏一個鍵，而且不會報錯**：建角段漏掉一個 Return，
# 後面的按鍵整串錯位一格，於是角色的名字變成種族鍵的數字
# （軌跡裡真的出現過「1 已加入隊伍第 2 位」）。
# 這種錯不會停下來，它會一路跑完然後給你一支看起來正常的隊伍。
#
# keydown → sleep → keyup 把鍵按住三幀以上（60 fps），就落得到邊界上。
send() {
    DISPLAY=:99 xdotool keydown --window $WID --clearmodifiers "$1"
    sleep 0.06
    DISPLAY=:99 xdotool keyup --window $WID --clearmodifiers "$1"
    sleep 0.14
}

# 目前狀態 = 軌跡檔的最後一行（軌跡只在狀態改變時寫，所以末行就是現況）。
now() { tail -1 /out/trace.txt 2>/dev/null || true; }

# 等到隊伍又走得動為止：軌跡的畫面欄位回到「野外」才算。
#
# **判準是白名單而不是黑名單。** 第一版列舉「文字框、戰鬥」兩種模態，
# 然後在第四段路上被商隊撞掉 —— 商隊也是吃掉方向鍵的模態畫面，
# 只是不在清單裡。列舉法每次都會漏掉下一種。
# 反過來寫「不是野外就等」，漏的是「該按什麼鍵」而不是「要不要等」，
# 而前者會停在軌跡裡看得見，後者會安靜地走錯路。
#
# 對應的按鍵：
#   文字框  → Return（翻頁）
#   戰鬥    → 不按，等 -autofight 打完
#   其餘    → Esc（本專案的通則：ESC 只 cancel/back，永遠不會結束遊戲）
#
# 上限存在的理由：關不掉就是真的卡住，要讓它留在軌跡裡，
# 不要用無窮迴圈把問題藏成「跑很久」。
settle() {
    for i in $(seq 1 400); do
        st="$(now)"
        # **順序有意義。** 軌跡一行同時帶「畫面名」與「文字框」旗標，
        # 所以模態旗標一定要排在畫面名之前 —— 反過來寫的話
        # `野外 … 文字框` 會先命中「野外」而直接返回，
        # 文字框沒關掉，後面的方向鍵全部被它吃掉。
        # （第一版就是這樣寫的，第一個事件格就卡住。）
        case "$st" in
            *文字框*) send Return ;;
            *戰鬥*)   sleep 0.25 ;;
            *標題*)   send Return ;;
            *野外*)   return 0 ;;
            *)        send Escape ;;
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
