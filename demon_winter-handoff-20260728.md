# 交接：2026-07-28（A6 前期垂直切片與後期抽樣完成）

接手的第一件事是讀 [`CONTEXT.md`](CONTEXT.md) —— 狀態的單一真相來源在那裡，
這份只補「上一輪做了什麼、哪裡踩過坑、下一步從哪個指令開始」。

交接檔按日期各存一份（`demon_winter-handoff-<YYYYMMDD>.md`），不覆蓋舊的 ——
坑與指令會隨版本漂，看得出「這是哪一天的說法」才判斷得出還適不適用。

`main` 的最後一個 commit 是 `b5a5d23`。工作樹乾淨、下面四道檢查全綠。

---

## 1. 起手式：確認樹是綠的

```bash
tools/go.sh build ./...        # 一律走 docker，不要在系統裝 Go
tools/go.sh vet ./...
tools/go.sh test ./...                    # 13 個套件
tools/go.sh run ./cmd/dwstrings check     # 資料字串 500/500
tools/go.sh run ./cmd/dwstrings uicheck   # 目前介面文案 691/691、硬編 0
```

四道都過才動手。`uicheck` 不是可選的：**打錯一個 key 的後果是靜默退回
fallback**，畫面一模一樣，只是那一條永遠不走目錄 —— 它第一次跑就抓到 51 條
已經在那裡的。

---

## 2. 上一輪做完的（`353a09a`..`b5a5d23`）

| commit | 內容 |
|---|---|
| `353a09a` | **D1 收尾**：介面文案 677 條全走目錄、硬編 0。`cmd/dwstrings uicheck` 四項檢查 ＋ `ui:dynamic` 標記（算出來的 key）|
| `8956040` | **世界地圖邊界換圖接上**（`world.CrossEdge` 原本零呼叫端）＋ `dwroute` 擴成跨子地圖 |
| `984128d` | **施法職業的初始法力**：原版是單一個 `職業 > 2` 的比較（連盜賊都有），不是只有巫師 |
| `ad4981a` | **建角要把人放進陣型**（`Formation.AddMember` 原本零呼叫端）—— 少這一步，新遊戲的隊伍每一場遭遇都只有怪物 |
| `974c098` | `playthrough.sh` 的按鍵改成 keydown→keyup（`xdotool key` 會漏鍵）|
| `a3055ee` | **A6 開場三段跑通**：建角 → 買裝備 → 分裝備 → 裝上 |

三個修法有同一個形狀，值得記住：**函式解完、單元測試寫好、逆向文件也有，
就是沒有人叫它**。`grep -rn <函式名> --include=*.go .` 回三行、全在自己的檔案裡。
那一類缺口**不會出現在任何待辦清單上**（清單是照「解了沒」列的），
而且只有實跑抓得到，還要**看狀態數字不是看畫面** —— 三次都是靠
「金幣／血量／座標／地圖編號不動」露出來的。詳見
[`docs/playtest/10`](docs/playtest/10-world-edge-was-never-wired.md)、
[`11`](docs/playtest/11-a6-leg1-and-the-silent-zero-sp.md)、
[`12`](docs/playtest/12-empty-battles-and-the-ship-you-already-own.md)。

---

## 3. A6 現在到哪裡

### 跑得起來的三段

```bash
# 第一段：建五個角色 → 8 步到海濱鎮 → 買 5 把匕首 ＋ 1 件布甲（75 → 60 金）
tools/playthrough.sh tools/playthrough/a6-leg1.txt workplace/dump/a6 \
    -newgame -autofight -seed=11

# 第二段：紮營 T 交出，四把匕首分給 b–e（吃上一段的存檔）
rm -rf workplace/dump/a6b && cp -r workplace/dump/a6 workplace/dump/a6b
rm -f workplace/dump/a6b/*.png
tools/playthrough.sh tools/playthrough/a6-leg2-share.txt workplace/dump/a6b \
    -autofight -seed=12

# 第三段：紮營 E 換裝，五個人全部裝上
rm -rf workplace/dump/a6c && cp -r workplace/dump/a6b workplace/dump/a6c
rm -f workplace/dump/a6c/*.png
tools/playthrough.sh tools/playthrough/a6-leg3-equip.txt workplace/dump/a6c \
    -autofight -seed=13
```

⚠ **`workplace/` 是 gitignore 的**，所以上一輪跑出來的存檔不在 repo 裡。
接手時要**重跑這三段**（約四分鐘）才會有 `workplace/dump/a6c/PARTY.DAT`
當後續的起點。三段都是可重跑的，不是一次性的手打指令。

