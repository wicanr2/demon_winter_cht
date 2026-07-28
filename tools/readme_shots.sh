#!/usr/bin/env bash
# 重拍 README 的六張截圖（docs/images/）。
#
#   tools/readme_shots.sh            # 全部重拍
#   tools/readme_shots.sh 02-world   # 只拍一張
#
# **為什麼要有這支。** 這六張圖以前是手打指令拍的，指令沒留下來 ——
# 於是每次改動之後沒有人知道「那張圖當初是怎麼拍的」，圖就一路過期
# （字型、狀態列欄位名、鍵位提示都可能已經不一樣了）。
# 版面類的錯誤單元測試看不到，只有截圖比對抓得到，所以流程本身要固定。
#
# 每一張的「這張圖要展示什麼」寫在下面，改動時照著判斷還需不需要重拍。
set -euo pipefail

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT=docs/images
mkdir -p "$OUT"

shot() {
    local name=$1; shift
    echo "== $name"
    timeout 300 tools/screenshot.sh "$OUT/$name.png" "$@"
}

TARGET=${1:-}

# 沒給參數就全拍；給了就只拍那一張。
# ⚠ 呼叫端一定要用 `if want X; then ...; fi`，不要寫 `want X && shot X`：
#    配 `set -e` 時 want 回 false 會讓整支腳本在那一行結束。
want() { [ -z "$TARGET" ] || [ "$TARGET" = "$1" ]; }

# 01 標題：原版美術一格未動，中文標題放在圖上方的黑邊。
#    不送任何按鍵 —— 標題畫面在開場就在，按下去就進遊戲了。
if want 01-title; then shot 01-title; fi

# 02 探索：真正的**世界地圖**（不是地城），正午 —— 白天視野才是完整 9×9。
#    地圖 34 那一帶有水、樹、沙地與山，一張圖看得到多種地形與可通行性。
if want 02-world; then shot 02-world KEYS=Return -map=34 -x=31 -y=44 -hour=12; fi

# 03 事件：地城的 3×3 照明 ＋ 骷髏室的中文敘述。
#    `-event` 直接叫出骷髏室那一筆，不必走到那一格。
#    ⚠ 值是 18 不是 19 —— `-event` 的索引與 `tools/parse_datatxt.py records`
#    印出來的 index 差 1，拍之前先看一眼拍到的是不是那一段。
if want 03-event; then shot 03-event KEYS=Return -event=18; fi

# 04 戰鬥：行動點、法術、地形與視線遮蔽、怪物 AI。
#    `-seed` 釘住亂數，重拍才會是同一批怪物與同一個先手。
if want 04-battle; then shot 04-battle KEYS=Return,b -battle -seed=4; fi

# 05 城鎮：阿薩特（14 號）—— 八種設施全有的那一座，對得上 README 的「八種設施」。
if want 05-town; then shot 05-town KEYS=Return -town=14; fi

# 06 手札：原版印在紙本手冊上的資料，搬進遊戲按 F2 直接查。
#    翻到「神祇」那一章（目錄第 3 項）—— 目錄本身看不出這功能有什麼內容。
if want 06-manual; then shot 06-manual KEYS=Return,F2,Down,Down,Return; fi

echo
echo "拍完。請逐張看過再 commit —— 這一步沒有測試會幫你擋。"
