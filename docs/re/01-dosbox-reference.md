# DOSBox 參考環境（reference oracle）

本檔記錄 Demon's Winter（SSI, 1988）逆向工程專案的 DOSBox headless 參考環境：
怎麼跑、CGA/EGA 怎麼切換、截圖與自動化輸入怎麼做、踩過的雷，以及一次完整的
存檔 diff 實驗（隊伍朝向／座標／時間欄位）。這個環境的用途是**原版遊戲實際行為的
裁判**——後續任何「我們重寫的引擎對不對」的問題，最終都拿這裡跑出來的畫面/存檔
去對答案。

## 環境現況（2026-07-24 建置）

- **DOSBox 0.74-3**（Debian bookworm 套件版本，`apt install dosbox`）
- base image `debian:bookworm-slim`
- image：`demwin-dosbox:latest`，Dockerfile 在 `docker/dosbox/Dockerfile`
- 全程 docker，系統上沒有另外裝 DOSBox / Xvfb / imagemagick / xdotool
- headless：容器內用 **Xvfb** 起虛擬 X framebuffer，**xdotool** 送鍵，
  **ImageMagick `import`** 對 X root window 截圖

選 DOSBox 0.74（而非 dosbox-x）的理由：這個遊戲附的原始 `dosbox.conf` 就是
0.74 格式（`# This is the configurationfile for DOSBox 0.74.`），Debian
官方套件庫直接有 0.74-3，不用額外編譯；0.74 的 `-conf` / `-c` / autoexec
機制已經足夠支撐本專案要的自動化程度（見下方「自動化輸入」一節的結論），沒有
用到 dosbox-x 才有的功能，維持環境單純優先。

## 執行指令（一行啟動）

```bash
cd /home/anr2/cht/daemon_winter

# 第一次跑（或改了 docker/dosbox/Dockerfile、entrypoint.sh 之後）會自動 build image，
# 也會自動把 workplace/orig/demwin/ 複製一份可寫副本到 workplace/dosbox/game/
# （workplace/orig/ 唯讀，不可動；遊戲需要寫入存檔，所以另外開一份可寫副本）。

# 最簡形式：只跑起來，等 5 秒截一張圖（存成 workplace/dosbox/shots/ega-default.png）
./tools/dosbox_run.sh ega

# 指定模式 + 自動化 timeline（見下方「自動化輸入」）
./tools/dosbox_run.sh ega "wait:1.5;shot:01-ega-opening;wait:2.5;shot:02-ega-mainmenu;key:Return;wait:1.5;shot:03-ega-ingame"

# CGA 模式（會卡住，見下方「踩到的雷」第一條，這是刻意示範，不是腳本壞掉）
./tools/dosbox_run.sh cga "wait:8;shot:05-cga-hang-open-pic"
```

`tools/dosbox_run.sh` 簽名：

```
tools/dosbox_run.sh <cga|ega> [timeline] [cycles]
```

- `mode`：`cga` 或 `ega`，對應 DOSBox `[dosbox] machine=` 設定，決定遊戲偵測到
  哪種顯示卡、載入哪一套素材（`.PIC/.SHP/.FNT` vs `.PIE/.SHE/.FNE`）
- `timeline`：選填，`;` 分隔的自動化步驟，見下一節
- `cycles`：選填，DOSBox `[cpu] cycles` 設定，預設 `"fixed 4000"`（見「踩到的雷」
  第三條，為什麼用 fixed 不用 auto）

截圖固定存到 `workplace/dosbox/shots/*.png`（不入版控）。遊戲可寫副本固定在
`workplace/dosbox/game/`（不入版控，`workplace/orig/` 全程唯讀未動）。

## CGA / EGA 切換

切換方式就是 `dosbox_run.sh` 的第一個參數，內部對應 DOSBox conf 的
`[dosbox] machine=cga` 或 `machine=ega`（entrypoint 腳本 `docker/dosbox/entrypoint.sh`
在容器內動態產生這份 conf，不是靜態檔案）。遊戲本身會偵測這個模擬的顯示卡，
自動載入對應副檔名的素材（`OPEN.PIC`＋`.SHP`＋`.FNT` vs `OPEN.PIE`＋`.SHE`＋`.FNE`）。

