# 28 — 介面文字與命令版面 JSON 資料分離

日期：2026-07-29

## 遷移範圍

- 766 條玩家介面文字由 `assets/lang/zh-Hant/ui.txt` 遷到 schema 化的
  `assets/lang/zh-Hant/ui.json`；
- 734 個 `Translator.UI(key, fallback)` 呼叫改為嚴格的 `UI(key)`；
- Go 套件層的面向、種族、職業、屬性、世界／營地／戰鬥命令表移除中文欄位；
- world／camp／battle／town 的復古順序、現代 tab、左右欄與按鈕順序移入
  `commandLayouts`；
- 原版有天然索引的事件、道具、怪物等散文 catalog 不改格式。

## 防退步閘門

`dwstrings uicheck` 現在驗證：

1. Go 使用的 key 全在 JSON；
2. JSON 沒有孤兒、重複、空白或含換行的條目；
3. 所有繁中字能由 Big5 倚天字型顯示；
4. command layout 的 item 都存在、復古與現代集合相同、每項只分組一次；
5. `cmd/demonwinter` 玩家程式硬編中文為 0。

缺 key 時 runtime 顯示 `⟦key⟧`，不再從引擎靜默取另一份中文。單元測試另釘住
重複 key、空白 text、排序 round-trip 及缺 key 的醒目行為。

## 實機無差異證據

固定：

```text
map=34 x=28 y=50 seed=11 controls=modern
```

以 JSON 文字與 `commandLayouts` 重建畫面後，和遷移前保存的
`control-mode-modern-command-runtime.png` 做 ImageMagick `compare -metric AE`：

```text
JSON 文字＋版面資料化後像素差異：0（compare=0）
```

這證明資料來源改變沒有偷偷改字、欄距、tab、停用狀態或地圖狀態。

## 裁決

介面資料分離完成；引擎仍保有必須由規則決定的熱鍵、action 與 enabled 條件，
但玩家文字、復古順序與現代排版分組不再硬編於 Go。
