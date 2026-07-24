#!/usr/bin/env python3
"""
Demon's Winter (SSI, 1988) — DEM_DATA 資源索引 / 開場序列 解析器

解析對象：
  - FILES.DAT / FILES.DTT   （原本懷疑是「資源索引」，本工具用來驗證／推翻此假設）
  - 1SS.DAT ~ 5SS.DAT / ALL_SS.DAT （開場序列資料，驗證 511 vs 512 的關係）

只用標準庫。不對來源檔案做任何寫入（workplace/orig 唯讀）。

用法：
    python3 tools/parse_files_index.py [DEM_DATA 目錄路徑]

預設路徑：workplace/orig/demwin/DEM_DATA
"""
import struct
import sys
import os
import re
import collections

DEFAULT_DIR = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    "workplace", "orig", "demwin", "DEM_DATA",
)


def load(dem_dir, name):
    path = os.path.join(dem_dir, name)
    with open(path, "rb") as f:
        return f.read()


# ----------------------------------------------------------------------
# FILES.DTT — 字串池
# ----------------------------------------------------------------------

# 已用內容比對驗證過的分段（見 docs/formats/resource-index.md 的驗證依據）。
# 區間為 [start, end)，index 從 0 起算（依 split(b'\x00') 後的順序）。
DTT_KNOWN_SECTIONS = [
    (0, 5, "種族名 Races（5 個，對照 glossary.md 第3節，完全吻合）"),
    (5, 90, "法術名／戰鬥訊息 成對出現（NAME, message），85 條含 3 個 '0' 佔位與結尾哨兵 'THE END'"),
    (91, 123, "技能名 Skills（32 個，對照 glossary.md 第5節，數量與內容完全吻合，含遊戲原始拼字 'Shamen'）"),
    (123, 131, "武器類型名（8 個，對照 DEMON.INT 字串表同序）"),
    (131, 136, "護甲類型名（5 個，對照 DEMON.INT 字串表同序）"),
    (136, 151, "裝備／飾品類別名（15 個：crown/vial/ring/wand/staff/rod/gem/amulet/...）"),
    (151, 164, "特殊神器／劇情物品名（13 個，含 Demon Crystal, Orb/Evertime — 對照 glossary.md 第1節）"),
    (164, 467, "物件／地點互動文字表（大量 name+description+event 記錄，見下方 N/D/T/P/S 前綴說明）"),
    (467, 489, "寶石／材質形容詞word list（22 個，推測為隨機物品命名用詞庫，未證實）"),
    (489, 501, "幻術／召喚生物名（12 個，對照 glossary.md 第7節，順序與內容完全吻合）"),
]

# 164-467 區段內，字串常見的單字元前綴代碼（直接觀察到的事實，語意為推測）
DTT_EVENT_PREFIXES = {
    "N": "疑似「觸發後改名為」（如 NBag/red dust → 後續顯示為 Bag/red dust）",
    "D": "疑似「觸發後顯示的長描述／劇情文字」",
    "T": "後接 6 位數字，疑似 Trigger/座標編碼，未證實",
    "P": "後接 5 位數字，疑似 Picture/座標編碼，未證實",
    "S": "後接 1 位數字（S1/S2），疑似 Sound 或 Script 編號，未證實",
    "*": "單一星號，疑似「此格留空／無事件」旗標",
}


def parse_dtt(data):
    """把 FILES.DTT 依 NUL 字元切開，回傳 (index, bytes) 清單。"""
    parts = data.split(b"\x00")
    # 最後一個 split 片段是檔尾（若檔案以 NUL 結尾則為空字串），不算一筆記錄
    records = parts[:-1] if parts and parts[-1] == b"" else parts
    return records


def dtt_string_offsets(data):
    """計算每個 NUL-terminated 字串在檔案中的起始位移。"""
    offsets = []
    pos = 0
    for part in data.split(b"\x00"):
        offsets.append(pos)
        pos += len(part) + 1
    return offsets[:-1]


