package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
	"github.com/wicanr2/demon_winter_cht/internal/ui"
	"github.com/wicanr2/demon_winter_cht/internal/ui/layout"
)

// 離開與存檔。
//
// 照 esc-cancel-f10-quit-autosave 鐵則：
//
//   - **ESC 只取消**，退回上一層，永遠不會結束遊戲。
//     按錯 ESC 就噴掉一小時進度是設計失誤，不是玩家的錯。
//   - **F10 才是離開**，而且先跳 Yes／No，選 Yes 才自動存檔並退出。
//
// 存檔路徑預設**不是**原版資料目錄。玩家的原版 `PARTY.DAT` 是他自己的
// 合法副本，把遊玩進度寫回去等於改壞人家的東西。

// loadSave 讀取存檔：優先讀玩家的進度檔，沒有就拿原版存檔當起始狀態。
//
// 回傳的第二個值是「這是不是全新開始」，用來決定要不要在畫面上提示。
func loadSave(savePath, dataDir string) (*scenario.SaveGame, bool, error) {
	if _, err := os.Stat(savePath); err == nil {
		s, err := scenario.LoadSaveGame(savePath)
		return s, false, err
	}
	s, err := scenario.LoadSaveGame(filepath.Join(dataDir, "PARTY.DAT"))
	return s, true, err
}

// writeSave 把目前的隊伍狀態寫回存檔檔案。
//
// **這裡漏一個欄位，玩家就會丟掉那部分進度而且畫面上看不出來** ——
// 買的東西、睡掉的時間、打來的糧食都在 app 這一層，不主動同步回
// `a.save` 的話，寫出去的還是載入時那份。金幣（`a.gold`／`a.setGold`）
// 直接操作 `a.save`，所以不在這裡列。
func (a *app) writeSave() error {
	for i := range a.members {
		if i >= len(a.save.Characters) {
			break
		}
		a.members[i].ApplyTo(&a.save.Characters[i])
	}
	a.save.PositionX = byte(a.party.X())
	a.save.PositionY = byte(a.party.Y())
	a.save.Facing = byte(a.party.Facing())
	a.save.MapID = byte(a.mapID)
	a.save.LightSource = a.torch

	a.save.Hour = byte(a.clock.Hour())
	a.save.Day = byte(a.clock.Day())
	a.save.Month = byte(a.clock.Month())
	a.save.TimeCounter = byte(a.clock.Steps())

	if err := os.MkdirAll(filepath.Dir(a.savePath), 0o755); err != nil {
		return fmt.Errorf("建立存檔目錄失敗: %w", err)
	}
	if err := a.writeSpecialTiles(); err != nil {
		return err
	}
	return a.save.SaveTo(a.savePath)
}

// writeSpecialTiles 把五張子地圖的特殊格清單寫進存檔目錄。
//
// 這正是上面那段警語講的情況：清單會被事件就地改寫（`docs/re/78` §3），
// 不寫出去的話「這個一次性事件已經觸發過」就在關掉遊戲時消失，
// **而且畫面上完全看不出來** —— 下次進同一個地城，用掉的事件又活了。
//
// 寫到存檔目錄，不是原版資料目錄。`nSS.DAT` 跟 `PARTY.DAT` 同一個等級：
// 玩家的原版檔是他自己的合法副本，遊玩進度不該寫回去。
func (a *app) writeSpecialTiles() error {
	return scenario.WriteSpecialTileSet(filepath.Dir(a.savePath), a.special)
}

// saveNow 存檔並把結果寫進狀態訊息。
func (a *app) saveNow() {
	if err := a.writeSave(); err != nil {
		a.message = fmt.Sprintf("存檔失敗：%v", err)
		return
	}
	// 訊息要塞進狀態欄的 21 格，完整路徑放不下（會被裁一半，看起來像壞掉）。
	a.message = "已存檔"
	if logToStderr {
		log.Printf("存檔寫到 %s", a.savePath)
	}
}

// updateQuitDialog 處理離開確認。回傳 true 代表這一幀已被對話框吃掉。
func (a *app) updateQuitDialog() (bool, error) {
	if !a.quitting {
		// F10 是唯一的離開手勢，刻意與 ESC 在鍵盤上分得很開。
		if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
			a.quitting = true
			return true, nil
		}
		return false, nil
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyY):
		if err := a.writeSave(); err != nil {
			// 存不進去就別走 —— 直接退出等於把進度丟了還不告訴人。
			a.quitting = false
			a.message = fmt.Sprintf("存檔失敗，沒有離開：%v", err)
			return true, nil
		}
		return true, ebiten.Termination
	case inpututil.IsKeyJustPressed(ebiten.KeyN):
		return true, ebiten.Termination
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.quitting = false
	}
	return true, nil
}

// drawQuitDialog 畫離開確認框。
func (a *app) drawQuitDialog(dst *ebiten.Image) {
	const w, h = 400, 6 * ui.LineHeight
	x := (layout.CanvasWidth - w) / 2
	y := (layout.CanvasHeight - h) / 2

	dst.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image).Fill(quitDialogBG)
	ui.StrokeRect(dst, x, y, w, h, markerColor)

	tx, ty := x+ui.LineHeight, y+ui.LineHeight
	a.font.Draw(dst, "要離開遊戲嗎？", tx, ty)
	a.font.Draw(dst, "Y：存檔後離開", tx, ty+ui.LineHeight*3/2)
	a.font.Draw(dst, "N：不存檔直接離開", tx, ty+ui.LineHeight*5/2)
	a.font.Draw(dst, "Esc：繼續玩", tx, ty+ui.LineHeight*7/2)
}

// quitDialogBG 是確認框的底色。刻意不透明 —— 蓋住底下的畫面才看得清楚在問什麼。
var quitDialogBG = color.RGBA{0, 0, 0, 0xff}

// debugGiveSkill 教第一名隊員幾個技能（`-give-skill`）。
//
// 起始隊伍沒有人會觀地或三種學識，那幾個紮營選項在 headless 驗收時
// 一步都走不到。**只動記憶體，不寫存檔**，除非之後真的按了存檔。
func (a *app) debugGiveSkill(spec string) error {
	if len(a.members) == 0 {
		return fmt.Errorf("隊伍是空的")
	}
	for _, f := range strings.Split(spec, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			return fmt.Errorf("技能 id %q 不是數字：%w", f, err)
		}
		if n < 0 || n >= gamedata.NumSkills {
			return fmt.Errorf("技能 id %d 超出 0–%d", n, gamedata.NumSkills-1)
		}
		a.members[0].Skills[n] = true
	}
	return nil
}
