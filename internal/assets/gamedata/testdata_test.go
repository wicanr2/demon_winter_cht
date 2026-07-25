package gamedata

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// repoRoot 回傳這個套件所在的 repo 根目錄，用 runtime.Caller 反推，
// 不寫死絕對路徑，才能在任何簽出位置正常運作
// （internal/assets/gamedata -> 往上 3 層就是 repo 根目錄）。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("gamedata: runtime.Caller 取得目前檔案路徑失敗")
	}
	// file 是 .../internal/assets/gamedata/testdata_test.go
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

// origDataDir 回傳 workplace/orig/demwin/DEM_DATA 的絕對路徑，並在該目錄
// 不存在時（例如未附完整原始遊戲資料的簽出環境）自動 t.Skip，
// 不讓測試在缺原始檔案的環境失敗。
func origDataDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "workplace", "orig", "demwin", "DEM_DATA")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("gamedata: 找不到原始遊戲資料目錄 %s，略過需要真實檔案的測試: %v", dir, err)
	}
	return dir
}