**目前只有 EGA 模式能真正玩到主選單以後**（見下面「踩到的雷」第一條，CGA 模式
在讀取 `GOT.FNT` 時卡死，原因是這個檔案不在我們手上這份 `Demons Winter (1988).zip`
裡）。這不是 DOSBox 環境的問題，是素材本身缺檔；CGA 的開場圖（`OPEN.PIC`）本身
渲染沒問題（見 `workplace/dosbox/shots/05-cga-hang-open-pic.png`），只是卡在那之後
的資料載入。

## 截圖

### 取得方式

容器內流程（`docker/dosbox/entrypoint.sh`）：

1. `Xvfb :99 -screen 0 1024x768x24 -nolisten tcp &` 起虛擬 framebuffer
2. `export DISPLAY=:99`，`dosbox -conf <動態產生的conf> -userconf &` 背景啟動
3. 用 `xdotool search --name DOSBox` 輪詢等視窗出現（最多 15 秒）
4. **關鍵**：`xdotool windowfocus <winid>`（不是 `windowactivate`，見「踩到的雷」）
5. 需要截圖時：`import -window root /shots/<NAME>.png`（對整個 X root 截圖，
   不特別指定 DOSBox 視窗 id——因為沒有 window manager，視窗永遠貼在 root 左上角，
   直接截 root 最簡單可靠）

### 存放位置

`workplace/dosbox/shots/*.png`（不入版控，`.gitignore` 已排除整個 `workplace/`）。

### 目前已驗證的截圖（本次任務產出，供後續 agent 直接參考）

| 檔名 | 內容 | 產生方式 |
|---|---|---|
| `01-ega-opening.png` | EGA 開場美術：紅色惡魔全彩插畫、標題字 logo "DEMON'S WINTER"、"Adapted for the IBM by Judit Buczolich and Laszlo Fazekas"、SSI + Novotrade 版權標 | `wait:1.5` 左右截到（時機抓太早是全黑、太晚已經跳到主選單，見「踩到的雷」第四條） |
| `02-ega-mainmenu.png` | 主選單：標題「DEMON'S WINTER / Version 1.0 / Designed by Craig Roth and David Stark」＋三個選項「Go adventuring / Character Utilities / Alternate Character Set」＋版權 | 開場後約 2.5 秒左右截到 |
| `03-ega-ingame.png` | 進入遊戲的實際畫面：左上小地圖（隊伍站在城鎮建築前）、右上隊伍狀態列（Gold: 65, Pro: 8, 5 名角色的 HP/SP：Wopple/Stumpy/Podgom/Worman/Menhir）、右側完整指令選單（Walk/Party info/Save Game/Camp/Cook/Take/Drop/Move/Examine/Use/Inspect/View room/View item/Read descr/Quit） | 主選單按 Return 選「Go adventuring」，載入專案內建的 `PARTY.DAT` 存檔 |
| `05-cga-hang-open-pic.png` | CGA 模式的開場圖：品紅/青兩色 CGA 調色盤版「DEMON'S WINTER」美術＋一個空的載入進度條方框，畫面永久停在這裡（見踩雷第一條） | CGA 模式跑 8 秒後截圖，供存證 |

## 自動化輸入：可行性結論

**結論：可行，且已驗證到「開機 → 選單導航 → 進遊戲 → 移動 → 存檔」全鏈路。**

用的是 **Xvfb + xdotool**，不是 DOSBox 內建的 `-c` 參數（那個只能在容器啟動時跑
DOS shell 指令如 `mount`，跑不了遊戲內的按鍵互動）。`docker/dosbox/entrypoint.sh`
定義了一個 timeline 小語言，`;` 分隔以下四種步驟：

| 步驟 | 語法 | 說明 |
|---|---|---|
| 等待 | `wait:N` | 睡 N 秒（可以是小數，如 `wait:1.5`） |
| 送鍵 | `key:KEYSYM` | 送一個 X keysym（`Return`／`space`／`Escape`／`Up`／`Down`／`Left`／`Right`／單一字母如 `s`） |
| 打字 | `type:STRING` | 逐字送一段文字（`xdotool type`），不含 Enter |
| 截圖 | `shot:NAME` | 存成 `/shots/<NAME>.png` |

