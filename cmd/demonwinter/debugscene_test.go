package main

import (
	"strings"
	"testing"

	"github.com/wicanr2/demon_winter_cht/internal/game"
)

func TestFindDebugScene(t *testing.T) {
	got, err := findDebugScene("  CIRCLE-LIGHT ")
	if err != nil {
		t.Fatal(err)
	}
	if got.MapID != 5 || got.X != 11 || got.Y != 48 {
		t.Fatalf("circle-light = %+v", got)
	}
}

func TestTrapPoolScenePinsFacingWithoutMoving(t *testing.T) {
	got, err := findDebugScene("trap-pool")
	if err != nil {
		t.Fatal(err)
	}
	if got.MapID != 1 || got.X != 12 || got.Y != 17 ||
		!got.SetFacing || got.Facing != game.West {
		t.Fatalf("trap-pool = %+v", got)
	}
}

func TestFindDebugSceneRejectsUnknownName(t *testing.T) {
	if _, err := findDebugScene("winterfell"); err == nil {
		t.Fatal("未知名稱沒有回報錯誤")
	}
}

func TestDebugSceneListIsStableAndUseful(t *testing.T) {
	got := debugSceneList()
	if !strings.Contains(got, "armory") || !strings.Contains(got, "void-riddle") {
		t.Fatalf("清單漏掉代表場景：\n%s", got)
	}
	if strings.Index(got, "armory") > strings.Index(got, "void-riddle") {
		t.Fatalf("清單沒有依名稱排序：\n%s", got)
	}
}
