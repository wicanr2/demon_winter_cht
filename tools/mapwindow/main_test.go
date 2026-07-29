package main

import (
	"os"
	"path/filepath"
	"reflect"
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
	  }
	}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	normal, winter, err := loadThemeCoverage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normal, map[byte]bool{0x15: true, 0x2a: true}) {
		t.Fatalf("normal = %#v", normal)
	}
	if !reflect.DeepEqual(winter, map[byte]bool{0x16: true, 0x2b: true}) {
		t.Fatalf("winter = %#v", winter)
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
