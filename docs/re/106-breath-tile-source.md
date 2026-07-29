# 噴吐動畫圖塊來源：terrain tile 6／7／8

> 日期：2026-07-29  
> 工具：IDA Pro 9.4，資料庫 `workplace/ida/DEMON.INT.asm`  
> 結論：原版沒有獨立的噴吐特效素材。龍種先選出 6／7／8，再沿錐形範圍呼叫
> 共用地形圖塊繪製器。

## 1. 為什麼先前會卡住

`docs/re/23-ai-spell-dispatch.md` 已追到噴吐時序與 `ds:4e2c` 被送入繪圖函式，
但當時沒有往上追完這個 byte 的所有賦值，所以 remake 暫時用單色方塊覆蓋。
從檔名找另一張「火焰 sprite sheet」也找不到，因為原版根本沒有那份檔案。

## 2. IDA 位址證據

IDA 的 `sub_15088` 對應既有 Ghidra 筆記的 `FUN_138d_17b8`。在
`sub_15088:15255` 起，怪物 race 依下表寫入 `ds:4E2C`：

| race | 原版分支 | 寫入值 |
|---:|---|---:|
| 8 | 固定值 | 6 |
| 9 | 固定值 | 7 |
| 10 | 固定值 | 8 |
| 11 | `Roll(3) + 5` | 6–8 |

到 `sub_15088:15329`，程式把三個參數壓棧：

1. `ds:4E2C`（上表選出的圖塊值）；
2. 目標格 `y + 5`；
3. 目標格 `x + 5`；

接著呼叫 `sub_1F50E`。後者是一般圖塊繪製 wrapper：正規化座標後進入 CGA／EGA
共用的 tile draw 路徑。它不是怪物 sprite 或獨立動畫解碼器。

因此 6／7／8 不是「學派色碼」或臨時效果編號，而是直接可索引
`DEMON.SHE`／`WINTER.SHE`（CGA 時為 `.SHP`）的 terrain tile 值。總覽中這幾格
看起來仍像地形並不構成反證；原版就是拿它們逐格覆寫戰場，形成抽象的噴吐動畫。

## 3. Remake 對應

`internal/game.Battle` 在 `Breathe` 時計錄這次由 race 決定的 tile；
`cmd/demonwinter/battleui.go` 沿既有錐形 cells，以目前 theme 的 terrain tileset
繪製該 index。這使同一套機制自然支援：

- 原版 EGA：EGA tile 6／7／8；
- 原版 CGA：對應 CGA tile 6／7／8；
- 未來 Modern EGA：同索引的重畫 tile。

動畫的命中、傷害與時序沒有改；修正只把暫代的橘色矩形換回原版素材來源。

## 4. 驗收

- 單元測試固定 race 8，驗證一次噴吐記錄 tile 6。
- F8 切換後 breath renderer 讀取當前 `a.normal`／`a.winter`，不快取舊 theme image。
- EGA／CGA 實跑截圖應看到相同格位、不同原版素材，不再出現純色方塊。

