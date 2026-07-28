# 《冬之魔》宣傳片

這裡只保存可重跑的擷取與剪輯腳本。實機錄影、原版圖像及 PC speaker WAV
都放在 `workplace/promo/`，由 `.gitignore` 排除，不隨程式散布。

原版沒有背景音樂；成片只使用引擎依反組譯資料還原的 PC speaker 音效，
不加入 AI 音樂或仿作配樂。

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
