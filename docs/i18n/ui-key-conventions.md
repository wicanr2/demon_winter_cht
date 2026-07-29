# 介面 JSON key 規約

`assets/lang/zh-Hant/ui.json` 使用語意 key：

```text
<畫面>.<群組>.<內容>
```

全小寫 ASCII，以 `.` 分層、必要時用 `_` 連接。key 描述用途，不放中文、行號
或容易改動的版面座標。

## 硬規則

1. 同一句話只有在語意與更新生命週期相同時才共用 key。
2. `%s`、`%d`、寬度與順序是資料契約；修改 JSON 時不得任意增減。
3. 開頭／結尾空白可能是表格排版的一部分，不可自動 trim。
4. `text` 必須是繁體中文且能由 Big5 倚天字型顯示。
5. 文案不含換行；換行與分頁屬引擎版面結構。
6. 套件層命令表只保存 action 對應的 key；順序與分組由 JSON
   `commandLayouts` 決定，例如：

```go
var commands = []struct {
    key   ebiten.Key
    uiKey string
    action game.Action
}{
    {ebiten.KeyA, "battle.cmd.attack", game.ActionAttack},
}
```

7. 動態 key 使用可枚舉前綴並以 `// ui:dynamic <prefix>` 標記；JSON 必須列出
   所有可能值。缺 key 在畫面上會顯示 `⟦key⟧`。

## 不屬玩家文案

命令列開發旗標、錯誤回傳、log、trace、`-scene` 描述及測試 fixture 不進
`ui.json`；`dwstrings uicheck` 只排除明確列名的開發檔案／呼叫端。

## 檢查

```text
go run ./cmd/dwstrings uicheck
```

通過條件是 key 完整、無孤兒、Big5 可顯示且玩家程式硬編中文為 0。