範例（本次任務實際用過、已驗證可行的 timeline）：

```
wait:1.5;shot:01-ega-opening;wait:2.5;shot:02-ega-mainmenu;key:Return;wait:1.5;shot:03-ega-ingame
```

```
wait:2.5;key:Return;wait:1.5;key:Down;wait:0.5;key:s;wait:1;shot:after-save
```

**技術關鍵**（沒抓到這點，鍵送了跟沒送一樣，見踩雷第二條）：

1. 沒有 window manager，`xdotool windowactivate` 一定失敗（缺
   `_NET_ACTIVE_WINDOW`）。要用 `xdotool windowfocus <winid>`（直接
   `XSetInputFocus`，不依賴 WM）。
2. 拿到焦點後，要用**全域** `xdotool key KEYSYM`（走 XTest，模擬真的鍵盤事件），
   不能用 `xdotool key --window <winid> KEYSYM`（那是 `XSendEvent` 送合成事件，
   SDL 1.2 預設會忽略 `send_event` 旗標為真的合成事件，鍵送了 DOSBox 完全沒反應，
   這個坑卡了本次任務最久，見踩雷第二條的完整排查過程）。

**已驗證可達成的自動化操作**（本次任務實測）：

- 開場畫面 → 主選單：`key:Return` 直接選中預設高亮的「Go adventuring」
- 進入遊戲後的行動選單：方向鍵（`Up`/`Down`/`Left`/`Right`）直接移動隊伍
  （不是選單游標移動——這個畫面沒有「先移動高亮再按 Enter 確認」的兩段式選單，
  方向鍵＝直接下達移動指令）
- 行動選單的功能靠**英文字首單鍵**觸發，不是方向鍵選單導航：`s` = Save Game
  （在最外層行動選單時）、`c` = Camp（進入紮營子選單）。**同一個按鍵在不同層級
  選單語意不同**（`s` 在外層是 Save Game，進了 Camp 子選單後 `s` 變成 Sleep）——
  寫自動化腳本時要注意目前在哪一層選單，不能死背按鍵。
- 存檔：`s` 觸發後畫面立即顯示「Game saved!」文字，**沒有確認對話框**，
  是最簡單可靠的一步操作，很適合當存檔 diff 實驗的收尾動作。

**未完全摸清的部分**（誠實記錄，不是「做不到」，是「這次任務沒有窮盡）：

- Camp 子選單（Party/Reorder/Sleep/Identify/Worship/Xorcise/View land/Trade/
  Drop/Equip/Use/Hunt/Cast/Quit）裡，試過的 `q`／`Escape`／`space` 都只會
  「關掉當前訊息回到 Camp 子選單」，沒有真的退回最外層的行動選單。`Sleep` 這個
  動作在本次測試中一直回報「You are restless」（可能是隊伍已滿血/滿魔導致睡不著，
  也可能有其他觸發條件），沒有實際確認「時間前進」的直接證據來自 Sleep——時間相關
  的證據改由「移動」間接取得（見下方存檔 diff 實驗）。真的要用 Camp/Sleep 需要
  再花時間摸，不在本次任務範圍內深挖。
- 沒有測試滑鼠自動化（`xdotool mousemove`/`click` 在容器裡指令本身能跑，但
  沒有找到這個遊戲有任何非用滑鼠不可的畫面，所以沒有必要性去驗證）。

## 踩到的雷（最有價值的部分）

### 1. CGA 模式在讀取 `GOT.FNT` 時卡死——素材缺檔，不是 DOSBox 的問題

**現象**：CGA 模式下，DEMON.EXE 顯示開場圖＋一個空的載入進度條方框後就永久停住，
不管等多久、送什麼鍵（`Return`/`space`/`Escape`/滑鼠點擊都試過）畫面完全沒反應，
但 CPU 使用率持續非零（~6-7%），代表**不是真的當機**，是在忙碌迴圈裡打轉。

**排查過程**：先懷疑是鍵盤事件沒送到（見雷 2），但確認鍵盤事件本身沒問題——用
`xdotool type 'demon'` 手動在 DOS 提示字元打指令能正常執行、能看到指令被執行的
畫面轉換，證明鍵盤管線是通的，問題出在遊戲執行到某個點之後就不再回應了。

