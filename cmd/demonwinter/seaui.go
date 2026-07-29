package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

var seaColor = color.RGBA{0x00, 0x00, 0xaa, 0xff}

func (a *app) startSeaBattle() {
	a.seaShipSlot = game.BoatIndex(a.save.Boat)
	hull := scenario.ShipMaxHull
	if a.seaShipSlot >= 0 && a.seaShipSlot < len(a.save.Ships) {
		hull = int(a.save.Ships[a.seaShipSlot].Hull)
	}
	count := a.rng.Roll(3)
	enemies := make([]*game.SeaUnit, 0, count)
	taken := map[[2]int]bool{{game.SeaCentre, game.SeaCentre}: true}
	for i := 0; i < count; i++ {
		kind, name, hp, exp := game.SeaPirate, a.tr.UI("sea.enemy.pirate"), 30, 60
		if a.rng.Roll(3) == 1 {
			kind, name, hp, exp = game.SeaMonster, a.tr.UI("sea.enemy.monster"), 20, 45
		}
		x, y, found := 0, 0, false
		for tries := 0; tries < 32; tries++ {
			x = game.SeaCentre - 4 + a.rng.Roll(9) - 1
			y = game.SeaCentre - 4 + a.rng.Roll(9) - 1
			if !taken[[2]int{x, y}] {
				found = true
				break
			}
		}
		if !found {
			for y = game.SeaCentre - 4; y <= game.SeaCentre+4 && !found; y++ {
				for x = game.SeaCentre - 4; x <= game.SeaCentre+4; x++ {
					if !taken[[2]int{x, y}] {
						found = true
						break
					}
				}
			}
			y-- // 外層 for 在找到空格後仍執行了一次 post。
		}
		taken[[2]int{x, y}] = true
		enemies = append(enemies, &game.SeaUnit{
			Name: fmt.Sprintf("%s %d", name, i+1), Kind: kind,
			X: x, Y: y, Facing: game.South, Hull: hp, MaxHull: hp, Experience: exp,
		})
	}
	a.sea = game.NewSeaBattle(a.rng, hull, a.tr.UI("sea.player"), enemies)
	a.log = []string{fmt.Sprintf(a.tr.UI("sea.encounter"), len(enemies))}
	a.trace.note("海戰：%d 名敵人，船體 %d", len(enemies), hull)
}

func (a *app) updateSeaBattle() error {
	b := a.sea
	switch b.Outcome {
	case game.SeaVictory:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
			inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			a.finishSeaBattle(false)
		}
		return nil
	case game.SeaEscaped:
		a.finishSeaBattle(false)
		return nil
	case game.SeaSunk:
		a.finishSeaBattle(true)
		return nil
	}

	acted := false
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		acted = b.Move(false)
		if acted {
			a.logLine(a.tr.UI("sea.move.forward"))
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyLeft):
		acted = b.Turn(-1)
		if acted {
			a.logLine(a.tr.UI("sea.move.left"))
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyRight):
		acted = b.Turn(1)
		if acted {
			a.logLine(a.tr.UI("sea.move.right"))
		}
	case inpututil.IsKeyJustPressed(ebiten.KeySlash):
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
			p := b.PlayerShip()
			a.logf(a.tr.UI("sea.hull.inspect"), p.Hull, p.MaxHull)
		} else {
			acted = b.Move(true)
			if acted {
				a.logLine(a.tr.UI("sea.move.reverse"))
			}
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyI):
		acted = a.fireCannon(game.North)
	case inpututil.IsKeyJustPressed(ebiten.KeyJ):
		acted = a.fireCannon(game.West)
	case inpututil.IsKeyJustPressed(ebiten.KeyK):
		acted = a.fireCannon(game.East)
	case inpututil.IsKeyJustPressed(ebiten.KeyM):
		acted = a.fireCannon(game.South)
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		b.Points = 0
		acted = true
	}
	if acted && b.Outcome == game.SeaOngoing && b.Points == 0 {
		for _, res := range b.EnemyTurn() {
			if res.Hit {
				a.logf(a.tr.UI("sea.enemy.hit"), res.Damage)
			} else if res.Fired {
				a.logLine(a.tr.UI("sea.enemy.miss"))
			}
		}
	}
	return nil
}

