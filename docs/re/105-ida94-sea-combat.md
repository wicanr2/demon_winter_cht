# IDA 9.4：海戰主迴圈、單位結構與結算

日期：2026-07-28  
輸入：`DEMON.INT`，SHA-256
`fc1df05513bfa0f1a38f95ce0fbe5e6ec390c8192b2f99b3b6118b3c23868ea5`  
工具：`/home/anr2/ida_94_official/dist` 的 `ida-pro-9.4-ver2`，
16-bit DOS loader，headless `idat -A -B`

這份筆記取代 `docs/re/75`、`78` 對 `1990:2b27` 的暫時判讀。
Ghidra 以 overlay segment 顯示 `1990:2b27`；IDA 把同一模組攤平成
`sub_1C427`。下列位址一律是 IDA database 的 linear address。

## 1. 可重現方式

原檔保持唯讀，只分析副本：

```sh
mkdir -p workplace/ida
cp --reflink=auto workplace/orig/demwin/DEMON.INT workplace/ida/DEMON.INT
sha256sum workplace/ida/DEMON.INT
docker run --rm \
  -v "$PWD/workplace/ida:/work" -w /work \
  ida-pro-9.4-ver2 idat -A -B DEMON.INT
```

產物 `workplace/ida/DEMON.INT.i64`、`.asm` 已由 `.gitignore` 排除；
原版 binary、IDB 與完整反組譯不進版控。

## 2. 呼叫圖

| IDA 位址 | 作用 | 直接證據 |
|---|---|---|
| `sub_1C427` | 建立海戰、配置單位、主回合與結算 | 唯一 caller `sub_17C5D+5D6`；初始化 `(13,13)`、載入 `SHIP` 資產、最後寫回船體與遭遇倒數 |
| `sub_1CB84` | 非玩家船隻 AI | 追蹤同行／同列目標、逼近、砲擊或近戰，扣移動點 |
| `sub_1D0B5` | 相鄰接觸處理 | 只由 AI 的迎面接觸路徑呼叫 |
| `sub_1D1F6` | 把指定單位設為畫面中心 | 主迴圈每名存活單位行動前呼叫 |
| `sub_1D299` | 砲擊／傷害處理 | 兩個 caller：AI 與玩家指令；均寫 `unit+0x0e` 的船體 |
| `sub_1D6A0` | 玩家海戰輸入 | 15-case key switch；移動、砲擊、檢視、結束回合 |

`sub_1C427` 不是一般戰鬥的變體：它使用另一組 16-byte 單位記錄、
另一個輸入函式與 `SHIP.SHP/.SHE`。

## 3. 海戰單位記錄

基底 `ds:5306`，每筆 16 bytes（索引先 `shl ax,4`）：

| offset | 語意 | 證據 |
|---:|---|---|
| `+0x00` | X | 玩家初始化為 13；移動與鏡頭讀寫 |
| `+0x02` | Y | 玩家初始化為 13；移動與鏡頭讀寫 |
| `+0x04` | 面向／船圖方向 | 玩家從 party `+0xa4` 載入；畫面索引為 `facing*2+0x2c` |
| `+0x06` | 本回合移動點 | 玩家固定初始化為 6；行動前複製到 `ds:5190`，每項操作遞減 |
| `+0x08` | 命中率 | 玩家固定寫 85；`Roll(100) <= value` 才命中 |
| `+0x0a` | 攻擊骰面／類型參數 | 玩家固定 −10；`Roll` 對負值取絕對值，因此是 1–10 |
| `+0x0c` | 防禦減傷 | 傷害路徑從攻擊骰結果扣除此欄，低於 1 判為未造成傷害 |
| `+0x0e` | 船體 | 玩家從目前船記錄 `+0x25` 載入；所有傷害在此扣除 |

玩家船移動點在 `1C472` 寫 6；其餘初始化證據在 `1C7A9–1C813`：
位置 `(13,13)`、命中 85、攻擊 −10、防禦 0，船體取
`partyShip[boatIndex*6+0x25]`。

## 4. 回合、移動、砲擊

- `sub_1D6A0` 的按鍵 switch 有 15 case；方向選擇與手冊的
  `I/J/K/M` 四向砲擊一致。
- `sub_1CB84:1CD27` 在攻擊前檢查移動點至少 3，然後
  `add ds:5190,-3`。
- `sub_1CB84:1CE3C` 的轉向路徑要求至少 2，寫回面向後連續遞減兩次。
- 前進、轉向、倒航成本 1／2／3 也由紙本手冊直接給出；remake 將這些
  成本釘在 `internal/game/seabattle_test.go`。
- 命中流程在 `1CD4D–1CD8F`：`Roll(100)` 與攻擊者 `+0x08` 比較。
- 傷害流程在 `1CD9B–1CE2D`：
  `Roll(attacker.+0x0a) - defender.+0x0c`，結果至少 1 才扣
  `defender.+0x0e`。玩家的 −10 因原版 `Roll` 先取絕對值，故為 1–10。
- 海盜會走同一砲擊路徑；海怪的 `+0x0a <= 0` 分支會追向玩家並在相鄰時
  進 `sub_1D0B5`，與手冊「海怪逼近後近戰」吻合。

手冊另外明載：遠距較容易落空，落空彈可能命中別艘船。這是玩家可觀察規格；
remake 的射線會在失準時偏到相鄰平行線，仍會檢查誤擊。

## 5. 退出與持久後果

`sub_1C427:1CAEE–1CB43` 是共同收尾：

1. `Roll(50)+150` 寫回 party `+0x9c`，所以海上下一次遭遇比陸戰慢。
2. 玩家單位 `+0x0e` 寫回目前船記錄 `+0x25`。
3. 劇情階段 3 會退回 2。
4. 函式回傳 0。

船體歸零時沿失敗回傳離開海戰。手冊補足玩家層語意：船沉等同全隊死亡。
勝利只發經驗，不發金幣或道具；逃跑是在戰場邊緣的離場點駛出，不能靠岸。

## 6. remake 對應

- 純規則：`internal/game/seabattle.go`
- 規則測試：`internal/game/seabattle_test.go`
- Ebiten 輸入／繪圖／結算：`cmd/demonwinter/seaui.go`
- 正常入口：航海時遭遇倒數歸零
- 確定性驗收入口：`-sea-battle -seed=N`
- 原版海戰 sprite：`SHIP.SHE`（EGA）／`SHIP.SHP`（CGA）

刻意沒有把角色 Strength、Skill、武器或裝備帶入海戰；原版玩家海戰記錄直接
寫固定命中與傷害參數，手冊也明載角色屬性不影響船戰。
