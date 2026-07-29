# Modern Icon M1：高解析世界地圖呈現層試驗

這裡驗證 Modern Icon 可在 `1280×800` 最終畫布直接覆寫地圖格，規則層仍使用
原版索引、碰撞與事件。loader 只接受逐索引列出的 `64×56` 不透明 PNG，並明確
拒絕舊的 `32×28` runtime trial。

`masters/normal-forest.png` 是依核准 concept 重新構圖的獨立森林母稿，不是從
contact sheet 裁切。`trial/normal-forest.png` 是首次 runtime 構圖試驗；目前
刻意掛在已知大量出現的 `0x23` 上，只為證明高解析覆寫位置、裁切與 F8 管線，
**不是索引語意核准或正式森林 atlas**。

以 `-modern-icon-dir artwork/modern-icon/m1/trial` 載入。未列出的格不會被舊圖
冒充 Modern Icon，而是保留底下的相容預覽，直到每個索引真正重畫並通過同場景
審核。下一批必須先建立實際畫面索引盤點，再分別重畫平原、深水、岸線、森林、
城鎮與隊伍，不能把同一張圖掛到不同語意。
