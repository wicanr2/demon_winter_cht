# PARTY trailer `+09h`：DOS 版未使用的保留 byte

> 狀態：**C9 結案**（2026-07-29）  
> 輸入：`DEMON.INT` SHA-256
> `fc1df05513bfa0f1a38f95ce0fbe5e6ec390c8192b2f99b3b6118b3c23868ea5`  
> 工具：IDA Pro 9.4 headless listing

## 1. 欄位邊界

`PARTY.DAT` 的 194-byte trailer 從檔案 `0x514` 開始：

| trailer offset | 長度 | 已確認用途 |
|---:|---:|---|
| `+00h`–`+08h` | 9 | 3×3 陣型格（`docs/re/34`） |
| `+09h` | 1 | 本篇稽核的 byte |
| `+0Ah`–`+0Dh` | 4 | 32-bit 隊伍金幣（`docs/re/19`） |

兩份隨原版保存的 `PARTY.DAT`／`PARTY.BAK` 在 `+09h` 都是 `00`。這只證明
兩個樣本值為零，不能單獨證明欄位無用途。

## 2. IDA 全檔 consumer 稽核

執行期 party trailer 由 far pointer `ds:4C76h` 指向。IDA listing 逐一檢查
這個 pointer 的全部引用，並另掃所有帶 literal displacement `+9` 的
`es:[bx+9]`：

- 全檔只有三條 literal `es:[bx+9]`；分別位於 `sub_1916D` 的角色／道具
  顯示記錄、`sub_1AA07` 的 17-byte 掉寶記錄，以及 `sub_26DDF` 的另一個
  16-byte 暫存記錄。三者的 `ES:BX` 都來自 stack argument/local far pointer，
  不是 `ds:4C76h`。
- party struct 開頭的索引式存取只屬於陣型。Reorder、離隊、入隊與佈陣四條
  路徑的迴圈上界都是 `index < 9`，因此只可能觸及 `+00h`–`+08h`。
- 新遊戲初始化在 `loc_20D84` 同樣以 `cmp ax,9 / jl` 把前九格寫成 `FF`；
  下一個已具名寫入直接是 `+0Ah/+0Ch` 的 32-bit 金幣 75，沒有碰 `+09h`。
- 金幣的所有算術都成對使用 `+0Ah` 與 `+0Ch` 兩個 word，沒有把前一 byte
  當符號、進位或額外高位。
- 載入／儲存會搬運完整 trailer；這是序列化保存，不是 gameplay consumer。

因此對這個具 hash 的 DOS 執行檔，最強而不過度延伸的結論是：

> trailer `+09h` 是未被玩法消費的保留 byte。

「保留」不等於知道它的歷史設計意圖，也不保證 Apple II、C64 或其他版本未曾
使用。不能為 remake 發明一個旗標或數值用途。

## 3. Remake 處理

- `SaveGame.Unknown09` 改名 `Reserved09`，撤回「尚待猜語意」的暗示。
- decoder 仍讀出此 byte，encoder 仍寫回；非零的外來存檔值也不得被清掉。
- 新遊戲沿用零值，但規則層不讀它。
- C9 至此沒有剩餘 gameplay 欄位；若日後取得其他平台 binary，應另立
  證據範圍，不回頭改寫本篇的 DOS 結論。
