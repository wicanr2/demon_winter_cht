package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/wicanr2/demon_winter_cht/internal/assets/world"
)

// 改寫地圖並存回（`docs/re/95` §3.9）。
//
// 原版有三處會就地改寫地圖緩衝區，而且**改完立刻把整張圖寫回檔案**
// （`122f:28d0(子地圖, 1)` → `0x18906`，寫 `0x1001` bytes 回 `MAP%d.MAP`）：
//
//	0x1a383   密語門答對，`map[0x48b] = 0`（`docs/re/84`），子地圖 5
//	0x19665   `Move:` 推開家具（`+3` 欄）
//	0x1856e   `U` 的 `P` 動作
//
// > 這裡原本三個地方都寫著「原版只改記憶體，離開地城再進來牆會回到原狀，
// > 照抄」。**那句話是錯的** —— 三處後面都跟著存檔呼叫。
// > 症狀是密語門解開之後換張地圖再回來，牆又長回去了，
// > 而畫面上看起來就像謎題沒解過。
//
// **本專案寫存檔目錄，不蓋原版資料目錄**（`workplace/orig` 唯讀，
// 而且玩家的原版檔是他自己的合法副本）。載入端的三段優先序在
// `world.LoadByID`，與 `nSS.DAT`／`ITEMLOCB.DAT` 同一套。

// mapIsStandalone 回報這個編號的地圖有沒有自己的 `MAPn.MAP`。
//
// **只有 1／3／5。** 其餘在 `SUM.MAP` 裡 —— 原版存檔那一支不分編號一律寫
// `MAP%d.MAP`，但載入端只有 1／3／5 讀這個檔名，所以在地圖 2／4 改的東西
// 會留下一個永遠不會被讀的檔案然後消失。那是原版的洞，不複製。
//
// 實務上碰不到：三件推得動的家具在 1／3，兩個 `P` 動作在 1／3，密語門在 5。
func mapIsStandalone(id int) bool { return id == 1 || id == 3 || id == 5 }

// writeTile 改寫目前這張地圖的一格，重畫，並存回存檔目錄。
//
// **三件事綁在一起是刻意的。** 少了重畫，牆開了畫面上還是牆；
// 少了存回，換張地圖再回來就白解了。這兩個症狀都不會有錯誤訊息。
func (a *app) writeTile(x, y int, tile byte) error {
	if err := a.tiles.SetTileAt(x, y, tile); err != nil {
		return err
	}
	a.drawTiles = ditheredTiles(a.tiles, uint16(a.ditherSeed), a.save.TempleRuins)
	return a.persistMap()
}

// persistMap 把目前這張地圖寫進存檔目錄。
func (a *app) persistMap() error {
	if !mapIsStandalone(a.mapID) {
		return nil
	}
	path := filepath.Join(a.saveDir(), world.MapFileName(a.mapID))
	if err := world.SaveMap(path, a.tiles); err != nil {
		return fmt.Errorf("寫回地圖 %d：%w", a.mapID, err)
	}
	if logToStderr {
		log.Printf("地圖 %d 已改寫，寫到 %s", a.mapID, path)
	}
	return nil
}
