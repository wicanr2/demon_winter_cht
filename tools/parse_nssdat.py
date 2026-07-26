#!/usr/bin/env python3
"""解析 `nSS.DAT` —— 子地圖的特殊格清單（`docs/re/77`）。

格式（由 `FUN_222f_1321` @ `0x17211` 反組譯得出）：

    偏移 0 起往後長：3-byte 記錄 (X, Y, attr)，X == 0 表示表尾
    偏移 510 起往前長：2-byte 座標對，供類別 4（傳送）使用
    attr = (類別 << 5) | 值

本檔的重點是最後那條**可被資料打死的預測**：類別 4 的筆數必須等於
檔尾非零座標對的個數（因為第 k 筆類別 4 用第 k 對，k 從檔尾往前數）。
五個檔案全對就不是巧合 —— 這比「反組譯讀起來合理」強一個等級。

用法：
    python3 tools/parse_nssdat.py                # 跑全部五個檔案
    python3 tools/parse_nssdat.py 3SS.DAT        # 只跑一個
"""

import sys
from pathlib import Path

DATA_DIR = Path("workplace/orig/demwin/DEM_DATA")
FILES = [f"{n}SS.DAT" for n in range(1, 6)]

# 檔尾反向座標對的起點。`0x1ff - (k*2 + 2)`，k 從 0 起。
TAIL_END = 0x1FF


def parse(data: bytes):
    """回傳 (前段 3-byte 記錄, 檔尾反向座標對)。"""
    records = []
    i = 0
    while i + 2 < len(data) and data[i] != 0:
        records.append((data[i], data[i + 1], data[i + 2]))
        i += 3

    # 檔尾往前讀 2-byte 對，遇到 (0,0) 就停 —— 中間那一大段填充零。
    pairs = []
    k = 0
    while True:
        off = TAIL_END - (k * 2 + 2)
        if off < i:  # 撞到前段就停，兩邊都往中間長
            break
        pair = (data[off], data[off + 1])
        if pair == (0, 0):
            break
        pairs.append(pair)
        k += 1
    return records, pairs


def report(name: str, data: bytes) -> bool:
    records, pairs = parse(data)
    classes = {}
    for _, _, attr in records:
        classes[attr >> 5] = classes.get(attr >> 5, 0) + 1

    cls4 = [r for r in records if r[2] >> 5 == 4]
    ok = len(cls4) == len(pairs)

    print(f"=== {name} ===")
    print(f"  {len(records)} 筆記錄   類別分布 {dict(sorted(classes.items()))}")
    print(f"  類別 4（傳送）{len(cls4)} 筆　檔尾非零座標對 {len(pairs)} 組"
          f"　{'✓ 相符' if ok else '✗ 不符'}")
    for k, (rec, dest) in enumerate(zip(cls4, pairs)):
        print(f"    [{k}] ({rec[0]:2},{rec[1]:2}) → ({dest[0]:2},{dest[1]:2})")
    return ok


def main():
    names = sys.argv[1:] or FILES
    allok = True
    for name in names:
        path = DATA_DIR / name
        if not path.exists():
            print(f"!! 找不到 {path}（原版資料不進版控，見 CLAUDE.md 的 workplace 慣例）")
            allok = False
            continue
        allok &= report(name, path.read_bytes())
    print()
    print("結論：類別 4 筆數 ＝ 檔尾座標對數，"
          + ("全部相符。" if allok else "**有檔案不相符 —— 格式判讀要重看**。"))
    return 0 if allok else 1


sys.exit(main())
