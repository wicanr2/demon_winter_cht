# 27 — FILES.DTT 的切表常式（分段邊界的硬證據）

追「月份名稱在哪」時撞到的東西。原本只想找 22 個月名，結果找到的是
**整個 FILES.DTT 怎麼被切成 8 張表**的那段程式碼 —— 分段邊界從此不用內容比對去猜。

---

## 1. 問題：分段是猜的

`FILES.DTT` 是 501 條 NUL 分隔的字串，沒有表頭、沒有長度前綴、沒有位移表。
本專案原本靠「內容看起來像什麼」把它切成種族名、法術名、技能名、武器名…，
其中兩段猜錯了（見 §5）。

---

## 2. 切表常式（`DEMON.INT 0x11d5d`）

整個 blob 的遠指標存在 `ds:0x54e8`。常式從那裡出發，一路掃 NUL、
把每個字串的位址存進一連串遠指標陣列：

```
11d78  les bx,[bp-6] / cmp byte es:[bx],0   ; 目前這個 byte 是 NUL 嗎
11d7f  jz  收下一條                          ; 是 → 下一條字串的起點
11d81  inc word [bp-6]                       ; 不是 → 往前一個 byte
...
11d9b  mov [bx+0x54ea],dx                    ; 存遠指標（段）
11d9f  mov [bx+0x54e8],ax                    ;         （偏移）
11da3  cmp word [bp-2],5 / jl 回圈           ; ← 這張表有幾條
```

同樣的形狀重複 8 次，只有**目的地基底**與**筆數**不同。筆數就寫在
每個迴圈末尾的 `cmp` 常數裡：

| # | 目的地 | 筆數 | FILES.DTT 索引 | 內容 |
|---|---|---|---|---|
| 1 | `ds:0x54e8` | 5 | 0–4 | 種族名 |
| 2 | `ds:0x4ccc` | 86 | 5–90 | 43 組（法術名, 命中訊息）|
| 3 | `ds:0x5538` | 32 | 91–122 | 技能名 |
| 4 | `ds:0x4e38` | 30 | 123–152 | **道具型別名** |
| 5 | `ds:0x4c98` | 11 | 153–163 | **神祇名** |
| 6 | `ds:0x55b8` | 303 | 164–466 | 場景／物件互動文字 |
| 7 | `ds:0x50f2` | 22 | 467–488 | **月份名** |
| 8 | （下一張）| 12 | 489–500 | 幻術／召喚生物名 |

**5 + 86 + 32 + 30 + 11 + 303 + 22 + 12 = 501** —— 與檔案裡的字串總數一個不差。
分段從此是原版程式碼直說的，不是推測。

> 第 1 張表的來源指標與目的地是同一個位址：`0x54e8[0]` 本來就是整個 blob 的
> 起點（也就是第 0 條字串），迴圈從索引 1 開始填。不是筆誤。

---

## 3. 月份名（`ds:0x50f2`，第 467–488 條）

讀取端在狀態列（`0x70ac`）：

```
70ac  mov al,es:[bx+0x9d]      ; 隊伍欄位 +0x9d = 月
70b3  shl ax,1 / shl ax,1      ; ×4（遠指標）
70b9  push [bx+0x50f4] / push [bx+0x50f2]
70c5  mov al,es:[bx+0x9e]      ; 日
70cd  mov al,es:[bx+0x9f]      ; 時
70d6  push DS:0x618            ; "Hour %d, Day %d in the Month of the %s"
```

22 個名字：

```
Ruby  Ebony  Gold  Comet  Spirit  Dragon  Rose  Sword  Unicorn  Metal  Lotus
Axe  Panther  Ice  Mandrake  Aurora  Onyx  Phoenix  Wind  Jade  Fire  Hyacinth
```

所以 **`+0x9d` 是 0-based 的名稱索引，不是序數**。原版存檔的 0 是合法值。
時間規格的其餘細節（含原版在月 = 22 的 off-by-one）見 `docs/spec/06-time.md`。

譯文在 `assets/lang/zh-Hant/months.txt`，目錄 key 是 `MONTHS`。

---

## 4. 神祇名（`ds:0x4c98`，第 153–163 條）

```
Omizeh  Balmur  Gamur  Vemarkn  Acisc  Maldorath  Volobews  Illo  Theryni
Camear  Ancient
```

讀取端 `0x4bea`：

```
4bea  mov al,es:[bx+si+0xf0]   ; 角色記錄 +0xf0
4bf1  dec ax                   ; 1-based → 0-based
4bf5  shl ax,1 / shl ax,1
4bfb  push [bx+0x4c9a] / push [bx+0x4c98]
```

`+0xf0` 是神祇編號，**0 代表沒有**（原版起始隊伍五個人都是 0）。
Maldorath 正是本作的魔王名，對得上。

**還沒定案的是 `+0xf0` 到底是「長期信仰」還是「當前生效的祝福」**：

- 神殿那一支（`0x1c53e`）寫入編號時同時把 `+0xeb` 設成 20，看起來像持續回合數
- 睡覺的夢境段（`0x3f0d`）每晚把全隊的 `+0xf0` 清成 0

信仰不該過夜就沒了，所以偏向後者。兩者都還沒實作。

> `docs/re/09` 曾把 `0x4c98/0x4c9a` 標成「施法者名稱指標表」。
> 那張表只有 11 條、而且讀取端拿的是 `+0xf0`，不是隊員索引 —— 該處的說法要修。

---

## 5. 修掉兩個猜錯的邊界

| 原本 | 現在 | 為什麼原本會錯 |
|---|---|---|
| 「15 個裝備類別」+「13 個神器」= 136–150、151–163 | 一張 30 條的**道具型別名**表（123–152）＋ 11 條神祇名（153–163）| `Demon Crystal`／`Orb/Evertime` 看起來像劇情神器，就被切成另一段。實際上它們是 ITEMS.DAT 的型別 28、29 —— 那張表剛好 30 筆，與這裡的 30 對上 |
| 467–488 是「隨機物品命名詞庫」（Comet Sword 之類）| **月份名** | 名字本身確實像形容詞，但沒去找讀取端。找到讀取端就結案了 |

第二條是 `rulebook/62`「找讀取端」的又一次驗證：
**只看內容永遠只能得到「像什麼」，找到讀取端才知道「是什麼」。**

---

## 6. 本專案的實作範圍

- `internal/assets/gamedata/strings.go`：分段 accessor 依本文更正，
  新增 `ItemTypeNames`／`DeityNames`／`MonthNames`
- `internal/game/clock.go`：月改成 0-based 名稱索引，`ClockAt` 從存檔還原時間
- `cmd/demonwinter`：狀態列與紮營畫面顯示月份名
- `cmd/dwstrings months`：抽月份名成翻譯目錄

**沒實作**：神祇／祈禱系統（`+0xf0`／`+0xeb`）、303 條場景互動文字那張表
（事件敘述走的是 `DATA*.TXT`，這張表是另一條路，還沒對上用途）。
