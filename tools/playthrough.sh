#!/usr/bin/env bash
# 全程試玩驗收（CONTEXT.md §7 的 A4）。
#
#   tools/playthrough.sh <腳本檔> <輸出目錄> [額外的程式參數…]
#
# 腳本檔一行一個指令：
#
#   key Return          送一個按鍵（xdotool 的鍵名）
#   rep 5 Up            同一個鍵送 5 次（每一步前後都等到畫面回到野外）
#   at 34:23,31         驗目前確實在指定地圖／座標；不符就立刻中止
#   to 34:23,31 Up      沿單一方向送鍵直到抵達座標（可抵抗 X11 偶發漏鍵）
#   hunt Left Right     兩方向各走 8 鍵巡邏，打完第一場隨機遭遇就停止
#   settle              關完目前所有模態畫面，直到回到野外
#   expect 已存檔       驗證 trace 至少出現過指定文字
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
# 設 `DW_RECORD=live.mp4` 可另外把整段實機操作錄成影片；檔名相對於輸出目錄。
# 這走 Xvfb 的 x11grab，錄到的是遊戲真正畫出的每一幀，不是重畫的 mockup。
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
    --network none \
    --memory 2g \
    --cpus 2 \
    --pids-limit 256 \
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
    -e "DW_RECORD=${DW_RECORD:-}" \
    -e "DW_RECORD_FPS=${DW_RECORD_FPS:-25}" \
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

REC_PID=
if [ -n "$DW_RECORD" ]; then
    case "$DW_RECORD" in
        */*|.*) echo "!! DW_RECORD 只能是輸出目錄內的檔名"; exit 1 ;;
    esac
    DISPLAY=:99 ffmpeg -y -loglevel error \
        -f x11grab -framerate "$DW_RECORD_FPS" -video_size 1600x900 -i :99.0 \
        -threads 2 -c:v libx264 -preset veryfast -pix_fmt yuv420p \
        "/out/$DW_RECORD" &
    REC_PID=$!
fi

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

# 長路徑不能只信「送了幾個鍵」。X11 偶發漏一鍵時，後面的路段仍可能全部
# 跑完，最後只留下一張看似合理、其實在別處的截圖。路線腳本可在轉折點用
# `at 地圖:X,Y` 及早失敗，把偏航定位在上一小段。
is_at() {
    map="${1%%:*}"
    xy="${1#*:}"
    x="${xy%%,*}"
    y="${xy#*,}"
    st="$(now)"
    printf "%s\n" "$st" | grep -Eq "\\( *${x}, *${y}\\) 地圖${map}( |$)"
}

assert_at() {
    if ! is_at "$1"; then
        echo "!! 座標驗證失敗：預期 $1，實際：$st"
        return 1
    fi
}

expect_trace() {
    text="$1"
    if ! grep -Fq "$text" /out/trace.txt; then
        echo "!! 軌跡缺少預期文字：$text；末態：$(now)"
        return 1
    fi
}

walk_to() {
    target="$1"
    key="$2"
    for i in $(seq 1 128); do
        if is_at "$target"; then
            return 0
        fi
        settle
        send "$key"
        settle
    done
    echo "!! 直線移動超過 128 鍵：預期 $target，實際：$(now)"
    return 1
}

# hunt_until_battle 給正常練功腳本用：只把方向鍵送進真實世界移動，
# 遭遇、戰鬥與獎勵仍全走遊戲本身。以 trace 中新增的「戰鬥」狀態為證，
# 一場結束後立刻停，不會因固定步數又撞進第二場。
hunt_until_battle() {
    first="$1"
    second="$2"
    start_lines="$(wc -l < /out/trace.txt)"
    for i in $(seq 0 199); do
        block=$((i / 8))
        key="$first"
        if [ $((block % 2)) -eq 1 ]; then
            key="$second"
        fi
        settle
        send "$key"
        settle
        if tail -n "+$((start_lines + 1))" /out/trace.txt | grep -q "戰鬥"; then
            return 0
        fi
    done
    echo "!! 巡邏 200 鍵仍未遇敵，停在：$(now)"
    return 1
}

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
            *死亡*)
                echo "!! 隊伍全滅，停在：$st"
                return 1
                ;;
            *標題*)   send Return ;;
            *紮營*)
                # 時辰 24 會由遊戲強制開營。正常流程在這裡睡一晚；
                # 睡完仍停在營地選單，所以再收帳回野外。
                send s
                sleep 0.2
                if now | grep -q "紮營"; then send Escape; fi
                ;;
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
        at)   assert_at "$arg" ;;
        to)   walk_to "$arg" "$rest" ;;
        hunt) hunt_until_battle "$arg" "$rest" ;;
        settle) settle ;;
        expect) expect_trace "$arg${rest:+ $rest}" ;;
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
if [ -n "$REC_PID" ]; then
    kill -INT "$REC_PID" 2>/dev/null || true
    wait "$REC_PID" 2>/dev/null || true
fi
kill -9 $APP_PID $XVFB_PID 2>/dev/null || true
wait $APP_PID 2>/dev/null || true
echo "--- app log ---"; cat /tmp/app.log || true
'
echo "軌跡 -> $OUT_ABS/trace.txt"
