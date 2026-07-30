# Windows 完整版 BAT／Wine 抽樣驗證

日期：2026-07-30

## 問題

私人 Windows 完整版 ZIP 內原本只有中文檔名的 `開始完整版.bat`。直接重新
解壓可證明該檔確實存在且為 103 位元組，但 Wine 9.0 的 `cmd` 無法可靠辨識
中文批次檔名。這也代表舊式解壓工具、非中文系統或其他相容層可能讓玩家誤以為
批次檔消失，或無法從命令列啟動它。

## 修正

`tools/package-full-local.sh` 現在另外產生相同內容、純 ASCII 檔名的
`start-full.bat`，並在包內 `README.md` 指定它為建議入口。原本的
`開始完整版.bat` 保留，未移除中文便利入口。

批次檔使用 `%~dp0` 取得自身目錄，再把下列套件內路徑傳給引擎：

- `original\DEM_DATA`
- `fonts\etan_font`

## 從 ZIP 重新解壓後的驗證

測試在一次性 Ubuntu 24.04／Wine 9.0／Xvfb Docker 容器執行，網路關閉；
輸入是實際的
`dist-all/DemonWinter-zh-Hant-0.1.0-windows-x86_64-full-local.zip`，不是工作樹
內的 Windows 執行檔。

驗證項目：

1. `start-full.bat` 與 `開始完整版.bat` 均存在且非空。
2. `demonwinter.exe -list-scenes` 可由 Wine 直接執行。
3. 以 Windows `Z:` 絕對路徑模擬啟動 `start-full.bat -list-scenes`，回傳成功，
   並輸出 `armory`、`bell`、`cage-secret` 等場景。
4. 下列四段 remake 自製配樂均存在且非空：
   - `docs/audio/remake-exploration.wav`
   - `docs/audio/remake-sanctuary.wav`
   - `docs/audio/remake-battle.wav`
   - `docs/audio/remake-finale.wav`

修正後 Windows 私人完整版 SHA-256：

```text
5c4e0b11802de01cf7749d35352aac6d27551a5607b4c6c0b048aad57b74edb1
```

這是 Wine 相容層的啟動抽樣，不冒充 Windows 實體機的顯示、音訊裝置或長時間
遊玩驗收；但已證明實際 ZIP 的 ASCII 批次入口能找到 EXE，並正確串接包內資料
與字型。
