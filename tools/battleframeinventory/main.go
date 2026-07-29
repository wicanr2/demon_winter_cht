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
		Combat      map[string]string                   `json:"combat"`
		Monsters    map[string]string                   `json:"monsters"`
		MonsterSets map[string]monsterDirectionManifest `json:"monsterSets"`
		Ships       map[string]string                   `json:"ships"`
	} `json:"battleSprites"`
}

type monsterDirectionManifest struct {
	South  string `json:"south"`
	SouthB string `json:"southB"`
	West   string `json:"west"`
	WestB  string `json:"westB"`
	East   string `json:"east"`
	EastB  string `json:"eastB"`
	North  string `json:"north"`
	NorthB string `json:"northB"`
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

	var covered, animated map[int]bool
	if *manifestPath != "" {
		covered, animated, err = loadMonsterCoverage(*manifestPath)
		if err != nil {
			panic(err)
		}
	}

	totalCovered, totalAnimated := 0, 0
	for _, sprite := range sortedKeys(groups) {
		names := groups[sprite]
		first, last := sprite*8, sprite*8+7
		n, a := 0, 0
		for frame := first; frame <= last; frame++ {
			if covered[frame] {
				n++
			}
			if animated[frame] {
				a++
			}
		}
		totalCovered += n
		totalAnimated += a
		fmt.Printf("sprite=%02d frames=%02x-%02x covered=%d/8 animated=%d/8 names=%s\n",
			sprite, first, last, n, a, strings.Join(names, "、"))
	}
	fmt.Printf("summary: monsters=%d appearances=%d frames=%d covered=%d animated=%d\n",
		table.Len(), len(groups), len(groups)*8, totalCovered, totalAnimated)
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

func loadMonsterCoverage(path string) (map[int]bool, map[int]bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, nil, err
	}
	out := make(map[int]bool, len(m.BattleSprites.Monsters))
	animated := make(map[int]bool)
	for key := range m.BattleSprites.Monsters {
		value := strings.TrimPrefix(key, "0x")
		frame, err := strconv.ParseUint(value, 16, 8)
		if err != nil {
			return nil, nil, fmt.Errorf("無效 monster frame %q: %w", key, err)
		}
		out[int(frame)] = true
	}
	for key, directions := range m.BattleSprites.MonsterSets {
		value := strings.TrimPrefix(key, "0x")
		sprite, err := strconv.ParseUint(value, 16, 8)
		if err != nil || sprite >= 30 {
			return nil, nil, fmt.Errorf("無效 monster set %q", key)
		}
		for frame := int(sprite) * 8; frame < int(sprite)*8+8; frame++ {
			out[frame] = true
		}
		for direction, phaseB := range []string{
			directions.SouthB, directions.WestB, directions.EastB, directions.NorthB,
		} {
			if phaseB == "" {
				continue
			}
			base := int(sprite)*8 + direction*2
			animated[base], animated[base+1] = true, true
		}
	}
	return out, animated, nil
}
