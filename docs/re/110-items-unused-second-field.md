# `ITEMS.DAT` 第二數值欄：DOS 執行檔未消費

> 狀態：**C7 結案**（2026-07-29）

## 1. 問題

`ITEMS.DAT` 每筆是名稱加七個數值。第二數值欄只有 0／1：

- 八種武器，以及 ring／wand／staff／rod／amulet／medallion／figurine／
  talisman／salve 為 1；
- 護甲、vial、gem、torch、lantern 與兩件主線物品為 0。

舊 parser 因這個分組把它命名為 `WeaponSlot`，但 salve 與 vial 的邊界不支持
「是不是手持裝備」的完整規則。worklist C7 因而要求找到 consumer，不能靠名稱
發明玩法。

## 2. IDA 全 xref 稽核

原版載入後的 30×7 word 數值表由 far pointer `ds:5300` 指向。IDA 9.4 listing
中 `les bx, ds:5300h` 全檔只有六處：

| listing 行附近 | 常式／用途 | 讀取欄位 |
|---:|---|---|
| 20032 | `sub_1AA07` 掉寶價格上限 | `record+0` 價格 |
| 20258 | 同常式選 charge kind | `record+4` 第三數值欄 |
| 20303 | 同常式複製四個效果類別 | `record+6..+12` |
| 22187 | `sub_1BAD1` 同類效果類別路徑 | `record+6..+12` |
| 22537 | 同常式讀 charge kind | `record+4` |
| 49056 | `sub_294EB` 估價 | `record+0` 價格 |

另兩個 `5300h` 參照在資源載入端寫入／整理 far pointer，不是資料 consumer。
全檔沒有 `record+2` 的讀取，也沒有把 `ds:5300` 複製成另一個 alias 後間接消費。

因此對這份 DOS `DEMON.INT` 可下的最強結論是：

> 第二數值欄存在於資料檔並被 loader 解析進表，但 runtime 不讀它。

它可能是跨平台共用資料、早期設計殘留或其他版本的旗標；這些都是未知，
不能反推 DOS remake 行為。

## 3. Remake 處理

- `Item.WeaponSlot` 改名 `UnusedFlag`。
- parser 仍保留原始 0／1，真實檔案測試仍釘住武器與護甲的資料錨點。
- 裝備分類繼續使用已確認的 ITEMS 索引 0–7／8–12 與原版裝備 UI 規則，
  不讀這個 unused flag。
- 不把「沒有 consumer」誤寫成「這個 byte 永遠沒有任何歷史用途」；結論範圍
  僅限目前具 hash 的 DOS 執行檔。
