# 《冬之魔》宣傳片

這裡只保存可重跑的擷取與剪輯腳本。實機錄影、原版圖像及 PC speaker WAV
都放在 `workplace/promo/`，由 `.gitignore` 排除，不隨程式散布。

原版沒有背景音樂；第一支成片只使用引擎依反組譯資料還原的 PC speaker
音效。第二支史詩宣傳片另有明確標示的 remake 新編曲，不冒充原版 BGM；
作曲方向與逐鏡規格見
[`docs/promo/demon-winter-epic-trailer-prompt.md`](../../docs/promo/demon-winter-epic-trailer-prompt.md)。

## 1. 擷取實機片段

先重建含 ffmpeg 的測試映像：

```sh
docker build -t demonwinter-go docker/go
```

再執行：

```sh
tools/promo/capture.sh
```

`capture.sh` 透過 `tools/playthrough.sh` 操作真正的遊戲視窗，並以 X11
逐幀錄製；同時保留 trace 與關鍵 PNG，方便證明片段不是 mockup。

## 2. 合成

```sh
tools/promo/make.sh
```

成品為 `workplace/promo/demon-winter-cht-promo.mp4`。合成固定限制兩顆 CPU、
H.264 veryfast、兩條執行緒；影片長度、影音串流與非靜音訊號會在結尾驗證。

目前產物只供本機預覽。若要公開上傳，原版圖像及音效仍有著作權風險，
需先由專案持有人決定發布方式。

## 3. 第二支宣傳片配樂原型

```sh
tools/promo/score.sh
```

這會由 `cmd/dwscore` 產生 72 秒 MIDI，再以 Debian 的 FluidR3 GM SoundFont
渲染 24-bit／48 kHz WAV，檢查時長、響度、真峰值並輸出波形圖。配樂以原作
11 音陣亡旋律的 `B–A–B–C–G–C` 輪廓作辨識動機，其餘和聲、節奏與配器都是
宣傳片新作。MIDI、WAV 與分析輸出均在 gitignored 的
`workplace/promo/score/`。

## 4. 第二支史詩宣傳片

先錄製五段目前 HEAD 的實機母帶：

```sh
bash tools/promo/capture-epic.sh
```

這會保留同一局連續 `F8`、Modern Icon 地城／城鎮、Xeres 代表戰鬥與海戰的
MP4、trace 及關鍵 PNG。接著以核准的新片頭與 72 秒配樂合成：

```sh
bash tools/promo/make-epic.sh
```

本機成品為
`workplace/promo/epic/out/demon-winter-epic-trailer.mp4`。剪輯腳本會強制檢查
72 秒影音長度，並輸出 EBU R128 響度與真峰值摘要。
