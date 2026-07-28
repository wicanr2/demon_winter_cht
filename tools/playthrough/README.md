# A6 全程試玩的腳本

一行一個指令，格式見 `tools/playthrough.sh` 開頭。跑法：

```
tools/playthrough.sh tools/playthrough/a6-leg1.txt workplace/dump/a6 -newgame -autofight -seed=11
```

**這些腳本要進版控**，理由與 `tools/readme_shots.sh` 相同：手打出來的指令
不留下來的話，下一輪沒有人知道當初那一段是怎麼跑的，而畫面類的
regression 只有重跑同一段才比得出來。

| 檔案 | 內容 | 紀錄 |
|---|---|---|
| `a6-leg1.txt` | 新遊戲 → 建五個角色 → 海濱鎮買闊劍、釘頭鎚、3 匕首與 5 布甲 | `docs/playtest/11`、`12` |
| `a6-leg2-share.txt` | 紮營 `T` 交出，把五人的武器與布甲分給 b–e | `docs/playtest/12` §5 |
| `a6-leg3-equip.txt` | 紮營 `E` 換裝，五人的武器與護甲全部裝上 | `docs/playtest/12` §5 |
| `a6-reform-corners.txt` | 把低 HP 施法者移到 A/C/I 三角、近戰守 D/E | A6 正常練功 |
| `a6-upgrade-armor.txt` | 用正常戰利金替 a/b 換皮甲、c 換鎖子甲 | A6 正常練功 |
| `a6-sell-spares.txt` | 賣掉換下護甲與戰利品，補回治療週轉金 | A6 正常練功 |
| `a6-kobold-camp.txt` | 由海濱鎮進圖 2，掃過八座帳篷與 Uffuspgot 首領戰 | A6 第一個主線地城 |
| `a6-kobold-return-heal.txt` | 營地通關後紮營、返海濱鎮並復活／治療全隊 | A6 第一個主線地城 |
| `a6-level-up-erguard.txt` | 由海濱鎮前往厄加德公會，讓四名達門檻角色升到 2 級 | A6 正常升級 |
| `a6-return-heal-elbarat.txt` | 升級後返回艾巴拉特，把提高後的生命上限補滿 | A6 長途主線整備 |
| `a6-arrive-gamur.txt` | 從厄加德穿過骸骨廳連接段，抵達北方加穆爾神殿 | A6 第二段主線 |
| `a6-leg2-observe.txt` | 來回走 190 步吃遭遇，量金幣／傷害的產出 | `docs/playtest/12` §1 |

**每一段結尾存檔、下一段從存檔接續**（`-newgame` 只有第一段要帶）。
跨模態選單的長腳本不要一次寫完 —— Escape 的層數脫拍一次，後面的方向鍵
就會走到世界地圖上，然後撞到隨機遭遇，而截圖看不出是哪一步錯的。

練功腳本在存檔前必須先 `settle`，再用 `expect 已存檔` 驗證。世界移動可能
隨機撞上商隊；若直接送 `s`，按鍵會被商隊畫面吃掉，腳本仍正常結束但
`PARTY.DAT` 完全沒更新。

驗收用 `PARTY.DAT` 直讀（金幣、陣型、背包型別、裝備格），比判讀截圖確定 ——
指令在 `demon_winter-handoff-20260728.md` §3。

移動段落是 `dwroute -world` 產生的，不是手打的：

```
tools/go.sh run ./cmd/dwroute -world -from 34:28,50 -to 44:34,11
```

⚠ **建角段要用 `key` 不能用 `rep`**：`playthrough.sh` 的 `settle()` 對
「建角」畫面會送 Escape，而強制流程裡 Escape 只會提示還剩幾個。

⚠ **數字鍵是 1-based**（`pressedDigit` 把 1–9 對到 0–8，`0` 回傳 9）。
寫 `key 0` 想選第一項的話那一格會靜默不動作，後面的按鍵全部錯位
（`docs/playtest/11` §2）。
