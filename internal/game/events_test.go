package game

import (
	"testing"
)

func TestTriggerFor(t *testing.T) {
	cases := []struct {
		tile byte
		want TriggerKind
	}{
		{0x11, TriggerLookup},
		{0x53, TriggerLookup},
		{0x35, TriggerPool},
		{0x25, TriggerSite},
		{0x26, TriggerSite},
		{0x2e, TriggerSite},
		{0x5b, TriggerSite},
		{0x64, TriggerSite}, // Ghidra 漏掉的那一個
		{0x00, TriggerNone},
		{0x01, TriggerNone},
		{0x7f, TriggerNone},
	}
	for _, c := range cases {
		if got := TriggerFor(c.tile); got != c.want {
			t.Errorf("tile 0x%02x：得到 %d，預期 %d", c.tile, got, c.want)
		}
	}
}
