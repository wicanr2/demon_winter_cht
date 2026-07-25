#!/usr/bin/env python3
"""
Demon's Winter (SSI, 1988) — 35 個法術的 K/M 常數表定位與 dump 工具。

背景：法術效果公式 magnitude = RNG(K*SP/M)（見 docs/re/09-spells-and-actions.md
§4.2）的 K/M 常數表位置原本未知。本工具重現找表過程與最終結果，證據與交叉驗證
細節見 docs/re/15-spell-constants.md。

找表方法：`M`（見 §2 的判定式）已由程式碼證實 = 法術的最低 SP 投入
（translations/glossary.md 第 6 節「最低 SP」欄），可當 oracle。對
DEM_DATA/FILES.DAT 做「record 大小 × 欄位偏移」暴力掃描，找出讓「表中的 M
欄位」與「已知 35 個最低 SP 值中的 34 個」逐一吻合的位置。

用法：
    python3 tools/find_spell_constants.py [DEM_DATA 目錄路徑]

預設路徑：workplace/orig/demwin/DEM_DATA
只讀，不寫任何檔案，不修改 workplace/orig/。
"""
import os
import struct
import sys

DEFAULT_DIR = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    "workplace", "orig", "demwin", "DEM_DATA",
)

# spell_index(0-42, 對照 FILES.DTT 名稱+訊息字串的成對順序) -> glossary.md 記載的最低 SP。
# 只列出 35 個真實符文法術；24/25/26(幻術/召喚/附身類別)、38-42(Et Cetera/Poison/
# Youth/Power Leech/The End，非玩家可施放的符文法術) 不在此 oracle 內。
EXPECTED_MIN_SP = {
    0: 1, 1: 16, 2: 10, 3: 4, 4: 2, 5: 10, 6: 15, 7: 1, 8: 2, 9: 3, 10: 6,
    11: 11, 12: 1, 13: 4, 14: 7, 15: 1, 16: 3, 17: 9, 18: 3, 19: 20, 20: 1,
    21: 2, 22: 3, 23: 15, 27: 10, 28: 11, 29: 11, 30: 13, 31: 5, 32: 1,
    33: 9, 34: 3, 35: 3, 36: 2, 37: 25,
}

SPELL_NAMES = [
    "Column of Fire", "Flame Strike", "Fire Storm", "Flame Shield", "Sword",
    "Chains", "Death Blade", "Strength", "Armor", "Rust Armor", "Tempest",
    "Still Air", "Wings of Victory", "Wings", "Hail Storm", "Chill", "Slow",
    "Freeze", "Ice Shield", "Spirit Wrack", "Weaken", "Clumsiness",
    "Sanctuary", "Wither Strike", "Summon", "Illusion", "Possession",
    "Wind Walk", "Melt", "Break Bonds", "Freedom", "Breath of Life", "Heal",
    "Cure Poison", "Transference", "Magic Torch", "Crystalight",
    "Resurrect", "Et Cetera", "Poison", "Youth", "Power Leech", "The End",
]

SCHOOLS = {1: "Fire", 2: "Metal", 3: "Wind", 4: "Ice", 5: "Spirit", 6: "Summon/Illusion"}


def find_table(data, min_score=30):
    """對 data 做 record_size x field_offset 暴力掃描，回傳 (start, record_size,
    m_field_offset, score) 依 score 降冪排序的清單。"""
    results = []
    for rec_size in range(2, 21):
        max_idx = max(EXPECTED_MIN_SP)
        max_start = len(data) - (max_idx * rec_size + rec_size)
        if max_start < 0:
            continue
        for m_off in range(rec_size):
            for start in range(0, max_start + 1):
                score = 0
                for idx, val in EXPECTED_MIN_SP.items():
                    pos = start + idx * rec_size + m_off
                    if pos >= len(data):
                        break
                    if data[pos] == val:
                        score += 1
                if score >= min_score:
                    results.append((score, start, rec_size, m_off))
    results.sort(reverse=True)
    return results


def dump_table(data, start, rec_size=10):
    print(f"\n=== 完整 43 筆記錄（table_start=0x{start:x}, record_size={rec_size}) ===")
    print(f"{'idx':>3} {'name':18} {'school':>6} {'type':>5} {'K':>5} {'M':>5} {'w4':>4}")
    mismatches = []
    for i in range(43):
        off = start + i * rec_size
        w = struct.unpack_from("<5h", data, off)
        school, etype, k, m, w4 = w
        sname = SCHOOLS.get(school, str(school))
        exp = EXPECTED_MIN_SP.get(i)
        flag = ""
        if exp is not None and exp != m:
            flag = f"  <-- glossary 記載 M={exp}，不一致"
            mismatches.append((i, SPELL_NAMES[i], exp, m))
        print(f"{i:3d} {SPELL_NAMES[i]:18} {sname:>6} {etype:5d} {k:5d} {m:5d} {w4:4d}{flag}")
    return mismatches


def main():
    dem_dir = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_DIR
    path = os.path.join(dem_dir, "FILES.DAT")
    with open(path, "rb") as f:
        data = f.read()

    print(f"讀取 {path}（{len(data)} bytes）")
    print(f"用 {len(EXPECTED_MIN_SP)} 個已知最低 SP 值當 oracle，掃描 record_size=2..20、"
          f"field_offset=0..rec_size-1 的全部組合 ...")

    hits = find_table(data)
    if not hits:
        print("沒有找到任何 >=30/35 命中的組合。")
        return 1

    best_score, best_start, best_rec, best_off = hits[0]
    table_start = best_start  # field_offset 併入 start 後即為 table 起點（M 在 offset 6）
    print(f"\n最佳命中：record_size={best_rec}, m field offset={best_off}, "
          f"table_start=0x{best_start + best_off - 6 if best_off != 6 else best_start:x}, "
          f"score={best_score}/{len(EXPECTED_MIN_SP)}")

    # 正規化成 "table 從 0 開始、M 在 offset 6" 的慣例（與 docs/re/15 一致）
    canonical_start = best_start + best_off - 6
    mismatches = dump_table(data, canonical_start, best_rec)

    print(f"\n命中 {len(EXPECTED_MIN_SP) - len(mismatches)}/{len(EXPECTED_MIN_SP)}，"
          f"落差：{mismatches if mismatches else '無'}")
    print("完整交叉驗證與逐法術判讀見 docs/re/15-spell-constants.md")
    return 0


if __name__ == "__main__":
    sys.exit(main())