func (a *app) fireCannon(dir game.Facing) bool {
	res := a.sea.Fire(dir)
	if !res.Fired {
		return false
	}
	if !res.Hit {
		a.logLine(a.tr.UI("sea.cannon.miss"))
	} else if res.Sunk {
		a.logf(a.tr.UI("sea.cannon.sunk"),
			res.Target.Name, res.Damage)
	} else {
		a.logf(a.tr.UI("sea.cannon.hit"),
			res.Target.Name, res.Damage)
	}
	return true
}

func (a *app) finishSeaBattle(sunk bool) {
	b := a.sea
	if a.seaShipSlot >= 0 && a.seaShipSlot < len(a.save.Ships) {
		a.save.Ships[a.seaShipSlot].Hull = byte(b.PlayerShip().Hull)
	}
	if sunk {
		for i := range a.members {
			a.members[i].CurrentHP = 0
			a.members[i].Status = scenario.StatusDead
		}
		a.sea = nil
		a.save.Boat = 0
		a.party.SetSailing(false)
		a.checkPartyDeath()
		return
	}
	if b.Outcome == game.SeaVictory {
		statuses := make([]game.UnitStatus, len(a.members))
		for i := range a.members {
			statuses[i] = game.UnitStatus(a.members[i].Status)
		}
		if per := game.AwardBattleExp(a.members, statuses, b.Experience()); per > 0 {
			a.message = fmt.Sprintf(a.tr.UI("sea.victory.exp"), per)
		}
	} else {
		a.message = a.tr.UI("sea.escaped")
	}
	a.sea = nil
}

func (a *app) drawSeaBattle(dst *ebiten.Image) {
	b := a.sea
	cellW, cellH, scale := a.tileMetrics()
	camX := b.PlayerShip().X - layout.ViewTilesX/2
	camY := b.PlayerShip().Y - layout.ViewTilesY/2
	for y := 0; y < layout.ViewTilesY; y++ {
		for x := 0; x < layout.ViewTilesX; x++ {
			sx, sy := camX+x, camY+y
			px, py := layout.MapOriginX+x*cellW, layout.MapOriginY+y*cellH
			ui.FillRect(dst, px, py, cellW, cellH, seaColor)
			if sx == 0 || sy == 0 || sx == game.SeaSize-1 || sy == game.SeaSize-1 {
				ui.FillRect(dst, px+cellW/2-1, py+cellH/2-1, 3, 3, frameColor)
			}
		}
	}
	pairs := [...]int{6, 4, 0, 2}
	for _, u := range b.Units {
		if !u.Alive() {
			continue
		}
		vx, vy := u.X-camX, u.Y-camY
		if vx < 0 || vy < 0 || vx >= layout.ViewTilesX || vy >= layout.ViewTilesY {
			continue
		}
		group := 0
		if u.Kind == game.SeaPirate {
			group = 1
		} else if u.Kind == game.SeaMonster {
			group = 2
		}
		frame := group*8 + pairs[u.Facing&3] + b.Round&1
		x, y := layout.MapOriginX+vx*cellW, layout.MapOriginY+vy*cellH
		if img := a.shipSprites.Frame(frame); img != nil {
			ui.DrawImageScaled(dst, img, x, y, scale)
		}
		border := enemyColor
		if u.Kind == game.SeaPlayer {
			border = partyColor
		}
		ui.StrokeRect(dst, x, y, cellW, cellH, border)
	}
	drawMapFrame(dst, cellW, cellH)
	a.drawLogPanel(dst, a.log)

	y := layout.StatusY
	line := func(s string) { a.font.Draw(dst, s, layout.StatusX, y); y += ui.LineHeight }
	p := b.PlayerShip()
	line(fmt.Sprintf(a.tr.UI("sea.header.round"), b.Round))
	line(fmt.Sprintf(a.tr.UI("sea.header.hull"), p.Hull, p.MaxHull))
	line(fmt.Sprintf(a.tr.UI("sea.header.points"), b.Points))
	for _, u := range b.Units[1:] {
		if u.Alive() {
			line(fmt.Sprintf("%s　%d/%d", u.Name, u.Hull, u.MaxHull))
		}
	}
	y = layout.MenuY
	for _, s := range []string{
		a.tr.UI("sea.keys.move"),
		a.tr.UI("sea.keys.reverse"),
		a.tr.UI("sea.keys.fire"),
		a.tr.UI("sea.keys.other"),
	} {
		a.font.Draw(dst, s, layout.StatusX, y)
		y += ui.LineHeight
	}
	if b.Outcome == game.SeaVictory {
		a.font.Draw(dst, a.tr.UI("sea.victory.dismiss"),
			layout.StatusX, y)
	}
}
