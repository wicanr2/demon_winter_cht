package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

func TestGroupMonstersAndSortedKeys(t *testing.T) {
	groups := groupMonsters([]gamedata.Monster{
		{Name: "Orc", SpriteIndex: 17},
		{Name: "Goblin", SpriteIndex: 17},
		{Name: "Rat", SpriteIndex: 19},
		{Name: "Fighter", SpriteIndex: 0},
	})
	if got, want := sortedKeys(groups), []int{0, 17, 19}; !reflect.DeepEqual(got, want) {
		t.Fatalf("keys = %v，預期 %v", got, want)
	}
	if got, want := groups[17], []string{"Orc", "Goblin"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sprite 17 names = %v，預期 %v", got, want)
	}
}

func TestLoadMonsterCoverageExpandsSets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "theme.json")
	raw := []byte(`{
		"battleSprites": {
			"monsters": {"0x8d": "orc.png"},
			"monsterSets": {"0x03": {
				"south": "s.png", "west": "w.png", "east": "e.png", "north": "n.png"
			}}
		}
	}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	covered, err := loadMonsterCoverage(path)
	if err != nil {
		t.Fatal(err)
	}
	for frame := 0x18; frame <= 0x1f; frame++ {
		if !covered[frame] {
			t.Errorf("monster set frame %#02x 未展開", frame)
		}
	}
	if !covered[0x8d] {
		t.Error("個別 monster frame 未保留")
	}
}
