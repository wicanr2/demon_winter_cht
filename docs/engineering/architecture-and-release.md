# 架構與發行工程

日期：2026-07-30

## 引擎與資料

- `internal/game` 保存規則與狀態，不依賴 Ebiten 畫面。
- `cmd/demonwinter` 負責輸入、畫面流程與規則結果呈現。
- 玩家文字、命令順序與分組在 `assets/lang/zh-Hant/ui.json`；Go 只保留
  穩定 key、格式參數、熱鍵與 action。`dwstrings uicheck` 是發行閘門。
- 原版地圖、事件、圖形、字型與存檔由玩家合法副本載入；未知 byte 原值
  round-trip，不因 remake 需要而杜撰規則。

## 畫面與操作

EGA、CGA 是原版主題；Modern Icon 是可選的高解析 remake 呈現層。三者共用
邏輯圖塊索引、碰撞、座標與 hitbox，`F8` 只換畫面。`F6` 在復古命令模式與
現代兩欄命令模式間切換；`F1` 固定為說明。關閉視窗與 `F10` 走相同的完整
自動存檔路徑，任一步失敗就留在遊戲。

## 音訊

原版範圍是八個 PC speaker 單音與死亡旋律，位址證據見
[`docs/re/117`](../re/117-audio-xrefs-and-breath-correction.md)。remake 新增的
背景配樂必須明確標成新編曲、使用獨立音量／開關，且不得改動原版音效 selector
或規則時序。

## 驗證

最低發行閘門：

1. Docker 內 `go test ./...`，Ebiten 套件在 Xvfb 下執行。
2. `dwstrings check` 為 500/500，`dwstrings uicheck` 無缺 key、無玩家中文硬編。
3. 固定 seed 的 A6 前期垂直切片與後期高風險抽樣。
4. EGA／CGA／Modern Icon 同狀態畫面檢查。
5. `git diff --check`。
6. 發行包禁入掃描、解壓後執行檔 smoke 與平台格式／相依庫稽核。

## 三平台發行

- Linux：AppImage 必須包含應用程式需要的非系統動態庫，並在乾淨容器執行
  `--help`／場景清單 smoke。
- Windows：ZIP 需包含所有非 Windows 系統 DLL；PE import 由 allowlist
  檢查。若沒有第三方 DLL，文件要明列「零個需附第三方 DLL」，不能只說已含。
- macOS：只能由原生 macOS runner 建置與測試 `.app`；每個架構需有
  `Info.plist`、Resources、簽署檢查及可啟動 smoke。`lipo -archs` 必須與
  目標架構相同；`otool -L` 只允許 `/System/Library` 與 `/usr/lib`，拒絕
  Homebrew、`@rpath` 或建置目錄相依，清單須隨包附上。Linux 交叉編出檔案
  不算證據。

所有包都必須排除原版 `.DAT/.DTT/.SHE/.SHP/.PIC/.PIE` 與倚天
`STDFONT.15`、`SPCFONT.15`。完成矩陣見
[`docs/playtest/50`](../playtest/50-completion-requirement-matrix.md)。
