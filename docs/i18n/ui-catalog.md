# 介面文案 JSON

執行期來源：`assets/lang/<lang>/ui.json`

## 邊界

- Go 引擎只保存穩定 key、格式化參數、熱鍵、action 與可用條件。
- 玩家可見的標題、命令、訊息、說明及格式字串全部在 JSON 的 `text`。
- `en` 只在有原版英文可對照時保存，不參與繁中查表。
- 原版有天然索引的事件、道具、怪物與月份資料仍使用各自純文字 catalog；
  `ui.json` 只處理 remake 自有、沒有 legacy index 的介面文案。

格式：

```json
{
  "locale": "zh-Hant",
  "commandLayouts": {
    "world": {
      "retro": ["world.menu.move", "world.menu.party"],
      "groups": [
        {
          "titleKey": "command.tab.common",
          "column": 0,
          "items": ["world.menu.move", "world.menu.party"]
        }
      ]
    }
  },
  "entries": [
    {
      "key": "plot.uncurse",
      "en": "UNCURSE",
      "text": "解咒"
    }
  ]
}
```

`commandLayouts` 也是資料契約：

- `retro` 決定原版紅色直式選單順序；
- `groups` 決定現代模式的 tab 標題、左右欄與按鈕順序；
- 每個 item 必須存在於 `entries` 與 `retro`，而且恰好分組一次；
- Go 只提供同 key 的 action／enabled 狀態，不決定排版。

## 嚴格載入

`i18n.Translator.UI("plot.uncurse")` 只接受 key。缺 key 不會靜默退回程式碼裡
另一份中文，而會顯示 `⟦plot.uncurse⟧`；發行前 `dwstrings uicheck` 會更早
以非零狀態擋住：

- 程式使用但 JSON 缺少的 key；
- JSON 沒有呼叫端的孤兒 key；
- 重複、空白或含換行的條目；
- 倚天 Big5 字型無法顯示的字；
- `cmd/demonwinter` 內任何玩家可見的硬編中文。

目前基線：766 條 catalog，玩家程式硬編中文 0 條。

## 為什麼 JSON 與原版散文分開

事件散文需要人類逐段校稿，純文字格式比較適合長文 diff；介面文案是短字串與
語意 key，JSON 能提供穩定 schema、重複 key 檢查與日後版面資料化的入口。
兩者分開可維持引擎／資料邊界，又不犧牲反組譯文本的可審閱性。
