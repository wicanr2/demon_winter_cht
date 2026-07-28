package main

import (
	"reflect"
	"testing"
)

func TestParseWorldCoord(t *testing.T) {
	got, err := parseWorldCoord("34:28,50")
	if err != nil {
		t.Fatalf("parseWorldCoord：%v", err)
	}
	if want := (wpoint{34, 28, 50}); got != want {
		t.Errorf("得到 %v，預期 %v", got, want)
	}
	for _, bad := range []string{"28,50", "34:28", "x:1,2", "34:a,2"} {
		if _, err := parseWorldCoord(bad); err == nil {
			t.Errorf("%q 應該要報錯", bad)
		}
	}
}

// 換圖那一步的方向要從**子地圖編號差**看，不能從座標差看 ——
// 座標已經 wrap 到另一側了（往東跨圖是 59 → 4，座標差是 −55 不是 +1），
// 照座標算會印出反方向的按鍵，而腳本照樣跑得完，只是走到別的地方。
func TestToWorldScript_CrossingDirectionComesFromMapID(t *testing.T) {
	cases := []struct {
		name string
		path []wpoint
		want []string
	}{
		{
			"往東跨圖",
			[]wpoint{{34, 58, 50}, {34, 59, 50}, {44, 4, 50}, {44, 5, 50}},
			[]string{"rep 2 Right", "# → 子地圖 44 的 (4,50)", "rep 1 Right"},
		},
		{
			"往南跨圖",
			[]wpoint{{34, 20, 59}, {35, 20, 4}},
			[]string{"rep 1 Down", "# → 子地圖 35 的 (20,4)"},
		},
		{
			"往西跨圖",
			[]wpoint{{34, 5, 20}, {24, 59, 20}},
			[]string{"rep 1 Left", "# → 子地圖 24 的 (59,20)"},
		},
		{
			"往北跨圖",
			[]wpoint{{34, 20, 5}, {33, 20, 59}},
			[]string{"rep 1 Up", "# → 子地圖 33 的 (20,59)"},
		},
		{
			"同一張圖內轉向",
			[]wpoint{{34, 10, 10}, {11, 10, 10}},
			nil, // 編號差 −23，不是四個方向之一 —— 不該猜一個方向出來
		},
	}
	for _, c := range cases {
		got := toWorldScript(c.path)
		if c.want == nil {
			// 這一筆只要求「不要編出方向」：唯一那行是註解。
			if len(got) != 1 || got[0] != "# → 子地圖 11 的 (10,10)" {
				t.Errorf("%s：得到 %q", c.name, got)
			}
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s：\n得到 %q\n預期 %q", c.name, got, c.want)
		}
	}
}