裝了 `dosbox-debug`（Debian 有現成套件，`apt install dosbox-debug`），透過
`tmux new-session -d 'dosbox-debug -conf ... -debug'` 起一個背景 tmux session
跑內建除錯主控台（`tmux capture-pane` 讀輸出），拿到的 file-open 記錄檔一路是：

```
3642: EXEC:Execute \demon.int 0
3642: FILES:file open command 0 file \demon.int
358548: FILES:file open command 0 file \dem_data\got.fnt
```

之後就不再前進，永遠停在剛開完 `GOT.FNT` 這一行。查 `workplace/dosbox/game/DEM_DATA/`
發現：

```
$ ls DEM_DATA/ | grep -iE "\.FN[TE]$"
ASC.FNE
ASC.FNT
GOT.FNE
```

**`GOT.FNT`（CGA 版的裝飾字型檔）不存在，只有 EGA 版的 `GOT.FNE`**。`ASC.FNT`／
`ASC.FNE` 這組是成對的（CGA／EGA 都在），但 `GOT` 這組只有 EGA 版留下來，
CGA 版遺失了。用 `python3 -c "import zipfile; ..."` 直接查
`Demons Winter (1988).zip` 內的檔案清單也證實：`zipfile.namelist()` 裡確實
沒有 `GOT.FNT`，這是壓縮檔本身就缺的素材，不是複製過程漏掉。

DEMON.INT 對這個 `open` 失敗顯然沒有做容錯處理（可能是死迴圈重試，也可能是
拿到失敗的 file handle 後續操作觸發了未定義行為的忙碌迴圈，本次沒有再往下反組譯
確認確切的失敗處理邏輯，不在任務範圍內）。

**結論**：**CGA 模式在目前這份遊戲檔案下無法玩超過開場畫面**，這是素材完整性問題，
不是環境問題。這件事應該讓 PLAN.md／README.md 的所有人知道（本 agent 依邊界規則
不能動那兩個檔案，這裡只負責如實記錄，交給下一手決定要不要找補檔或標記
CGA 為「開場美術可用，互動不可用」）。EGA 模式完全不受影響，可以正常玩到主選單、
進遊戲、移動、存檔。

### 2. 送鍵沒反應：`windowactivate` 失敗 + `XSendEvent` 被 SDL 忽略

**現象**：一開始所有 `xdotool key --window <id> Return` 都送出去了（沒有錯誤訊息），
但畫面完全沒反應，怎麼等都是同一張截圖。第一次以為是「這是真的卡住的載入畫面」
（結果後來發現在 EGA 模式下這個懷疑是錯的，畫面其實只是在等鍵——這也是為什麼
第一條雷會先誤判成鍵盤問題,兩個雷互相干擾,拆解花了不少來回）。

**根因有兩層**：

1. `xdotool windowactivate` 在沒有 window manager 的 Xvfb 裡必定失敗：
   ```
   Your windowmanager claims not to support _NET_ACTIVE_WINDOW, so the attempt
   to activate the window was aborted.
   ```
   這個失敗**不影響後續指令繼續執行**（xdotool 只是印警告，exit code 還是會
   讓 shell 繼續跑），所以很容易忽略這個警告，以為視窗已經有焦點了，其實沒有。

2. 就算不理會 `windowactivate` 的失敗，改用 `xdotool key --window <id> KEYSYM`
   （用 `XSendEvent` 把事件直接送到指定視窗，理論上不需要焦點），SDL 1.2（DOSBox
   0.74 用 SDL 1.2）預設會**忽略帶 `send_event=True` 旗標的合成事件**——這是
   SDL 的安全設計，避免其他程式偽造鍵盤事件。結果就是鍵「送了」，DOSBox 收到了
   X11 event，但直接丟棄。

**解法**：改用 `xdotool windowfocus <id>`（直接呼叫 `XSetInputFocus`，不透過
window manager，沒有 WM 也能成功）把輸入焦點設到 DOSBox 視窗，然後用**全域**的
`xdotool key KEYSYM`（不帶 `--window`，走 XTest 擴充，模擬真正的硬體鍵盤事件，
送到目前有焦點的視窗）。這條路徑下鍵盤事件是「真的」事件，SDL 不會過濾掉。

