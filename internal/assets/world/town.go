package world

import (
	"fmt"
	"os"
)

// townRecordSize、townRecordCount、townTrailerSize、townFileSize 是
// TOWN*.DAT 的固定佈局：30 筆 17-byte 記錄在前，2 bytes 在檔尾，共
// 512 bytes（已驗證，見 docs/formats/town-and-map.md §1.1：切成 30*17
// 的分法讓 §1.1 的 3-byte 錨點序列在 96% 記錄裡精準落在記錄尾端 3
// bytes，不是巧合）。
const (
	townRecordSize  = 17
	townRecordCount = 30
	townTrailerSize = 2
	townFileSize    = townRecordCount*townRecordSize + townTrailerSize // 512
)

// TownRecord 是單筆 17-byte 城鎮記錄的原始佈局：
//
//	偏移(記錄內)  長度   欄位
//	0             1      Code    類型碼，語意未解（28 種相異值，見 docs/formats/town-and-map.md §1.2）
//	1–13          13     Payload 多為 0，有時是數值（例如售價）
//	14–16         3      Tail    尾端 3 bytes，值不固定，語意未解
//
// **注意**：25 個 TOWN*.DAT 裡有 14 個在段落 B（record 11–28 附近）
// 混入了 SSI「Town Maker」編輯工具留下的未初始化緩衝區殘留（Applesoft
// BASIC 碎片、"ELRIC" 測試角色樣板，見 docs/formats/town-and-map.md
// §1.3）。這不是遊戲內容，本型別刻意**不**替這段資料賦予任何語意
// （沒有 NPC／名單專屬欄位），一律以原始 Code/Payload/Tail 呈現，
// 呼叫端若要過濾垃圾資料需自行依上游文件的判斷式處理。
type TownRecord struct {
	Code    byte
	Payload [13]byte
	Tail    [3]byte
}

// Town 是一座城鎮的完整資料：30 筆記錄 + 2 bytes 檔尾。
//
// 建立方式一律透過 LoadTown，零值不可用。
type Town struct {
	// Records 是 30 筆原始記錄，順序與檔案內原始順序一致。
	Records [townRecordCount]TownRecord
	// Trailer 是檔尾 2 bytes。多數城鎮是 0x0000，但已知有例外
	// （TOWN5/6.DAT=0x504f、TOWN7.DAT=0x00e1、TOWN25.DAT=0x0002），
	// 不是恆定值，語意未解（見 docs/formats/town-and-map.md §1.2 末段）。
	Trailer [townTrailerSize]byte
}

// LoadTown 解析指定路徑的城鎮定義檔（TOWN1.DAT..TOWN25.DAT）。
//
// 格式（已驗證，見 docs/formats/town-and-map.md §1.1）：512 bytes =
// 30 筆 17-byte 記錄 + 2 bytes 檔尾（不是先前假設的「2 bytes header
// 在檔頭」）。解析失敗一律回傳 error，不 panic；28 個 type_code 的語意
// 未解，本函式保留原始值即可，不嘗試推測。
func LoadTown(path string) (*Town, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("world: 讀取 %s 失敗: %w", path, err)
	}
	if len(data) != townFileSize {
		return nil, fmt.Errorf("world: %s 長度 %d 不等於預期的 %d (%d*%d + %d)",
			path, len(data), townFileSize, townRecordCount, townRecordSize, townTrailerSize)
	}

	var t Town
	for i := 0; i < townRecordCount; i++ {
		off := i * townRecordSize
		rec := TownRecord{Code: data[off]}
		copy(rec.Payload[:], data[off+1:off+14])
		copy(rec.Tail[:], data[off+14:off+17])
		t.Records[i] = rec
	}
	copy(t.Trailer[:], data[townRecordCount*townRecordSize:])

	return &t, nil
}
