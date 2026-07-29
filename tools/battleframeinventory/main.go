// Command battleframeinventory 盤點 MONSTER.DAT 外觀組與 Modern Icon frame 覆寫率。
//
// 怪物資料以 SpriteIndex 指向 MONSTER.SHE 的八幀外觀組；不同名稱可能共用同組。
// 美術量產應依外觀組而不是 99 個名稱重複製作。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/assets/gamedata"
)

type manifest struct {
	BattleSprites struct {
		Combat      map[string]string          `json:"combat"`
		Monsters    map[string]string          `json:"monsters"`
		MonsterSets map[string]json.RawMessage `json:"monsterSets"`
		Ships       map[string]string          `json:"ships"`
	} `json:"battleSprites"`
}

func main() {
	dataDir := flag.String("data", "workplace/orig/demwin/DEM_DATA", "原版資料目錄")
	manifestPath := flag.String("manifest", "artwork/modern-icon/m1/trial/theme.json",
		"Modern Icon theme.json；空字串表示只列原版外觀")
	flag.Parse()

	table, err := gamedata.LoadMonsterTable(filepath.Join(*dataDir, "MONSTER.DAT"))
	if err != nil {
		panic(err)
	}
	groups := groupMonsters(table.All())

	var covered map[int]bool
	if *manifestPath != "" {
		covered, err = loadMonsterCoverage(*manifestPath)
		if err != nil {
			panic(err)
		}
	}

	totalCovered := 0
	for _, sprite := range sortedKeys(groups) {
		names := groups[sprite]
		first, last := sprite*8, sprite*8+7
		n := 0
		for frame := first; frame <= last; frame++ {
			if covered[frame] {
				n++
			}
		}
		totalCovered += n
		fmt.Printf("sprite=%02d frames=%02x-%02x covered=%d/8 names=%s\n",
			sprite, first, last, n, strings.Join(names, "、"))
	}
	fmt.Printf("summary: monsters=%d appearances=%d frames=%d covered=%d\n",
		table.Len(), len(groups), len(groups)*8, totalCovered)
}

func groupMonsters(monsters []gamedata.Monster) map[int][]string {
	out := make(map[int][]string)
	for _, monster := range monsters {
		out[monster.SpriteIndex] = append(out[monster.SpriteIndex], monster.Name)
	}
	return out
}

func sortedKeys(groups map[int][]string) []int {
	keys := make([]int, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}

func loadMonsterCoverage(path string) (map[int]bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	out := make(map[int]bool, len(m.BattleSprites.Monsters))
	for key := range m.BattleSprites.Monsters {
		value := strings.TrimPrefix(key, "0x")
		frame, err := strconv.ParseUint(value, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("無效 monster frame %q: %w", key, err)
		}
		out[int(frame)] = true
	}
	for key := range m.BattleSprites.MonsterSets {
		value := strings.TrimPrefix(key, "0x")
		sprite, err := strconv.ParseUint(value, 16, 8)
		if err != nil || sprite >= 30 {
			return nil, fmt.Errorf("無效 monster set %q", key)
		}
		for frame := int(sprite) * 8; frame < int(sprite)*8+8; frame++ {
			out[frame] = true
		}
	}
	return out, nil
}
