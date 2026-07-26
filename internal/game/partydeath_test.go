package game

import (
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

func TestPartyWiped(t *testing.T) {
	alive := Character{CurrentHP: 1}
	deadHP := Character{CurrentHP: 0}
	deadStatus := Character{CurrentHP: 5, Status: scenario.StatusDead}

	cases := []struct {
		name  string
		party []Character
		want  bool
	}{
		{"空隊伍（還沒建角）", nil, false},
		{"全員存活", []Character{alive, alive}, false},
		{"一個還活著", []Character{deadHP, alive, deadHP}, false},
		{"血歸零", []Character{deadHP, deadHP}, true},
		{"血還有但判死", []Character{deadStatus, deadStatus}, true},
		{"兩種死法混合", []Character{deadHP, deadStatus}, true},
	}
	for _, c := range cases {
		if got := PartyWiped(c.party); got != c.want {
			t.Errorf("%s：得到 %v，預期 %v", c.name, got, c.want)
		}
	}
}

// 空隊伍不算全滅 —— 那是「還沒建角」。
//
// 回 true 會讓新遊戲一開場就宣告全隊死亡（新遊戲的人數從 0 起算，
// `docs/re/88`）。單獨釘一條，因為這是最容易在「簡化」時被拿掉的判斷。
func TestPartyWipedEmptyIsNotDeath(t *testing.T) {
	if PartyWiped(nil) {
		t.Error("nil 隊伍不該算全滅")
	}
	if PartyWiped([]Character{}) {
		t.Error("空隊伍不該算全滅")
	}
}

// 戰鬥的傷害要寫回隊伍，而且要照槽位配對。
//
// **沒有這個寫回，戰鬥完全沒有後果**：打完一場慘勝回到地圖全隊滿血，
// 打輸了每個人也還是滿血、`PartyWiped` 永遠 false、死亡畫面永遠不出現。
func TestWriteBackParty(t *testing.T) {
	members := []Character{
		{Name: "甲", CurrentHP: 20, MaxHP: 20, CurrentSP: 5},
		{Name: "乙", CurrentHP: 20, MaxHP: 20, CurrentSP: 5},
	}
	units := []*Unit{
		// 怪物槽 —— 不該影響隊伍。
		{Slot: 0, HP: 1},
		{Slot: PlayerSlotStart, HP: 3, CurrentSP: 1, Side: SidePlayer},
		// **被魅惑的隊員**：`Side` 變了但槽位沒變，傷害照樣要寫回。
		{Slot: PlayerSlotStart + 1, HP: 0, CurrentSP: 0,
			Side: SideCharmedPlayer, Status: UnitStatus(scenario.StatusDead)},
		// 召喚物槽 —— 不是隊伍成員，不寫回。
		{Slot: SummonSlotStart, HP: 7, Side: SideSummon},
	}

	WriteBackParty(members, units)

	if members[0].CurrentHP != 3 || members[0].CurrentSP != 1 {
		t.Errorf("甲 = %d HP／%d SP，預期 3／1", members[0].CurrentHP, members[0].CurrentSP)
	}
	if members[1].CurrentHP != 0 {
		t.Errorf("乙 = %d HP，預期 0（被魅惑也要寫回）", members[1].CurrentHP)
	}
	if members[1].Status != scenario.StatusDead {
		t.Errorf("乙的狀態 = %d，預期死亡", members[1].Status)
	}
	if !PartyWiped(members[1:]) {
		t.Error("乙已死，只有他的隊伍該算全滅")
	}
}

// 槽位越界不能寫到別人身上，也不能 panic。
func TestWriteBackPartyOutOfRange(t *testing.T) {
	members := []Character{{Name: "甲", CurrentHP: 20}}
	units := []*Unit{
		{Slot: PlayerSlotStart + 4, HP: 1, Side: SidePlayer}, // 隊伍只有 1 人
		nil,
	}
	WriteBackParty(members, units)
	if members[0].CurrentHP != 20 {
		t.Errorf("甲的血被改成 %d —— 越界的槽位不該寫到任何人身上", members[0].CurrentHP)
	}
}
