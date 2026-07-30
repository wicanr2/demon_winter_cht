# remake 場景配樂與正式封裝證據

日期：2026-07-30
狀態：配樂程式、Linux AppImage、Windows ZIP 技術驗證完成；macOS 原生 CI 與
實際音訊裝置聽測尚待執行

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

`.github/workflows/cross-platform-release.yml` 會在原生 Intel／Apple Silicon runner
建立各自 `.app` ZIP，包含 `Info.plist`、Resources、ad-hoc codesign 與
`-list-scenes` smoke。Linux 端不冒充 macOS 動態證據；CI 綠燈後才更新本節。

## 尚未通過的硬閘門

1. 有聲裝置實際聽測 F7、四場景切換及原版音效疊加。
2. macOS amd64／arm64 原生 workflow。
3. 使用者 P4 最終視覺核准。
4. 上述完成後重跑 A6 抽樣、禁入、解壓 smoke，才可改稱第一版正式 release。
