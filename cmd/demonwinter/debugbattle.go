package main

import (
	"fmt"

	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// debugBattleExamineFixture 只補「?」面板需要、但第一回合尚未自然產生的狀態：
// 怪物的記憶目標與一隻召喚物。它不替玩家攻擊、不花回合、不寫存檔。
func (a *app) debugBattleExamineFixture() error {
	if a.battle == nil {
		return fmt.Errorf("尚未開始戰鬥")
	}
	var player *game.Unit
	for _, u := range a.battle.Units() {
		if u != nil && u.IsPlayer && u.Slot < game.SummonSlotStart {
			player = u
			break
		}
	}
	if player == nil {
		return fmt.Errorf("戰場上沒有隊員")
	}
	for _, u := range a.battle.Units() {
		if u != nil && !u.IsPlayer {
			u.AITargetSlot = player.Slot
		}
	}

	entry, err := a.tables.Summon(0)
	if err != nil {
		return err
	}
	x, y, ok := a.battle.SummonPosition(player)
	if !ok {
		return fmt.Errorf("沒有召喚物落點")
	}
	slot := a.battle.FreeSummonSlot()
	if slot < 0 {
		return fmt.Errorf("沒有召喚物槽位")
	}
	u := a.battle.PlaceSummon(slot, entry, game.KindSummon, x, y)
	u.Name = "QA summon"
	return nil
}

// debugBattleIllusionFixture 建立一隻最高速幻象，並把下一次 Roll(10) 固定在
// 消失區間。這只用來截取「第一次行動前消散」的可視證據；不寫存檔，也不替
// 正式規則另開捷徑，實際判定仍由 Battle.Current 執行。
func (a *app) debugBattleIllusionFixture() error {
	if a.battle == nil {
		return fmt.Errorf("尚未開始戰鬥")
	}
	var player *game.Unit
	for _, u := range a.battle.Units() {
		if u != nil && u.IsPlayer && u.Slot < game.SummonSlotStart {
			player = u
			break
		}
	}
	if player == nil {
		return fmt.Errorf("戰場上沒有隊員")
	}
	entry, err := a.tables.Summon(0)
	if err != nil {
		return err
	}
	x, y, ok := a.battle.SummonPosition(player)
	if !ok {
		return fmt.Errorf("沒有幻象落點")
	}
	slot := a.battle.FreeSummonSlot()
	if slot < 0 {
		return fmt.Errorf("沒有召喚物槽位")
	}
	u := a.battle.PlaceSummon(slot, entry, game.KindIllusion, x, y)
	u.Name = "測試幻象"
	u.Speed = 0x7fff

	// state=1 的下一步是 125，Roll(10)=1，落在原版的 1／2 消失區間。
	// PlaceSummon 已先消耗面向亂數，所以在它之後才設定。
	a.rng.SetState(1)
	a.battle.BeginRound()
	return nil
}