驗證方式：在 DOS 提示字元手動打 `xdotool type 'demon'` + `xdotool key Return`，
親眼看到畫面從文字提示字元轉成遊戲開場圖，確認這條路徑真的有效之後，才回頭
確認雷 1（CGA 卡住）是素材問題不是這條雷的殘留影響。

### 3. `cycles=auto` vs `cycles=fixed`：本專案選 fixed 4000

DOSBox 的 `[cpu] cycles=auto` 會依偵測到的「遊戲需要多少速度」動態調整模擬速度，
對一般玩家很方便，但對**要精準控制 timeline 等待秒數的自動化腳本不利**——
同一份 timeline 在不同次執行可能因為 auto 判斷結果不同而讓某個 `wait:N` 秒數
提早或延後對上該有的畫面（本次任務實測過：同一支 `wait:1;shot:xxx` 有時候截到
全黑畫面、有時候截到正確畫面，就是這個原因，不是腳本邏輯錯）。

改用 `cycles=fixed 4000`（固定模擬速度，數值上大約對應一台偏快的 286/386 等級），
拿掉這個變因後，同一份 timeline 重跑的截圖時機才穩定可重現。`fixed 4000` 這個
數字不是精算出來的「正確」值，是實測夠用的經驗值，可以用 `dosbox_run.sh` 第三個
參數覆蓋。

### 4. 開場畫面轉場很快，`wait` 秒數卡在很窄的窗口內

即使固定 cycles，「開場美術畫面」到「自動跳到主選單」這段轉場本身就很快
（原本 1988 年放這張圖是為了掩護軟碟機讀取 173KB 主程式的時間，見
`docs/research/ssi-engine-architecture.md`；在容器裡讀取本機檔案系統幾乎瞬間完成，
所以這張「用來拖時間的圖」幾乎是一閃即逝）。實測：

- `wait:0.3` → 畫面全黑（DOSBox 視窗還沒切到遊戲畫面模式）
- `wait:1.5` → 穩定截到開場美術
- `wait:2` 之後 → 已經跳到主選單

要截到開場美術，`wait` 要抓在 1～1.5 秒這個窄窗口內；再晚一點點就直接錯過，
只能截到主選單。這個時間窗會隨 `cycles` 設定值變動，改了 cycles 記得重新校準。

### 5. 遊戲資料要用可寫副本，不能唯讀掛載

第一輪測試把 `workplace/orig/demwin/` 直接以 `:ro` 唯讀掛進容器，懷疑過這是
CGA 卡住的原因（猜測遊戲開機時要寫暫存檔）。後來證實跟這個無關（雷 1 的真正
原因是缺檔），但**存檔功能本身確實需要可寫**——`Save Game` 要把新內容寫回
`PARTY.DAT`，唯讀掛載下這個操作會靜默失敗或報錯。`tools/dosbox_run.sh` 固定用
可寫的 `workplace/dosbox/game/` bind mount（rw），不是唯讀掛 `workplace/orig/`。

## 存檔動態 diff 實驗（stretch goal，已完成）

### 實驗設計

目標：`docs/formats/game-data-tables.md` 第 5 節「未解欄位清單」列出「隊伍位置
與朝向」「遊戲內時間」兩個尚未解出的 `PARTY.DAT` trailer 欄位，該文件建議的下一步
正是「DOSBox 內移動隊伍朝不同方向後存檔 diff」，本次任務就是照這個建議做。

流程：

1. 用 `workplace/orig/demwin/DEM_DATA/PARTY.DAT`（專案內建存檔，5 名角色：
   Wopple/Stumpy/Podgom/Worman/Menhir，金幣 65）當基準，每次實驗前重置
   `workplace/dosbox/game/DEM_DATA/PARTY.DAT` 回這份基準，確保每次都是同一個
   起始狀態，排除「上一次實驗的殘留變化」污染下一次結果。
2. 用 `tools/dosbox_run.sh ega "wait:2.5;key:Return;wait:1.5;key:<方向>;wait:0.5;key:s;wait:1"`
   進遊戲、往一個方向移動一步、存檔。
3. 把存完的 `PARTY.DAT`複製出來，用 `python3 tools/parse_party.py --diff <基準> <實驗後>`
   （專案既有工具，直接拿來用）比對。