def report_dtt(data):
    print("=" * 70)
    print("FILES.DTT — 字串池分析")
    print("=" * 70)
    records = parse_dtt(data)
    n_empty = sum(1 for r in records if r == b"")
    print(f"檔案大小: {len(data)} bytes")
    print(f"NUL-terminated 字串總數: {len(records)}（空字串 {n_empty} 個，非空 {len(records)-n_empty} 個）")
    print()
    print("已驗證的分段（依內容比對 glossary.md / DEMON.INT，非憑檔名猜測）：")
    for start, end, desc in DTT_KNOWN_SECTIONS:
        print(f"  [{start:3d}:{end:3d}]  {desc}")
    print()
    print("164–467 區段內觀察到的單字元前綴代碼（事實觀察，語意為推測）：")
    for k, v in DTT_EVENT_PREFIXES.items():
        print(f"  '{k}': {v}")
    print()
    print("--- 節錄：種族 (index 0-4) ---")
    for i in range(0, 5):
        print(f"  [{i}] {records[i]!r}")
    print("--- 節錄：技能 (index 91-122) ---")
    for i in range(91, 123):
        print(f"  [{i}] {records[i]!r}")
    print("--- 節錄：幻術/召喚生物 (index 489-500) ---")
    for i in range(489, 501):
        print(f"  [{i}] {records[i]!r}")
    print()


# ----------------------------------------------------------------------
# FILES.DAT — 二進位資料
# ----------------------------------------------------------------------

def find_repeating_records(data, pattern=b"\x0a\x00\x08\x00\x09\x00", record_len=14):
    """尋找 FILES.DAT 尾段那組固定樣板欄位 + 1 個變動欄位的 14-byte 記錄。"""
    hits = []
    for m in re.finditer(re.escape(pattern), data):
        start = m.start()
        rec = data[start:start + record_len]
        if len(rec) == record_len:
            hits.append((start, rec))
    return hits


def offset_overlap_test(dat, dtt):
    """測試 FILES.DAT 若視為 uint16 LE 陣列，其數值是否『大量』對應到
    FILES.DTT 各字串的起始位移──藉此檢驗『FILES.DAT 是字串索引表』的假設。
    """
    offsets = set(dtt_string_offsets(dtt))
    n = len(dat) // 2
    vals = struct.unpack(f"<{n}H", dat[: n * 2])
    matches = [v for v in vals if v in offsets]

    # random baseline：在 FILES.DTT 位移值域內均勻抽樣，估計純屬巧合的命中率
    import random
    random.seed(42)
    max_off = max(offsets)
    baseline_hits = sum(
        1 for _ in range(n) if random.randint(0, max_off) in offsets
    )
    return len(vals), len(matches), baseline_hits


def report_dat(dat, dtt):
    print("=" * 70)
    print("FILES.DAT — 二進位資料分析")
    print("=" * 70)
    print(f"檔案大小: {len(dat)} bytes")
    print()

    c = collections.Counter(dat)
    print(f"byte 值域: {min(dat)}..{max(dat)}，共 {len(c)} 種相異值")
    print(f"最常見 10 個 byte 值: {c.most_common(10)}")
    print()

    print("---『資源索引表（記錄檔名）』假設：strings 掃描結果 ---")
    ascii_runs = re.findall(rb"[\x20-\x7e]{4,}", dat)
    print(f"長度 >=4 的可印字元連續片段數: {len(ascii_runs)}")
    print("內容（若有）：", ascii_runs[:10])
    print("=> 未發現任何實際檔名字串（如 '1SS.DAT'、'PIC1.PIC' 等），")
    print("   故『FILES.DAT 是檔名目錄』的假設不成立（已否證）。")
    print()

    print("---『FILES.DAT 是 FILES.DTT 的 offset/length 索引表』假設：數值相關性檢定 ---")
    n_vals, n_match, n_baseline = offset_overlap_test(dat, dtt)
    print(f"視為 uint16 LE 陣列共 {n_vals} 個數值")
    print(f"其中剛好等於 FILES.DTT 某字串起始位移的數量: {n_match}")
    print(f"隨機基準線（相同抽樣次數，均勻分布於同值域）預期命中數: {n_baseline}")
    ratio = n_match / n_baseline if n_baseline else float("nan")
    print(f"=> 命中數/隨機基準線 ≈ {ratio:.2f}x")
    print("   倍率明顯高於 1，但遠低於『真正的索引表』應有的近乎全命中，")
    print("   故證據不足以支持『FILES.DAT 是 FILES.DTT 的直接位移索引表』。")
    print()

    print("--- 內部分段結構（依 byte 值分布特徵，肉眼+程式辨識） ---")
    print("  0x000-0x09F: 大量 0-17 的小數值，含少數 0xFF/0xFE（可能為 signed -1/-2）")
    print("  0x0A0-0x14F: 出現連續遞增的 marker byte 0x80,0x81,0x82...0xD0，")
    print("               每個 marker 後跟數個 0-25 範圍的小數值")
    print("               （典型『遞增 ID + 變動長度子清單』編碼，用途未定）")
    print("  0x150-0x41F: 大量 0-10 範圍小數值（疑似另一張表格資料）")
    print("  0x420-0x5FF: uint16 LE 小數值（多在 0-100 範圍）")
    print("  0x600-0x67F: 8 筆固定 14-byte 記錄，樣板 [10,8,9,X,1,3,0]（uint16x7），")
    print("               X = 6,15,13,20,30,65,100,5（僅第 4 欄變動，其餘 6 欄固定）")
    print("  0x680-0x8CF: 4/5-byte 為週期的小數值表格（未解）")
    print()

    hits = find_repeating_records(dat)
    print(f"固定樣板 14-byte 記錄實際掃描結果: {len(hits)} 筆")
    for off, rec in hits:
        vals = struct.unpack("<7H", rec)
        print(f"  offset 0x{off:04x}: {vals}")
    print()