### 驗收用存檔直讀，不要判讀截圖

```python
d = open('workplace/dump/a6c/PARTY.DAT', 'rb').read()
REC, TR = 0x104, 5 * 0x104
print("金幣", int.from_bytes(d[TR+0x0a:TR+0x0e], 'little'),
      "陣型", list(d[TR:TR+9]), "人數", d[TR+0x9a])
for i, n in enumerate("abcde"):
    rec = d[i*REC:(i+1)*REC]
    print(n, "背包型別", [rec[0x0c+17*s] for s in range(10)],
          "武器格", rec[0x100], "護甲格", rec[0x101])
```

跑完三段應該看到：

```
金幣 60　陣型 [0,1,2,3,4,255,255,255,255]　人數 5
a 背包型別 [255,255,255,255,0,8,255,…] 武器格 4 護甲格 5
b 背包型別 [0,255,…]                    武器格 0 護甲格 255
c／d／e 同 b
```

背包型別是 `ITEMS.DAT` 的索引：`0` 匕首、`8` 布甲、`255` 空格
（`tools/parse_items.py workplace/orig/demwin/DEM_DATA/ITEMS.DAT` 印全表）。

### 下一段要解的第一個問題

> **同日接手後訂正：本節的「36 步拿船」前提錯了。**
> `EXITS.DAT` 的 `(25,48) → 1:(20,48)` 來源格實際由世界子地圖 **64**
> 認領，不是起點子地圖 34。從 34 實跑到 `(25,48)` 不會換圖；
> `dwroute -world-reach -from 34:31,45` 也確認開局陸路只到
> 34／2／44／1。詳見 `docs/playtest/12` §4 的訂正。

新遊戲**就有一艘船**（地圖 1 的 (13,51)，船體 67），從起點 36 步走得到：

| 步驟 | 座標 | 步數 |
|---|---|---|
| 起點 | 34:(28,50) | — |
| 地圖 1 的入口 | 34:(25,48) | 5 |
| 換圖落點 | 1:(20,48) | — |
| 船邊的陸地 | 1:(13,52) | 31 |
| 往北一步上船 | 1:(13,51) | 1 |

**未解**：那艘船停在地圖 1 的水域，而地圖 1 不是世界子地圖（編號 < 11），
邊界換圖對它不適用 —— 從那片水域怎麼開到世界地圖上？要找一個海路走得到的
出口格。這是 A6 續跑的第一道題。

之後是**練到全隊 120 點法力**（解咒 ×3 各 50、禁錮 100；1 級全隊只有 32）。
升級要找城鎮公會，最近的在東邊的厄加德。升級 SP 是 `max(兩次 Roll(智力/2+1))`。

路線一律用工具產生，不要手打：

```bash
tools/go.sh run ./cmd/dwroute -world -from 34:28,50 -to 44:34,11   # 跨子地圖
tools/go.sh run ./cmd/dwroute -world-reach -from 34:28,50 -sailing # 有船到得了哪
tools/go.sh run ./cmd/dwroute -map 2 -list-sites                   # 某張圖上的地點
```

---

## 4. 會浪費時間的坑（都踩過了）

**腳本／harness**

1. **踏進城鎮那一步不能用 `rep`。** `rep` 每步之後都 `settle`，而 `settle`
   的白名單只有「野外」，其餘一律送 Escape —— 一進城就被踢出來，
   下一個鍵落到世界地圖上（`m` 變成推家具）。用 `key`。
2. **每一段都要先關掉標題畫面**（`key Return` ＋ `wait 1`）。從存檔接續
   一樣會先進標題，第一個按鍵會被它吃掉。
3. **裝上成功之後畫面自己關回紮營選單，不要再送 Escape。** 多送那一下會
   收掉紮營，而 `s` 在紮營選單裡是**睡覺**不是存檔 —— 於是截圖看得到
   「a 換上匕首」，存檔裡卻什麼都沒裝上。
4. **交出道具時收方游標預設指向「下一個人」**，給第二個人是 0 次 Down；
   而且**交出之後那一格會變空、剩下的不往前補**，第 k 次要選第 k 格。
5. **數字鍵是 1-based**（`pressedDigit` 把 1–9 對到 0–8，`0` 回傳 9）。
   寫 `key 0` 想選第一項會靜默不動作，後面整串錯位。
6. **跨模態選單的長腳本不要一次寫完。** 拆段、每段結尾存檔、下段從存檔接續，
   錯的時候才不會連鎖。

**規劃工具**

7. **船停在水上，而水在可通行表裡是擋住的。** 拿船的座標餵 `dwroute -to`
   永遠說走不到 —— 上船靠 `Boardable` 這個例外。要問它旁邊那格陸地。