4. 對照組：完全不移動、直接存檔，確認「什麼都不做的存檔」是否 byte-for-byte
   跟基準一致（驗證存檔本身是確定性的，排除「隨機雜訊」干擾後續判讀）。
5. 分別對 `Down`／`Up`／`Left` 三個方向各做一次，比較哪些 byte 隨方向變化、
   哪些不變、變化量多少，交叉定位語意。

實驗產出檔案在 `workplace/dosbox/saves/`（不入版控）：

| 檔名 | 內容 |
|---|---|
| `PARTY.baseline.dat` | 未經任何操作的原始存檔（= `workplace/orig/demwin/DEM_DATA/PARTY.DAT`） |
| `PARTY.control-nochange.dat` | 進遊戲後立刻存檔、不做任何操作 |
| `PARTY.after-move-down.dat` | 進遊戲、按一次 `Down`、存檔 |
| `PARTY.after-move-up.dat` | 進遊戲、按一次 `Up`、存檔 |
| `PARTY.after-move-left.dat` | 進遊戲、按一次 `Left`、存檔 |

### 結果

**對照組（不操作直接存檔）**：`PARTY.control-nochange.dat` 跟 `PARTY.baseline.dat`
**完全 byte-for-byte 相同**（`diff` 無輸出）。證實存檔動作本身是確定性的，
沒有隨機雜訊或時間戳記混進檔案，後續 diff 出來的每一個 byte 差異都可以放心
歸因給「移動」這個單一動作。

**三個方向分別 diff 基準檔**（`tools/parse_party.py --diff` 輸出）：

```
# Down
abs 0x05b0 (trailer rel=0x09c): 56(0x38) -> 55(0x37)
abs 0x05b4 (trailer rel=0x0a0):  1(0x01) ->  2(0x02)
abs 0x05b6 (trailer rel=0x0a2): 32(0x20) -> 33(0x21)

# Up
abs 0x05b0 (trailer rel=0x09c): 56(0x38) -> 55(0x37)
abs 0x05b4 (trailer rel=0x0a0):  1(0x01) ->  2(0x02)
abs 0x05b6 (trailer rel=0x0a2): 32(0x20) -> 31(0x1f)
abs 0x05b8 (trailer rel=0x0a4):  2(0x02) ->  0(0x00)

# Left
abs 0x05b0 (trailer rel=0x09c): 56(0x38) -> 55(0x37)
abs 0x05b4 (trailer rel=0x0a0):  1(0x01) ->  2(0x02)
abs 0x05b5 (trailer rel=0x0a1):  9(0x09) ->  8(0x08)
abs 0x05b8 (trailer rel=0x0a4):  2(0x02) ->  3(0x03)
```

### 解讀

| trailer 相對位移 | 絕對位址 | 三方向行為 | 判讀 | 信心 |
|---|---|---|---|---|
| `0x0a2` | `0x5b6` | Down +1／Up -1／Left 不變 | **Y 座標**（螢幕座標系，向下為正） | 高——三方向行為完全符合「垂直座標」的預期模式 |
| `0x0a1` | `0x5b5` | Left -1／Down、Up 不變 | **X 座標**（水平座標，向左為負） | 高——只有左右移動才動這個 byte，跟 Y 座標的行為互斥，符合兩個獨立座標軸的預期 |
| `0x0a4` | `0x5b8` | 基準值 2；Down 不變（=2）／Up 變 0／Left 變 3 | **隊伍朝向（facing）**，推測編碼 `0=北 1=東 2=南 3=西`（順時針） | 中高——三筆資料完美對應一個順時針四方位 enum：基準朝南（2），按 Down（往南走，跟目前朝向一致）不用轉向所以不變；按 Up（往北走，跟朝向相反）要轉向，變成 0＝北；按 Left（往西走）變成 3＝西。三個值剛好落在同一個 enum 上，不像巧合，但這份存檔只有一個角色朝向可測，还没验证「已经朝北时再按 Down 会不会变回 2」这种反向确认，留給後續驗證 |
| `0x09c` | `0x5b0` | 三方向都 -1 | 未定，候選：移動消耗的資源（口糧／體力），或某種與方向無關的計次 | 低——只知道「每移動一步就 -1」，還沒有測試「移動很多步之後這個值會不會歸零觸發事件」之類的邊界行為，不確定是不是 `docs/formats/game-data-tables.md` 想找的「遊戲內時間」欄位本身，也可能是它的關聯欄位 |
| `0x0a0` | `0x5b4` | 三方向都 +1 | 未定，候選：**遊戲內時間／回合數計數器**（跟 `0x09c` 反向連動，很符合「時間前進、口糧減少」的典型 CRPG 機制） | 中——跟 `docs/formats/game-data-tables.md` 第 6 節建議「鎖定會規律遞增的 byte」的描述完全吻合，是目前對「遊戲內時間」最有力的候選，但只驗證了「移動會讓它 +1」，還沒有驗證它是否對應遊戲內實際顯示的時/日/月數字（這個存檔畫面沒有看到時鐘/日期的 UI 顯示，需要進一步找有沒有查看時間的畫面來對照） |

