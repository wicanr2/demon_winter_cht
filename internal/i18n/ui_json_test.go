package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadUICatalogJSONRejectsDuplicateKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui.json")
	body := `{"locale":"zh-Hant","entries":[
		{"key":"same.key","text":"甲"},
		{"key":"same.key","text":"乙"}
	]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadUICatalogJSON(path); err == nil || !strings.Contains(err.Error(), "重複 key") {
		t.Fatalf("重複 key 應失敗，得到 %v", err)
	}
}

func TestLoadUICatalogJSONRejectsEmptyText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui.json")
	body := `{"locale":"zh-Hant","entries":[{"key":"empty.text","text":""}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadUICatalogJSON(path); err == nil || !strings.Contains(err.Error(), "沒有 text") {
		t.Fatalf("空白 text 應失敗，得到 %v", err)
	}
}

func TestWriteAndLoadUICatalogJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui.json")
	c := &Catalog{Entries: []Entry{
		{Name: "b.key", Source: "B", Target: "乙"},
		{Name: "a.key", Target: "甲"},
	}}
	if err := WriteUICatalogJSON(path, "zh-Hant", c); err != nil {
		t.Fatal(err)
	}
	got, err := LoadUICatalogJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 2 || got.Entries[0].Key != "a.key" || got.Entries[1].Key != "b.key" {
		t.Fatalf("JSON 應依 key 排序，得到 %#v", got.Entries)
	}
}
