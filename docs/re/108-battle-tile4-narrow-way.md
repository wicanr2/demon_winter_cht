# 108 — 戰場 tile 4：兩張鄰接表產生橫／直窄道

> 狀態：**已解、已實作、已用真實地圖座標截圖驗證**
> 日期：2026-07-29

## 1. 問題

`docs/re/36` 已解出一般戰場是 3×3 個世界 tile，各放大成 5×5，但留下
tile `4` 分支：

- `DS:0A20` 與 `DS:0A32` 是什麼？
- 它如何改寫 5×5 區塊？
- remake 直接把 tile 4 均勻放大是否會改變戰場？

答案是：**會。tile 4 不是均勻 5×5；它依相鄰 tile 決定中央橫向或縱向
窄道，最後只把正中央恢復成 4。**

## 2. 工具與輸入

- IDA 9.4 official：`/home/anr2/ida_94_official/dist`
- IDA listing：`workplace/ida/DEMON.INT.asm`
- 原始輸入：`workplace/ida/DEMON.INT`
- SHA-256：
  `fc1df05513bfa0f1a38f95ce0fbe5e6ec390c8192b2f99b3b6118b3c23868ea5`

資料段是 `31f0`；依 `docs/re/00` 的換算式，
`31f0:0a0e／0a20／0a32` 對應 file offset
`0x2650e／0x26520／0x26532`。

原檔 `0x2650e` 起的 word：

```text
0A0E: 0186 018b 0190 02c6 02cb 02d0 0406 040b 0410
0A20: 0001 0001 ffff 0001 0001 ffff 0001 0001 ffff
0A32: 0003 0003 0003 0003 0003 0003 fffd fffd fffd
```

即：

```text
block start: (6,6) (11,6) (16,6) ... (16,16)
across:      +1 +1 -1 / +1 +1 -1 / +1 +1 -1
perpendicular:
             +3 +3 +3 / +3 +3 +3 / -3 -3 -3
```

兩張表都保證 3×3 索引不會越界。

## 3. 指令證據

函式 `sub_17C5D`，關鍵區間 `17c5:17cb`–`17ed`
（IDA label `loc_17DCB`–`loc_17EDF`）。

### 3.1 選 5×5 底色

```asm
tile       = blocks[i]
source     = i
if tile == 4:
    across = i + word[0A20 + i*2]
    if blocks[across] == 0:
        source = i + word[0A32 + i*2]
fill 5×5 with blocks[source]
```

所以同列相鄰值為 0 時，不以 tile 4 填底，而是取同欄相鄰世界 tile。

### 3.2 寫中央窄道

只有原始 `tile == 4` 才執行：

```asm
if blocks[i + across[i]] == 0:
    p = blockStart + 0x84
    repeat 5: map[p--] = 0
else:
    p = blockStart + 2
    repeat 5: map[p] = 0; p += 0x40

map[blockStart + 0x82] = 4
```

64-byte stride 下：

- `+0x84` 是區塊內 `(4,2)`，倒寫五格得到中央橫列。
- `+2` 加五次 `0x40` 得到中央直欄。
- `+0x82` 是 `(2,2)`，最後一定恢復為 4。

這是逐指令可證明的幾何；「道路／橋樑」的敘事語意仍沒有字串或資料名稱佐證，
因此文件只稱**窄道模板**。

## 4. Remake 對應

`internal/game/battleterrain.go` 現在：

1. 先完整取樣 3×3 `blocks`，避免邊建圖邊看不到鄰格。
2. 用兩張原值表選底色。
3. 寫橫／直五格 0。
4. 恢復中央 tile 4。

兩個單元測試逐格核對完整 5×5：

- 同列相鄰為 0：取同欄相鄰值作底，中央橫列為
  `0,0,4,0,0`。
- 同列相鄰非 0：tile 4 作底，中央直欄除中心外為 0。

## 5. 實機驗證

原版地圖實際含 tile 4：

- `MAP3.MAP`：(17,46)、(17,49)
- `MAP5.MAP`：(40,14)、(44,17)、(37,23)、(39,23)、(39,26)、
  (39,28)、(39,29)

以 `MAP3 (17,46)`、固定 seed 11 強制開戰，Xvfb 截圖
`/tmp/dw-tile4-battle.png`：

- 5×5 tile 4 區塊不再是一整片相同圖塊。
- 中央窄道依鄰格方向成形。
- 隊伍與怪物都能完成部署，戰鬥進入第 1 回合。
- 右欄與中文 UI 無溢出。

截圖是暫存驗收證據，不提交原版衍生素材。

## 6. 剩餘不確定性

- tile 4 在世界觀裡究竟叫道路、橋或其他地形，仍無名稱證據。
- `arg_0 >= 0x80` 的預製戰場是另一條建圖路徑，不屬於本分支。

兩點都不影響已解的幾何與 remake 實作。
