# remake 場景配樂與正式封裝證據

日期：2026-07-30
狀態：配樂程式、四段 WAV、Linux AppImage、Windows ZIP、macOS 原生 CI 與
A6 正常玩家第一段完成；實際音訊裝置聽測尚待使用者

## 配樂邊界

原版 DOS 無背景配樂。`internal/audio/music` 新增的是明確標示的 remake 原創
配樂，以程式合成三角波，不使用 SoundFont、第三方取樣或原版素材：

| 場景 | 用途 |
|---|---|
| Exploration | 世界與地城探索 |
| Sanctuary | 標題、城鎮、營地、商隊 |
| Battle | 一般戰鬥與海戰 |
| Finale | 死亡、魔王與結局 |

`F7` 獨立切換配樂；`-music-volume 0–1` 控制音量。原版 PC speaker 音效仍走
既有 Sound 旗標，兩者可同時存在。單元測試確認四組 PCM 非空、16-bit 對齊，
零音量與 Silent 不產生波形；完整 Go 與 `uicheck` 839/839、硬編中文 0 通過。
同一套合成器另輸出四段 44.1 kHz、16-bit、單聲道 WAV 至 `docs/audio/`，供
GitHub 直接試聽；它們不是另一套人工後製音檔。

## Help 與包內文件

`F1` 固定開啟完整繁中手札；第一章新增 remake 操作、F6／F7／F8／F10 與探索
命令速查，後續原有故事、規則、數值及查詢訊息全部保留。AppImage、Windows ZIP
與 macOS `.app` 的 staging 都包含 `README.md`、`開始遊戲.txt` 與同一份手札。

## Linux AppImage

- 以官方 AppDir → appimagetool Type 2 流程建立，不是更名的 tar。
- appimagetool 與 Type 2 runtime 均釘選 SHA-256。
- 私有 `usr/lib` 收錄非 glibc 的直接動態相依，`AppRun` 指向包內翻譯、手札與
  Modern Icon。
- RC1 已成功產生並可 `--appimage-extract`；包內執行檔、README 與手札存在。

## Windows ZIP

- `CGO_ENABLED=0` 的 amd64 PE 經 `objdump -p` 稽核 import。
- 現況只有 Windows 系統 DLL，需隨包附帶的第三方 DLL 為 0；ZIP 內附實際
  import 清單與「第三方 DLL：0」說明，避免含糊宣稱。
- ZIP 包含引擎、翻譯、手札、README、開始說明與 Modern Icon。

## macOS

`.github/workflows/cross-platform-release.yml` 已在原生 Intel／Apple Silicon runner
建立各自 `.app` ZIP，包含 `Info.plist`、Resources、ad-hoc codesign 與
`-list-scenes` smoke。包含四段 WAV 預覽的最終候選版，兩架構與
AppImage／Windows job 全綠：
[run 30522387676](https://github.com/wicanr2/demon_winter_cht/actions/runs/30522387676)。

## Help 與既有訊息保留

F1 開啟的手札只在最前方新增 remake 操作章；原有繁中說明、規則提示、數值
與查詢訊息仍完整接在後方。發行包同時放入 `README.md`、`開始遊戲.txt` 與
`assets/manual/zh-Hant/manual.txt`，因此遊戲內、離線套件與 GitHub 三處都能
查到操作方式，而不是用新版速查取代舊訊息。

## 最後 A6 抽樣

`a6-leg1.txt` 以 `-newgame -autofight -seed 11` 重跑正常玩家第一段；沒有
`-scene` 或劇情旗標。Trace 共 19 個狀態節點：五人建角、首次存檔、抵達海濱
鎮、購買武器／護甲、離城與再次存檔均成功；產出四張實機截圖、`PARTY.DAT`、
五張 `nSS.DAT` 與 `ITEMLOCB.DAT`。新增配樂同步及 Help 沒有改動正常流程。

## 尚未通過的硬閘門

1. 使用者試聽四段 WAV，並於有聲裝置實際確認 F7 與原版音效疊加。
2. 使用者 P4 最終視覺核准。
3. 上述核准後，以正式版本號重建 artifact 並建立 GitHub Release。
