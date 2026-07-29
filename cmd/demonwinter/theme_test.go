package main

import "testing"

func TestThemeCycle(t *testing.T) {
	got := themeEGA
	want := []themeID{themeCGA, themeModern, themeEGA, themeCGA}
	for i, expected := range want {
		got = nextThemeID(got)
		if got != expected {
			t.Fatalf("第 %d 次 F8 = %q，預期 %q", i+1, got, expected)
		}
	}
}