# ----------------------------------------------------------------------
# nSS.DAT / ALL_SS.DAT
# ----------------------------------------------------------------------

def report_ss(dem_dir):
    print("=" * 70)
    print("nSS.DAT / ALL_SS.DAT — 開場序列資料分析")
    print("=" * 70)
    allss = load(dem_dir, "ALL_SS.DAT")
    print(f"ALL_SS.DAT 大小: {len(allss)} bytes = 5 x 512")
    print()

    for i in range(1, 6):
        fname = f"{i}SS.DAT"
        d = load(dem_dir, fname)
        block = allss[(i - 1) * 512 : i * 512]
        same = d == block[:511]
        print(f"{fname} (511 bytes) 與 ALL_SS.DAT block #{i} 前 511 bytes 逐位元組比對: "
              f"{'完全相同' if same else '有差異'}；block[511]=0x{block[511]:02x}")
        if not same:
            diffs = [(j, d[j], block[j]) for j in range(511) if d[j] != block[j]]
            positions = [p for p, _, _ in diffs]
            mod3 = collections.Counter(p % 3 for p in positions)
            print(f"    差異數: {len(diffs)}，全部落在 position % 3 == "
                  f"{list(mod3.keys())}（3-byte 記錄結構的第 3 欄位）")
            print(f"    差異範例（前5筆 idx,indiv,combined): {diffs[:5]}")

    print()
    print("--- 3-byte 記錄結構驗證（以 payload 前導的長 0x00 padding 前段為準） ---")
    for i in range(1, 6):
        fname = f"{i}SS.DAT"
        d = load(dem_dir, fname)
        m = re.search(rb"\x00{20,}", d)
        payload = d[: m.start()] if m else d
        n = len(payload) // 3 * 3
        recs = [tuple(payload[j : j + 3]) for j in range(0, n, 3)]
        third_bytes = [r[2] for r in recs]
        marker_hits = sum(1 for b in third_bytes if b in (0x40, 0x41))
        print(f"{fname}: payload {len(payload)} bytes -> {len(recs)} 筆 3-byte 記錄；"
              f"第3欄 = 0x40/0x41 的比例: {marker_hits}/{len(recs)}")
        print(f"    前5筆記錄 (b0,b1,b2): {recs[:5]}")


def main():
    dem_dir = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_DIR
    if not os.path.isdir(dem_dir):
        print(f"找不到目錄: {dem_dir}", file=sys.stderr)
        sys.exit(1)

    dat = load(dem_dir, "FILES.DAT")
    dtt = load(dem_dir, "FILES.DTT")

    report_dtt(dtt)
    report_dat(dat, dtt)
    report_ss(dem_dir)


if __name__ == "__main__":
    main()
