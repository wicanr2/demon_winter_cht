package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/game"
	"github.com/wicanr2/demon_winter_cht/internal/ui/textlayout"
)

// 紮營選單的 Reorder（規則在 internal/game/formation.go，出處 docs/re/34）。
//
// **原版不是「交換兩個人」，是整張表重填**：一開始把九格全部清成 0xFF，
// 然後照隊伍順序一個一個問「你站哪一格」，直到每個人都有位置。
// 中途沒有確認步驟，問完就結束。這裡照做，只多一個 Esc 取消還原。

// reorderScreen 是 Reorder 的進行狀態。
type reorderScreen struct {
	// grid 是編到一半的陣型；確定之後才寫回存檔。
	grid game.Formation
	// before 是進來時的陣型，Esc 取消時還原用。
	before game.Formation
	// next 是下一個要安排的成員編號。等於隊伍人數時代表編完了。
	next int
	// message 是上一次按鍵的結果（例如「那一格有人了」）。
	message string
}

func (a *app) openReorder() {
	if len(a.members) == 0 {
		a.camp.message = a.tr.UI("camp.empty")
		return
	}
	r := &reorderScreen{before: a.save.Formation, grid: a.save.Formation}
	r.grid.Clear()
	a.camp.message = ""
	a.camp.reorder = r
}

func (a *app) updateReorder() error {
	r := a.camp.reorder

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.save.Formation = r.before
		a.camp.reorder = nil
		a.camp.message = a.tr.UI("reorder.kept")
		return nil
	}
	if r.next >= len(a.members) {
		// 編完了，按任意確認鍵收工。
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
			inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			a.save.Formation = r.grid
			a.camp.reorder = nil
			a.camp.message = a.tr.UI("reorder.updated")
		}
		return nil
	}

	for cell := 0; cell < game.FormationCells; cell++ {
		key := ebiten.KeyA + ebiten.Key(cell)
		if !inpututil.IsKeyJustPressed(key) {
			continue
		}
		if !r.grid.Place(cell, r.next) {
			r.message = fmt.Sprintf(a.tr.UI("reorder.occupied"), game.CellLabel(cell))
			return nil
		}
		r.message = ""
		r.next++
		if r.next >= len(a.members) {
			// 全員就位就直接寫回 —— 原版問完最後一個人就結束，
			// 這裡多留一個畫面讓人看清楚結果，但資料先落地。
			a.save.Formation = r.grid
		}
		return nil
	}
	return nil
}

func (a *app) drawReorder(line func(string)) {
	r := a.camp.reorder

	line(a.tr.UI("reorder.header"))
	line("")
	a.drawFormationGrid(line, r.grid)
	line("")

	if r.next < len(a.members) {
		line(fmt.Sprintf(a.tr.UI("reorder.prompt"), a.members[r.next].Name))
	} else {
		line(a.tr.UI("reorder.done"))
	}
	if r.message != "" {
		line(r.message)
	}
	line("")
	line(a.tr.UI("reorder.keys"))
}

// drawFormationGrid 把九格畫成 3×3，每一格顯示站在那裡的隊員名字。
//
// 原版只印成員編號（`grid[i] + '1'`），這裡改印名字 —— 編號要對照
// 隊伍名冊才知道是誰，在中文介面上沒有理由保留那個抽象。
func (a *app) drawFormationGrid(line func(string), f game.Formation) {
	for row := 0; row < game.FormationRows; row++ {
		s := "  "
		for col := 0; col < game.FormationCols; col++ {
			cell := row*game.FormationCols + col
			name := "－"
			if f.Occupied(cell) {
				if m := int(f[cell]); m >= 0 && m < len(a.members) {
					name = a.members[m].Name
				} else {
					name = fmt.Sprintf("?%d", f[cell])
				}
			}
			s += textlayout.PadCells(
				fmt.Sprintf("%s %s", game.CellLabel(cell), name), 12)
		}
		line(s)
	}
}
