# 怪物範圍法術中心不能讀玩家游標

> 日期：2026-07-29
> 性質：收尾稽核找到的呼叫端缺口，不是原版規則未知。

## 症狀

怪物 AI 的前半段已完整：

1. 從玩家側挑一個單位作 5×5 中心。
2. 計算框內敵我數量。
3. `己方命中 × 2 > 敵方命中` 時否決施法。

但 `monsterCast` 最後呼叫共用 `applySpell`；`EffectAOE` 從
`app.aoeX/aoeY` 取中心。這兩格屬於**玩家選點游標的 UI 狀態**，怪物沒有設定，
所以效果可能落在 `(0,0)`，或落在玩家上一次選過的位置。AI 的選點與誤傷判定
雖然正確，真正受傷的人卻是另一群。

## 修正

- 共用效果端改為 `applySpellAt(..., areaX, areaY)`，中心成為明確參數。
- 玩家施法的 `applySpell` wrapper 仍傳入玩家的 `aoeX/aoeY`。
- 怪物施法傳入 `monsterCastTarget` 回傳單位的 `(X,Y)`。
- 單體法術的中心固定為零值且不會被讀取。

這不改原版公式、AI 選點、法力投入或誤傷否決，只修正 remake 的 UI 狀態洩漏。

## 回歸證據

`cmd/demonwinter/battleui_test.go` 釘住：

- AOE 中心等於 AI 選中的目標座標。
- 單體法術不帶範圍中心。

全套 `go test ./...` 已在 Docker 的 Xvfb 下通過。Ebiten 套件不能用無
`DISPLAY` 的普通 shell 測試；第一次直接執行的 GLFW 失敗不是程式回歸，
重新依專案驗收規則在 Xvfb 下執行後全綠。
