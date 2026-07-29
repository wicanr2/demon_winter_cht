package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// origParty 是原版出貨的 PARTY.DAT。
func origParty(t *testing.T) *scenario.SaveGame {
	t.Helper()
	p := filepath.Join("..", "..", "workplace", "orig", "demwin", "DEM_DATA", "PARTY.DAT")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("找不到原版 PARTY.DAT，略過：%v", err)
	}
	s, err := scenario.LoadSaveGame(p)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// **出貨的 PARTY.DAT 不是新遊戲。**
//
// 這條測試釘住的不是「我的常數對」，是「這兩份狀態不一樣」——
// 也就是為什麼 `-newgame` 這個旗標必須存在。
//
// 之前所有試玩都是從出貨存檔開始的，等於從地城深處的中段開始玩，
// 而**沒有任何訊號提示這件事**（畫面上就是一支正常的五人隊伍）。
// 如果哪天有人「簡化」掉 `-newgame`，這條會炸。
func TestShippedSaveIsNotANewGame(t *testing.T) {
	got := origParty(t)

	type field struct {
		name        string
		shipped     int
		wantNewGame int
	}
	fields := []field{
		{"糧食", int(got.Rations), NewGameRations},
		{"日", int(got.Day), NewGameDay},
		{"時", int(got.Hour), NewGameHour},
		{"X", int(got.PositionX), NewGameX},
		{"Y", int(got.PositionY), NewGameY},
		{"地圖", int(got.MapID), NewGameMapID},
		{"朝向", int(got.Facing), NewGameFacing},
		{"金幣", got.Gold, NewGameGold},
	}
	same := 0
	for _, f := range fields {
		if f.shipped == f.wantNewGame {
			same++
			t.Logf("%s 相同（%d）", f.name, f.shipped)
		}
	}
	if same == len(fields) {
		t.Error("出貨存檔與新遊戲起始值完全相同 —— 那 -newgame 就沒有存在的必要，" +
			"要重新確認 docs/re/87 的判讀")
	}
	// 最關鍵的一項單獨釘：地圖。1 是地城、34 是世界地圖。
	if got.MapID == NewGameMapID {
		t.Errorf("出貨存檔的地圖 = %d，與新遊戲相同 —— 判讀有問題", got.MapID)
	}
}

// ApplyNewGame 要把每一個欄位都設到位。
func TestApplyNewGame(t *testing.T) {
	s := origParty(t)
	ApplyNewGame(s, NewGameEncounterMin)

	checks := []struct {
		name      string
		got, want int
	}{
		{"糧食", int(s.Rations), NewGameRations},
		{"日", int(s.Day), NewGameDay},
		{"時", int(s.Hour), NewGameHour},
		{"步數計數", int(s.TimeCounter), NewGameTimeCounter},
		{"X", int(s.PositionX), NewGameX},
		{"Y", int(s.PositionY), NewGameY},
		{"地圖", int(s.MapID), NewGameMapID},
		{"朝向", int(s.Facing), NewGameFacing},
		{"光源", int(s.LightSource), NewGameLight},
		{"金幣", s.Gold, NewGameGold},
		{"商隊基準", int(s.MerchantBase), NewGameMerchantBase},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d，預期 %d", c.name, c.got, c.want)
		}
	}

	// 隊伍人數歸零 —— 原版建角是「每造一個 +1」（`0x15109`）。
	if s.PartySize != 0 {
		t.Errorf("隊伍人數 = %d，新遊戲還沒建角應該是 0", s.PartySize)
	}

	// 陣型格全部是 0xff（空），**不是 0** —— 0 是合法的槽位編號。
	for i, f := range s.Formation {
		if f != newGameFormationEmpty {
			t.Errorf("陣型格 %d = 0x%02x，預期 0x%02x", i, f, newGameFormationEmpty)
		}
	}
}

// 起始只有一艘船，在第 9 格，而且**船體不是滿值**。
//
// 出貨存檔裡有兩艘船。只覆蓋第 9 格的話會把別人玩剩的船帶進新遊戲 ——
// 而玩家不會知道那艘船是哪來的。
func TestApplyNewGameShips(t *testing.T) {
	s := origParty(t)
	before := 0
	for _, sh := range s.Ships {
		if sh.Exists() {
			before++
		}
	}
	if before == 0 {
		t.Log("出貨存檔沒有船 —— 那「清空」這一步就驗不到了")
	}

	ApplyNewGame(s, NewGameEncounterMin)

	n := 0
	for i, sh := range s.Ships {
		if !sh.Exists() {
			continue
		}
		n++
		if i != NewGameShipSlot {
			t.Errorf("第 %d 格有船，新遊戲只該有第 %d 格", i, NewGameShipSlot)
		}
	}
	if n != 1 {
		t.Errorf("新遊戲有 %d 艘船，預期 1", n)
	}

	sh := s.Ships[NewGameShipSlot]
	if sh.X != NewGameShipX || sh.Y != NewGameShipY || sh.MapID != NewGameShipMapID {
		t.Errorf("船在 圖%d (%d,%d)，預期 圖%d (%d,%d)",
			sh.MapID, sh.X, sh.Y, NewGameShipMapID, NewGameShipX, NewGameShipY)
	}
	if sh.Hull != NewGameShipHull {
		t.Errorf("船體 = %d，預期 %d", sh.Hull, NewGameShipHull)
	}
	if sh.Hull == scenario.ShipMaxHull {
		t.Error("船體是滿值 —— 原版起始那艘是 67，不是 75（別順手補滿）")
	}
}

// 遭遇倒數要夾在 15–19；Roll(5) 是 1-based。
func TestApplyNewGameEncounterClamp(t *testing.T) {
	for _, in := range []int{-5, 0, 14, 15, 18, 19, 20, 999} {
		s := origParty(t)
		ApplyNewGame(s, in)
		got := int(s.EncounterCountdown)
		if got < NewGameEncounterMin || got > NewGameEncounterMax {
			t.Errorf("給 %d 得到 %d，超出 %d–%d",
				in, got, NewGameEncounterMin, NewGameEncounterMax)
		}
	}
	low, high := origParty(t), origParty(t)
	ApplyNewGame(low, 0)
	ApplyNewGame(high, 999)
	if low.EncounterCountdown != 15 || high.EncounterCountdown != 19 {
		t.Fatalf("鉗制端點 = %d／%d，預期 15／19",
			low.EncounterCountdown, high.EncounterCountdown)
	}
}