**與現有文件的關係**：`docs/formats/game-data-tables.md` 第 5 節「未解欄位清單」
明確列了「隊伍位置與朝向」（只找到疑似隊形順序表，朝向完全沒候選）跟「遊戲內時間」
（原本懷疑的欄位已被推翻）這兩項，第 6 節建議的下一步正是「DOSBox 內移動隊伍
朝不同方向後存檔 diff」「鎖定會規律遞增的 byte」。本次實驗的 `0x5b5`/`0x5b6`（X/Y
座標）與 `0x5b8`（朝向）三個欄位可視為這兩項未解欄位裡「朝向與位置」子項的**初步
解答**；`0x5b0`/`0x5b4` 這對反向連動的計數器是「遊戲內時間」的**強力候選**但尚未
最終驗證。這幾個欄位目前只記錄在本檔案（`docs/re/01-dosbox-reference.md`），
還沒有寫回 `docs/formats/game-data-tables.md`——依任務邊界規則，本 agent
不能修改 `docs/formats/`，這幾個發現需要另外請負責維護該文件的人或後續任務
接手核實、寫入正式欄位表。

### 如何重跑這個實驗

```bash
cd /home/anr2/cht/daemon_winter

# 重置成乾淨基準
cp workplace/orig/demwin/DEM_DATA/PARTY.DAT workplace/dosbox/game/DEM_DATA/PARTY.DAT

# 做一個單一變數操作（這裡示範往右移動一步）
./tools/dosbox_run.sh ega "wait:2.5;key:Return;wait:1.5;key:Right;wait:0.5;key:s;wait:1"

# 比對
cp workplace/dosbox/game/DEM_DATA/PARTY.DAT /tmp/party-after-right.dat
python3 tools/parse_party.py --diff workplace/orig/demwin/DEM_DATA/PARTY.DAT /tmp/party-after-right.dat
```

## 這個環境能不能支撐後續逐機制對拍驗證？

**能，但目前只有 EGA 模式能撐到「開機 → 選單 → 進遊戲 → 互動 → 存檔」全鏈路**，
CGA 模式受限於缺檔（`GOT.FNT`）只能驗證到開場美術，互動邏輯測不了，這點會限制
「兩種模式都要能跑」這個目標裡 CGA 那一半的對拍能力——CGA 美術素材本身（`.PIC`/
`.SHP`/`.FNT` 裡有的部分）可以拿這裡的環境截圖對，但 CGA 版的**遊戲邏輯**行為
（戰鬥計算、事件觸發等）目前拿不到 CGA 模式下的真實對照組，只能退而求其次拿
EGA 模式的行為結果當邏輯層的 oracle（合理，因為兩個模式共用同一份 `DEMON.INT`
引擎，圖形素材不同但邏輯應該一致，除非邏輯本身有依顯示卡分支的特例）。

自動化輸入這條路線（Xvfb + xdotool windowfocus + 全域 key）已經驗證足夠支撐
「寫一支 timeline 腳本、無人值守跑一輪、拿到截圖／存檔」這個工作模式，
可以重複用在後續任何「這個機制在原版怎麼表現」的問題上——寫法就是本檔案
「自動化輸入」一節說明的 `wait`/`key`/`type`/`shot` 四步驟語言，不需要每次
重新摸索 X11／SDL 的坑。
