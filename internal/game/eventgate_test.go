package game

import (
	"os"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/scenario"
)

// 類別 1 是一次性的：值 0 播一次，標記成 1 之後永遠沒反應。
func TestEventGateClassOneIsOneShot(t *testing.T) {
	act, c := EventGate(scenario.SpecialClassEventA, 0, RereadIdle)
	if act != EventFire {
		t.Fatalf("類別 1 值 0 = %v，預期觸發", act)
	}
	if c != RereadIdle {
		t.Errorf("類別 1 不該武裝重讀，倒數 = %d", c)
	}
	// 看過之後：不觸發，而且**不武裝** —— 所以 R 也讀不回來。
	act, c = EventGate(scenario.SpecialClassEventA, 1, RereadIdle)
	if act != EventNone || c != RereadIdle {
		t.Errorf("類別 1 看過之後 = %v／倒數 %d，預期無事件且不武裝", act, c)
	}
	if _, ok := RequestReread(c); ok {
		t.Error("類別 1 看過之後按 R 竟然讀得到")
	}
}

// 類別 2 看過之後改成「站上去武裝、按 R 才重讀」。
func TestEventGateClassTwoIsRereadable(t *testing.T) {
	act, c := EventGate(scenario.SpecialClassEventB, 0, RereadIdle)
	if act != EventFire {
		t.Fatalf("類別 2 值 0 = %v，預期觸發", act)
	}

	// 看過之後再踩：不自動播，改成武裝。
	act, c = EventGate(scenario.SpecialClassEventB, 1, RereadIdle)
	if act != EventNone {
		t.Errorf("類別 2 看過之後自動播了（%v）", act)
	}
	if c != RereadArmed {
		t.Fatalf("倒數 = %d，預期武裝成 %d", c, RereadArmed)
	}

	// 按 R：倒數變 2，再查一次就放行。
	c, ok := RequestReread(c)
	if !ok || c != RereadRequested {
		t.Fatalf("按 R 之後 ok=%v 倒數=%d", ok, c)
	}
	act, c = EventGate(scenario.SpecialClassEventB, 1, c)
	if act != EventFire {
		t.Errorf("重讀沒觸發（%v）", act)
	}
	if c != RereadArmed {
		t.Errorf("重讀之後倒數 = %d，預期減回 %d", c, RereadArmed)
	}
}

// 走一步會把武裝清掉；但**不能清掉玩家剛按下的請求**。
func TestTickRereadOnlyClearsArmed(t *testing.T) {
	if got := TickReread(RereadArmed); got != RereadIdle {
		t.Errorf("武裝走一步 = %d，預期歸零", got)
	}
	if got := TickReread(RereadRequested); got != RereadRequested {
		t.Errorf("請求被走一步清掉了（%d）", got)
	}
	if got := TickReread(RereadIdle); got != RereadIdle {
		t.Errorf("閒置變成 %d", got)
	}
}

// 其餘類別照舊每次都觸發；類別 0 與傳送各走各的。
func TestEventGateOtherClasses(t *testing.T) {
	for _, c := range []struct {
		class, value int
		want         EventGateAction
		label        string
	}{
		{0, 0, EventNone, "類別 0（用掉了）"},
		{scenario.SpecialClassTrap, 0, EventFire, "陷阱"},
		{scenario.SpecialClassTrapAlt, 0, EventFire, "標記過的陷阱"},
		{scenario.SpecialClassLocationPlot, 3, EventFire, "地點劇情"},
		{scenario.SpecialClassTeleport, 0, EventTeleport, "傳送"},
	} {
		got, _ := EventGate(c.class, c.value, RereadIdle)
		if got != c.want {
			t.Errorf("%s = %v，預期 %v", c.label, got, c.want)
		}
	}
	// 地點劇情的「值」是 case 編號不是已造訪旗標 —— 值非 0 也要照樣觸發。
	if got, _ := EventGate(scenario.SpecialClassLocationPlot, 15, RereadIdle); got != EventFire {
		t.Errorf("地點劇情 case 15 = %v，預期觸發", got)
	}
}

// 出貨資料裡**類別 2 真的有值 1 的記錄** —— 「類別 2 會被標記成已造訪」
// 不是推論，而且「值 != 0」那條分支確實會走到。
//
// `1SS`／`2SS` 是壓片前有人玩過一段的存檔（`docs/re/78` §2），
// `3SS`–`5SS` 原封不動。
func TestShippedDataHasVisitedClassTwo(t *testing.T) {
	const dir = "../../workplace/orig/demwin/DEM_DATA"
	if _, err := os.Stat(dir); err != nil {
		t.Skip("沒有原版資料，跳過")
	}
	want := map[string]struct{ one, two int }{
		"1SS.DAT": {7, 9},
		"2SS.DAT": {5, 3},
		"3SS.DAT": {0, 0},
		"4SS.DAT": {0, 0},
		"5SS.DAT": {0, 0},
	}
	for name, w := range want {
		raw, err := os.ReadFile(dir + "/" + name)
		if err != nil {
			t.Fatalf("讀 %s：%v", name, err)
		}
		st, err := scenario.ParseSpecialTiles(raw)
		if err != nil {
			t.Fatalf("解析 %s：%v", name, err)
		}
		one, two := 0, 0
		for _, tile := range st.Tiles {
			if tile.Value() == 0 {
				continue
			}
			switch tile.Class() {
			case scenario.SpecialClassEventA:
				one++
			case scenario.SpecialClassEventB:
				two++
			}
		}
		if one != w.one || two != w.two {
			t.Errorf("%s：已造訪的類別 1／2 = %d／%d，預期 %d／%d",
				name, one, two, w.one, w.two)
		}
	}
}
