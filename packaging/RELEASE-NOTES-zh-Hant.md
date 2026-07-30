# Demon's Winter 冬之魔繁中 remake 第一版

這是 SSI 1988 年 DOS CRPG《Demon's Winter》的乾淨重寫繁體中文版。

## 下載選擇

- Linux x86_64：AppImage。
- Windows x86_64：可攜式 ZIP。執行檔只依賴 Windows 系統 DLL；包內附有
  實際匯入清單與第三方 DLL 稽核結果。
- macOS Intel：`macOS-amd64.zip`。
- macOS Apple Silicon：`macOS-arm64.zip`。

每個產物旁皆附 `.sha256` 校驗檔。解壓後請先閱讀包內的 `開始遊戲.txt` 與
`README.md`。

## 玩家需要自行準備的檔案

基於著作權與字型授權，本 Release 不包含 DOS 原版遊戲資料或倚天
`STDFONT.15`／`SPCFONT.15`。請以 `-data` 指向自己的合法 DOS 版
`DEM_DATA`，並以 `-eten` 指向自己的倚天 16×15 字型目錄。進度預設寫到
獨立存檔位置，不會覆蓋原版資料。

## remake 功能

- 完整繁體中文劇情、介面與遊戲內手札；`F1` 隨時查詢說明，既有規則、
  數值與查詢訊息全部保留。
- `F6` 切換復古紅色命令介面／現代兩欄操作介面。
- `F7` 獨立開關四組 remake 新編場景配樂，不影響原版短音效。
- `F8` 輪替 EGA、CGA 與 Modern Icon。
- `F10` 安全離開；關閉視窗會先嘗試自動存檔，失敗時留在遊戲。
- 主線、戰鬥、城鎮、地城、海戰、存讀檔與結局已完成；提供具名
  `-scene` 書籤方便除錯，但不會偷偷設定劇情旗標。

Modern Icon 是 remake 自製的可選主題，不是 1988 原版素材。原版沒有背景
配樂；本版配樂亦明確標示為 remake 新編曲。
