# 58 — 第二支史詩宣傳片製作與驗收

日期：2026-07-30

## 成品

- 影片：[`demon-winter-epic-trailer.mp4`](../promo/demon-winter-epic-trailer.mp4)
- 九宮格：[`demon-winter-epic-review.png`](../promo/demon-winter-epic-review.png)
- 可重播工具：`tools/promo/capture-epic.sh`、`tools/promo/make-epic.sh`
- 本機未剪輯母帶：`workplace/promo/epic/capture/`（Git 忽略）

## 四項硬閘門

| 閘門 | 結果 | 證據 |
|---|---|---|
| Modern 新片頭 | 通過 | 0–3 秒局部揭幕；66–72 秒完整呈現 `demon-winter-modern-title-v1.png` |
| 實際遊玩 | 通過 | 世界、地城、城鎮市集、Xeres 代表戰鬥及海戰均由目前 HEAD 的 X11 視窗錄製 |
| `F8` 三主題 | 通過 | `f8.mp4` 為 17.44 秒不中斷母帶；同一固定種子／座標依 EGA → CGA → Modern Icon → EGA 切換 |
| 配樂與音效 | 通過 | 核准的 72 秒宣傳片新編配樂，加上原版 PC speaker 點音；片尾明示原版沒有 BGM |

`F8` 母帶的四張關鍵圖及 `trace.txt` 位於
`workplace/promo/epic/capture/f8/`。固定條件為 `seed=11`、
`map=34`、`x=28`、`y=50`；切換期間沒有載入存檔、改金幣、糧食、時間、
隊伍或座標。影片不是用四張靜圖模擬切換。

## 實機母帶

| 片段 | 時長 | 內容 |
|---|---:|---|
| `f8/f8.mp4` | 17.44 秒 | 世界行走及連續 F8 |
| `dungeon/dungeon.mp4` | 7.88 秒 | `armory` 書籤的 Modern Icon 地城 |
| `town/town.mp4` | 8.16 秒 | 現代兩欄城鎮選單及市集 |
| `boss/boss.mp4` | 15.08 秒 | 怪物 41、74、91、10；其中 91 是 Xeres |
| `sea/sea.mp4` | 14.04 秒 | Modern Icon 海戰 |

每段都是 1600×900、25 fps 的 H.264 X11 擷取，並保留 trace 及起迄 PNG。
具名場景只負責定位，不自動解謎或修改劇情旗標。

## 成片技術驗收

```text
video: H.264, 1280×720, 25 fps
audio: AAC stereo, 48 kHz
duration: video 72.000000 s / audio 72.000000 s
size: 8,670,960 bytes
integrated loudness: -15.2 LUFS
loudness range: 4.2 LU
true peak: -1.7 dBFS
```

第一輪混音雖然也是 72 秒，但 AAC 後真峰值為 `+0.3 dBFS`，未通過；最終版把
總混音降低約 2 dB 後才達 `-1.7 dBFS`。這項失敗紀錄保留，避免日後重剪時
把削波版本誤當成品。

## 內容裁決

- 1988 段落使用本專案可重現的原版標題實機畫面；不是第三方網站下載影片。
- Modern 新片頭是 remake 新設計；影片模型沒有重新生成或拼寫標題。
- 宣傳片配樂是 remake 新編曲，不冒充原版 BGM。
- Xeres 使用 `MONSTER.DAT` 索引 91；同場的 41、74、10 是既有高風險視覺
  抽樣，不捏造新的魔王或戰鬥規則。
- 公開成片包含遊戲實機影像，但不夾帶可抽取的原版資料檔或倚天字型檔。
