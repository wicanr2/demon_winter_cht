package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadThemeCoverageIncludesTilesAndVariants(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "theme.json")
	raw := []byte(`{
	  "tiles":{"normal":{"0x15":"n.png"},"winter":{"0x16":"w.png"}},
	  "tileVariants":{
	    "normal":{"0x2a":["a.png","b.png"]},
	    "winter":{"0x2b":["a.png","b.png"]}
	  },
	  "dungeonTiles":{"0x0d":"wall.png"}
	}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	normal, winter, dungeon, err := loadThemeCoverage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normal, map[byte]bool{0x15: true, 0x2a: true}) {
		t.Fatalf("normal = %#v", normal)
	}
	if !reflect.DeepEqual(winter, map[byte]bool{0x16: true, 0x2b: true}) {
		t.Fatalf("winter = %#v", winter)
	}
	if !reflect.DeepEqual(dungeon, map[byte]bool{0x0d: true}) {
		t.Fatalf("dungeon = %#v", dungeon)
	}
}

func TestWriteInventoryJSONSeparatesDungeonNamespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	uses := map[byte]tileUse{
		0x0d: {count: 3, mapID: 1, x: 2, y: 4, byMap: map[int]int{1: 2, 3: 1}},
	}
	passability := map[byte]byte{0x0d: 0xff}
	if err := writeInventoryJSON(path, []int{1, 3}, []int{0x0d}, uses, passability); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `"namespace": "dungeon"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("JSON 沒有 %s：%s", want, raw)
	}
	if !strings.Contains(string(raw), `"3": 1`) {
		t.Fatalf("JSON 沒有逐地圖計數：%s", raw)
	}
}

func TestInventoryBehaviorUsesOriginalPassabilitySemantics(t *testing.T) {
	tests := []struct {
		index, passability byte
		want               string
	}{
		{0x24, 0xfd, "exit"},
		{0x14, 0xff, "submap-floor"},
		{0x62, 0xff, "submap-floor"},
		{0x0d, 0xff, "blocked"},
		{0x00, 0x04, "terrain-4"},
		{0x58, 0xfe, "special"},
	}
	for _, tc := range tests {
		if got := inventoryBehavior(tc.index, tc.passability); got != tc.want {
			t.Errorf("index %#02x passability %#02x = %q，預期 %q",
				tc.index, tc.passability, got, tc.want)
		}
	}
}

func TestFormatTileList(t *testing.T) {
	if got := formatTileList(nil); got != " none" {
		t.Fatalf("空清單 = %q", got)
	}
	if got := formatTileList([]int{0x15, 0x2f}); got != " 15 2f" {
		t.Fatalf("清單 = %q", got)
	}
}

func TestCheckDungeonReview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	var elements []string
	for i := 0; i < 12; i++ {
		elements = append(elements, fmt.Sprintf(
			`{"id":"e%d","label":"元素%d","row":%d,"column":%d,`+
				`"decision":"pending","batch":"D1","mustPreserve":["規則"]}`,
			i, i, i/4+1, i%4+1))
	}
	raw := fmt.Sprintf(`{"schema":1,"directionImage":"direction.png",`+
		`"status":"pending","elements":[%s]}`, strings.Join(elements, ","))
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkDungeonReview(path); err != nil {
		t.Fatal(err)
	}
}

func TestCheckDungeonReviewRejectsDuplicateGridCell(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	raw := `{"schema":1,"directionImage":"direction.png","status":"pending","elements":[`
	for i := 0; i < 12; i++ {
		if i != 0 {
			raw += ","
		}
		raw += fmt.Sprintf(
			`{"id":"e%d","label":"元素","row":1,"column":1,`+
				`"decision":"pending","batch":"D1","mustPreserve":["規則"]}`, i)
	}
	raw += `]}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkDungeonReview(path); err == nil {
		t.Fatal("重複格位沒有被拒絕")
	}
}

func TestCheckDungeonReviewRejectsInconsistentApproval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	var elements []string
	for i := 0; i < 12; i++ {
		decision := "approved"
		if i == 11 {
			decision = "pending"
		}
		elements = append(elements, fmt.Sprintf(
			`{"id":"e%d","label":"元素","row":%d,"column":%d,`+
				`"decision":%q,"batch":"D1","mustPreserve":["規則"]}`,
			i, i/4+1, i%4+1, decision))
	}
	raw := fmt.Sprintf(`{"schema":1,"directionImage":"direction.png",`+
		`"status":"approved","elements":[%s]}`, strings.Join(elements, ","))
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkDungeonReview(path); err == nil {
		t.Fatal("總狀態 approved 與 pending 元素不一致時沒有被拒絕")
	}
}

func TestMissingThemeCoverageDoesNotCrossNamespaces(t *testing.T) {
	keys := []int{0x01, 0x0d, 0x23}
	worldUses := map[byte]bool{0x01: true, 0x23: true}
	dungeonUses := map[byte]bool{0x0d: true}
	worldCovered := map[byte]bool{0x01: true}
	dungeonCovered := map[byte]bool{0x0d: true}

	if got := missingThemeCoverage(keys, worldUses, worldCovered); !reflect.DeepEqual(got, []int{0x23}) {
		t.Fatalf("世界缺格 = %#v，預期只缺 0x23", got)
	}
	if got := missingThemeCoverage(keys, dungeonUses, dungeonCovered); len(got) != 0 {
		t.Fatalf("地城不應被世界索引污染，卻缺 %#v", got)
	}
}
