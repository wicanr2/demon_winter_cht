package main

import (
	"strings"
	"testing"
)

func validModernIconManifest() modernIconManifest {
	m := modernIconManifest{
		Schema: modernIconSchema, ID: "modern-icon",
		FrameWidth: modernIconWidth, FrameHeight: modernIconHeight,
	}
	m.Tiles.Normal = map[string]string{"0x01": "normal-01.png"}
	m.Tiles.Winter = map[string]string{"0x01": "winter-01.png"}
	return m
}

func TestValidateModernIconManifest(t *testing.T) {
	if err := validateModernIconManifest(validModernIconManifest()); err != nil {
		t.Fatalf("有效 manifest 被拒絕：%v", err)
	}
	tests := []struct {
		name string
		edit func(*modernIconManifest)
	}{
		{"schema", func(m *modernIconManifest) { m.Schema++ }},
		{"id", func(m *modernIconManifest) { m.ID = "modern-ega" }},
		{"width", func(m *modernIconManifest) { m.FrameWidth = 32 }},
		{"height", func(m *modernIconManifest) { m.FrameHeight = 28 }},
		{"empty", func(m *modernIconManifest) {
			m.Tiles.Normal, m.Tiles.Winter = nil, nil
		}},
		{"index", func(m *modernIconManifest) {
			m.Tiles.Normal = map[string]string{"0xff": "bad.png"}
		}},
		{"path", func(m *modernIconManifest) {
			m.Tiles.Normal = map[string]string{"0x01": "../normal.png"}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := validModernIconManifest()
			tc.edit(&m)
			if err := validateModernIconManifest(m); err == nil {
				t.Fatal("無效 manifest 未被拒絕")
			}
		})
	}
}

func TestModernIconRejectsLegacyFrameSize(t *testing.T) {
	m := validModernIconManifest()
	m.FrameWidth, m.FrameHeight = 32, 28
	err := validateModernIconManifest(m)
	if err == nil || !strings.Contains(err.Error(), "64x56") {
		t.Fatalf("32×28 拒絕訊息 = %v", err)
	}
}
