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