**背景作業**

8. `pkill -f playthrough.sh` 會把**自己這個 shell** 一起殺掉（命令列含同一字串）。
   停背景作業請用 `kill <pid>`，並且**只** `docker stop` 自己那個
   `dw-playthrough-<pid>` 容器。

---

## 5. 建議的順序，各附第一個動作

| 順位 | 項目 | 第一個動作 |
|---|---|---|
| 1 | **F2** 地圖／戰場雙線白框 ＋ 拿掉戰場格線 ＋ EGA 配色分區 | 讀 `docs/ui/02-ui-plan.md` §4 階段 2 與 `01` §3.5 的配色表。它修的是**功能性**問題：視野內縮時黑格與背景同色，邊界讀不出來 |
| 2 | **A6 續跑** | 先解 §3 的「從地圖 1 的水域開出去」。`dwroute -map 1 -list-sites` 找海路走得到的出口格 |
| 3 | **E4** headless 確定性回歸測試 | `tools/` 下目前沒有任何 golden／regression。這一輪三個缺口全是實跑抓到的，而實跑沒有自動比對 —— 把 §3 那段 `PARTY.DAT` 斷言變成腳本就是第一步 |
| 4 | **C4** 移動操作與原版不同 | 引擎是「方向鍵＝轉向並前進」，原版是相對轉向 ＋ Return 前進。影響「背後攻擊 +3」與船隻轉向的移動點。**要嘛寫進差異表要嘛改回去**，別放著 |
| 後 | **C3** 海戰 | 全專案零實作，量最大、收益最低（不擋破關）。手冊規則很細：移動點、砲擊 1–10、離場點逃跑、沉船全滅 |

F3–F6、C1／C5–C10／C13、E1–E3 的逐項現況見 `CONTEXT.md` §7。

---

## 6. 一律不要做

- **不要碰共用的 docker 資源。** 這台機器同時放著多個客戶專案的 image／volume。
  只能 `docker stop`／`docker rm` 自己這次建立的容器。
  `docker image prune`／`system prune`／`volume prune`／`builder prune`／`rmi`／
  `container prune`／`network prune` **一律禁止**，不論看起來多安全。
  （2026-07-27 有過事故：一個驗收 subagent「順手」清 dangling image，
  連帶刪掉另外七個客戶專案的 image。）
- **不要在系統裝 Go 或 pip install。** 編譯一律 `tools/go.sh`，Python 一律 docker uv.venv。
- `workplace/orig/` 唯讀，原版執行檔／資料／美術／音樂一律不散布、不入版控。
- **不要重造已經有的實作。** 一條規則只留一份 —— 為了「看一眼」重抄的那份
  一定會漂，症狀是看起來合理的錯數字。動手前先 `grep`。
- 不要 force push 到 `main`，不要跳過 hook。
- 規劃／設計文件寫成 repo 裡的 markdown，不要放 Claude Artifact。

---

## 7. 同日續跑結果（交付前狀態）

使用者同意把驗收範圍定為「前期完整垂直切片＋後期高風險抽樣」，
不逐格重跑所有大型地城。正常流程已實跑至加穆爾神殿：

- 狗頭人營地首領與三批守軍、正常 EXP／金幣結算
- 死亡、紮營不復活、返城復活與治療
- 四名角色在厄加德升至 2 級
- 艾巴拉特補滿升級後生命上限
- 穿越 34 → 1 → 34，依疲勞規則紮營後抵達 Gamur 神殿

這段新增／修正：

- `DATA1..5.TXT` 隨地城編號切換；`-events` 只保留為偵錯覆寫
- `Character.CombatUnit` 帶入死亡狀態，HP 0 寫回與休息皆強制視為死亡
- `dwroute -world` 遇 `EXITS.DAT` 傳送時不再漏掉觸發方向鍵
- `tools/playthrough/` 新增狗頭人營地、治療、升級、Gamur 路線
- 倚天 16×15 預設橫向粗體（`-eten-bold=true`）
- 實機錄影與本機宣傳片流程在 `tools/promo/`

後期抽樣已看過：

- 新格里昂成功購船：1000 金 → 390 金，船體 75/75
- 神殿門房英文密語輸入畫面
- 中文結局長文
- 既有 `docs/re/59`–`64` 的三符印、光之環、禁錮與結局分段證據

宣傳片輸出位於 gitignore 的
`workplace/promo/out/demon-winter-cht-promo.mp4`（61 秒，1280×720，
實機畫面與原版 PC Speaker 音效；未公開上傳）。
