# Modern EGA B — M1 bounded runtime study

> **試片，未核准；不得量產成完整 atlas。**
>
> 這個目錄不含原版 bitmap。`generate.go` 只用手工像素 primitive 產生 32×28 PNG，
> 原版 atlas 僅用來核對 index、語意、frame 大形與 anchor。

## 範圍

本輪刻意只做能回答 runtime 問題的九個 index（DEMON／WINTER 各一份）：

| 輸出 | index | 證據／限制 |
|---|---:|---|
| plain | `0x23` | `FILES.DAT[0x23] = 1`，屬 plains；本輪只批准這一格的畫法 |
| deep water A | `0x14` | `world.OceanTile`，不可步行 |
| deep water B | `0x62` | `world.OceanTileAlt`，與 `0x14` 同語意的外觀替換 |
| NW coast corner | `0x17` | EGA atlas 的岸／水大形候選；**只適用此方向試片**，須再以實機鄰接圖批准 |
| forest | `0x01` | `FILES.DAT[0x01] = 0`，forest 組的第一格 |
| mountain | `0x63` | EGA atlas 的尖峰大形候選；語意是 visual candidate，不冒充已證實的 hills `0x0e/0x2b` |
| town | `0x2e` | `TownTileValues` 第一格；不得套到 Asaht 的 `0x64` |
| party north A/B | `0x1e/0x1f` | `partyGlyphBase[North]` 的兩步動畫，anchor 相同 |

「同一地形類別」不代表圖能互換。特別是 coast、town、mountain 與 party，這九張
以外的 index 一律不在本輪可套範圍。

## 像素大形

- plain：全格單一大底色，只放五個低對比 3×1 土痕與三個 2×1草／雪痕；
  四邊 2 px 保持乾淨，避免 B 直接縮圖的高頻草點。
- deep water：整格深水，四邊 2 px 都是相同深水 port；水平浪紋只在內部，
  A/B 可換相位而不形成硬色縫。
- coast NW：`x+y<20` 為深水，`20..22` 為淺水，其他為陸；上、左是 water port，
  右邊於 y=10/11、下邊於 x=10/11 轉成 land port。
- forest：2 px 陸地邊界，中央一個 22×16 三階樹冠；不把每片葉子畫成亮點。
- mountain：4 px 陸地邊界，中央單一不對稱三階岩面；不縮小山景插畫。
- town：24×21 建築、寬屋頂、7×8 深門洞；門是第一讀點。
- party：18×24、腳底 y=25、中心 x=16；兩 frame 只換腿，不移 anchor。

WINTER 不是在 DEMON 上蓋白色：同一大形改以藍灰陰影、暖白積雪與較冷地表表達；
水和角色功能色維持，確保 F8 不改變語意。

## Palette

所有 PNG 使用這組不透明 sRGB 色：

```text
void     #12151D
water    #133559 #1F587B #66C9D4
earth    #6C4C2D #AA783F #D8B45A
grass    #256849 #4FA862 #83C772
trunk    #3B2A28 #824D35
stone    #384756 #6F7F8B #B7B4AA
snow     #6F8BA3 #B5C9D5 #E8DFC8
roof     #622431 #B63942
gold     #EDCE68
cloth    #4F78B8
skin     #E0A687
steel    #D8DDDD
```

## Continuity contract

用「邊界 port」驗收，不憑 contact sheet 好不好看：

- `plain`、`forest`、`mountain`、`town`：四邊都是各季節 land port；可接 plain。
- `deep-water-a/b`：四邊都是 water port；A/B 任意相鄰不得有底色裂縫。
- `coast-nw-corner`：top/left 是 water；right/bottom 是 water→land transition。
  它只能接具有相反 port 的 coast tile。因本輪沒有畫完整方向族，**不能放進正式地圖**。
- party 是整格覆寫，四邊不承諾與地形 continuity；這符合原版 glyph 的黑底語意。

## 為何不能直接縮 B

`/tmp/modern-b-runtime-world.png` 的三個失敗在這裡逐一修正：

1. 海岸不再由一張大圖套多個 index；角落以明確 edge port 定義，其他方向留白。
2. 平原刪掉每格重複的高亮草點，改成低對比、低數量的大形。
3. party 不把直式概念角色縮入格內；直接在 32×28 畫 18×24 silhouette，
   且保留兩步動畫與共同 anchor。

## 產生與驗收

```bash
go run artwork/modern-ega/m1/generate.go
```

驗收順序：

1. 檢查 `tiles/*.png` 全部是 32×28 RGBA、不透明。
2. 以 1× 看單格、最近鄰 4× 看 `m1-b-contact-sheet.png`；不能只看放大圖。
3. 做 3×3 同格平鋪：plain 不得出現規則亮點；deep-water A/B 棋盤排列不得有裂縫。
4. 用 port 測試 coast：本輪只驗 top/left 水與 right/bottom transition，未有互補格前
   不進 runtime atlas。
5. 將 `0x23`、`0x14/0x62`、`0x01`、`0x2e`、`0x1e/0x1f` 各自只覆蓋同 index，
   以同 save／座標截 EGA、Modern B；F8 前後地圖、狀態、選取與碰撞必須不變。
6. party 以北向連走兩步驗兩 frame：人物不得滑動，視覺份量須大於失敗試片。

只有逐項截圖批准後，才能把相同語彙擴展到其他 index；本試片本身不是成品批准。
