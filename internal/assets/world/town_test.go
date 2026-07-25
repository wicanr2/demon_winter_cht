package world

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTown_All25Files(t *testing.T) {
	dir := origDataDir(t)

	for i := 1; i <= 25; i++ {
		i := i
		name := fmt.Sprintf("TOWN%d.DAT", i)
		t.Run(name, func(t *testing.T) {
			town, err := LoadTown(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("LoadTown(%s) 失敗: %v", name, err)
			}
			if len(town.Records) != townRecordCount {
				t.Fatalf("%s Records 長度 = %d, want %d", name, len(town.Records), townRecordCount)
			}
		})
	}
}

func TestLoadTown_Town1KnownAnchors(t *testing.T) {
	dir := origDataDir(t)
	town, err := LoadTown(filepath.Join(dir, "TOWN1.DAT"))
	if err != nil {
		t.Fatalf("LoadTown 失敗: %v", err)
	}

	// 錨點取自 docs/formats/town-and-map.md §1.2 的 TOWN1.DAT 逐筆 dump：
	// rec 0 code=0x00 payload=00x13 tail=0a 01 01
	rec0 := town.Records[0]
	if rec0.Code != 0x00 {
		t.Errorf("rec0.Code = 0x%02x, want 0x00", rec0.Code)
	}
	wantTail := [3]byte{0x0a, 0x01, 0x01}
	if rec0.Tail != wantTail {
		t.Errorf("rec0.Tail = %v, want %v", rec0.Tail, wantTail)
	}
	for _, b := range rec0.Payload {
		if b != 0 {
			t.Errorf("rec0.Payload 預期全零，出現非零值: %v", rec0.Payload)
			break
		}
	}

	// rec 7 code=0x19(25) payload=00 00 00 00 c8 00 0d 06 00 00 00 00 00  tail=00 01 01
	rec7 := town.Records[7]
	if rec7.Code != 0x19 {
		t.Errorf("rec7.Code = 0x%02x, want 0x19", rec7.Code)
	}
	wantPayload7 := [13]byte{0x00, 0x00, 0x00, 0x00, 0xc8, 0x00, 0x0d, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00}
	if rec7.Payload != wantPayload7 {
		t.Errorf("rec7.Payload = %v, want %v", rec7.Payload, wantPayload7)
	}
	wantTail7 := [3]byte{0x00, 0x01, 0x01}
	if rec7.Tail != wantTail7 {
		t.Errorf("rec7.Tail = %v, want %v", rec7.Tail, wantTail7)
	}
}

func TestLoadTown_WrongSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "BAD.DAT")
	if err := os.WriteFile(path, make([]byte, 10), 0o644); err != nil {
		t.Fatalf("寫測試檔失敗: %v", err)
	}
	if _, err := LoadTown(path); err == nil {
		t.Error("LoadTown 對長度不符的檔案預期回傳 error，卻沒有")
	}
}

func TestLoadTown_MissingFile(t *testing.T) {
	if _, err := LoadTown("/nonexistent/path/TOWN1.DAT"); err == nil {
		t.Error("LoadTown 對不存在的檔案預期回傳 error，卻沒有")
	}
}
