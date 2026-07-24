#!/usr/bin/env python3
"""
Demon's Winter (SSI, 1988) -- DEM_DATA/*.TXT 事件腳本解析器。

背景與已驗證結論見 docs/formats/event-script.md，本檔只放「可重跑的解析邏輯」。

已驗證的檔案格式：
  - 分隔符是 NUL (0x00)，不是換行。
  - 欄位是 ASCII 十進位文字（"255" = 三個字元 '2' '5' '5'），不是二進位數字。
  - DATA1~5.TXT 是「事件 = 敘述文字 + 數字欄位」的結構化紀錄；
    TOWN.TXT / T.TXT / T2C.TXT / T2D.TXT / EREGORE.TXT / WIN.TXT 是純文字索引表（無數字欄位）。
  - DATA*.TXT 的數字欄位目前最強的驗證手段，是把疑似「怪物 ID」的數字對回 MONSTER.DAT
    的怪物名稱順序（0-based），再核對敘述文字是否語意相符（例如 "hill giant" ⇄ index 10）。

用法：
    # 純粹把檔案 tokenize 成欄位序列，標好型別（不做任何結構假設，最保守、最不會出錯）
    python3 tools/parse_datatxt.py dump DEM_DATA/DATA1.TXT

    # 針對 DATA*.TXT 套用已驗證的「事件記錄」結構做分解（見文件的「已驗證結構」章節）
    python3 tools/parse_datatxt.py records DEM_DATA/DATA1.TXT --monster-dat DEM_DATA/MONSTER.DAT

    # 輸出成 JSON（--records 模式時，附帶怪物名稱查表；--dump 模式時，只有 token 序列）
    python3 tools/parse_datatxt.py records DEM_DATA/DATA1.TXT --monster-dat DEM_DATA/MONSTER.DAT --json > data1.json

只用標準庫，不需要 pip install。
"""
from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass, field, asdict
from pathlib import Path


# ---------------------------------------------------------------------------
# 底層 tokenizer：NUL 分隔、標型別
# ---------------------------------------------------------------------------

def is_ascii_decimal(b: bytes) -> bool:
    """欄位是否為純 ASCII 十進位整數（含負號）。空字串不算。

    用 int() 判定而非 str.isdigit()：MONSTER.DAT 裡實際存在帶尾隨空白的欄位
    （例如 Swarm 的最後一個 stat 是 "0 "，多一個空格），isdigit() 會把它誤判成文字，
    導致整份表從那筆之後 index 全部錯位一格。int() 會自動 strip 空白，行為與此處
    「這欄位是不是數字」的語意一致，已用 MONSTER.DAT 的怪物名稱/ID 對照驗證過。
    """
    if not b:
        return False
    try:
        int(b.decode("latin1"))
        return True
    except ValueError:
        return False


@dataclass
class Token:
    index: int          # 第幾個欄位（0-based）
    offset: int          # 在檔案中的起始 byte offset
    kind: str            # "N"（純數字）或 "T"（文字）
    raw: bytes
    text: str = field(init=False)
    value: int | None = field(init=False)

    def __post_init__(self):
        self.text = self.raw.decode("latin1")
        self.value = int(self.text) if self.kind == "N" else None


def tokenize(data: bytes) -> list[Token]:
    """依 NUL 切欄位，標好每個欄位的型別與檔案內 offset。
    檔案通常以 NUL 結尾（trailing empty field），該空欄位會被捨棄。
    """
    tokens: list[Token] = []
    offset = 0
    parts = data.split(b"\x00")
    # 捨棄最後一個「檔尾 NUL 造成的空欄位」
    if parts and parts[-1] == b"":
        parts = parts[:-1]
    for i, p in enumerate(parts):
        kind = "N" if is_ascii_decimal(p) else "T"
        tokens.append(Token(index=i, offset=offset, kind=kind, raw=p))
        offset += len(p) + 1  # +1 = 這個欄位後面吃掉的 NUL
    return tokens


# ---------------------------------------------------------------------------
# 已驗證的 DATA*.TXT 事件記錄結構
#
#   file := header_numbers* record*
#   record := TEXT number_list
#   number_list := [] | [255] | [count, id_1..id_count, 255?, trailer...]
#
# 細節與各欄位的信心程度見 docs/formats/event-script.md。這裡的切法保守：
# 不強行把每個 record 塞進單一 grammar，只在能明確辨識時標出 count / monster_ids /
# terminator / trailer，辨識不出來的一律原樣保留在 raw_numbers。
# ---------------------------------------------------------------------------

SENTINEL = 255  # 已驗證：欄位裡最常見的「無/終止」哨兵值，見文件


@dataclass
class EventRecord:
    index: int
    text: str
    raw_numbers: list[int]
    # 以下是「count-prefixed 怪物清單」假說套用後的分解結果；套不上就都是 None。
    count: int | None = None
    monster_ids: list[int] | None = None
    monster_names: list[str] | None = None
    terminator_present: bool = False
    trailer: list[int] = field(default_factory=list)
    control_prefix: str | None = None  # 文字開頭若是控制字元('%'、數字)則記在這裡


def load_monster_names(monster_dat: Path) -> list[str]:
    """MONSTER.DAT 是同構的 NUL 分隔表：name, stat_1..stat_11, name, stat_1..stat_11, ...
    回傳依檔案順序（= 0-based 怪物 ID）排列的名字清單。
    """
    data = monster_dat.read_bytes()
    tokens = tokenize(data)
    names = []
    i = 0
    while i < len(tokens):
        assert tokens[i].kind == "T", f"MONSTER.DAT 預期第 {i} 欄是名字，卻是數字：{tokens[i].raw}"
        names.append(tokens[i].text)
        i += 1
        while i < len(tokens) and tokens[i].kind == "N":
            i += 1
    return names


def parse_records(data: bytes, monster_names: list[str] | None = None) -> tuple[list[int], list[EventRecord]]:
    tokens = tokenize(data)
    idx = 0
    header: list[int] = []
    while idx < len(tokens) and tokens[idx].kind == "N":
        header.append(tokens[idx].value)
        idx += 1

    records: list[EventRecord] = []
    rec_i = 0
    while idx < len(tokens):
        tok = tokens[idx]
        if tok.kind != "T":
            # 不應該發生（tokenizer 保證 T/N 交替於 record 邊界），保留原始資料以防萬一
            idx += 1
            continue
        text = tok.text
        idx += 1
        nums: list[int] = []
        while idx < len(tokens) and tokens[idx].kind == "N":
            nums.append(tokens[idx].value)
            idx += 1

        rec = EventRecord(index=rec_i, text=text, raw_numbers=list(nums))
        if text and (text[0] == "%" or text[0].isdigit()):
            rec.control_prefix = text[0]

        # 套「count 前綴的怪物清單」假說：n = [count, id*count, 255?, trailer...]
        if nums:
            count = nums[0]
            if 0 <= count <= 20 and len(nums) >= 1 + count:
                ids = nums[1 : 1 + count]
                rest = nums[1 + count :]
                rec.count = count
                rec.monster_ids = ids
                if monster_names is not None:
                    rec.monster_names = [
                        monster_names[i] if 0 <= i < len(monster_names) else f"?id{i}"
                        for i in ids
                    ]
                if rest and rest[0] == SENTINEL:
                    rec.terminator_present = True
                    rec.trailer = rest[1:]
                else:
                    rec.trailer = rest
        records.append(rec)
        rec_i += 1

    return header, records


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def cmd_dump(args):
    data = Path(args.file).read_bytes()
    tokens = tokenize(data)
    if args.json:
        out = [
            {"index": t.index, "offset": t.offset, "kind": t.kind, "value": t.value, "text": t.text}
            for t in tokens
        ]
        print(json.dumps(out, ensure_ascii=False, indent=2))
        return
    for t in tokens:
        if t.kind == "N":
            print(f"[{t.index:4d}] @0x{t.offset:05x} N {t.value}")
        else:
            disp = t.text if len(t.text) <= 90 else t.text[:87] + "..."
            print(f"[{t.index:4d}] @0x{t.offset:05x} T {disp!r}")


def cmd_records(args):
    data = Path(args.file).read_bytes()
    monster_names = None
    if args.monster_dat:
        monster_names = load_monster_names(Path(args.monster_dat))
    header, records = parse_records(data, monster_names)

    if args.json:
        out = {
            "file": str(args.file),
            "header": header,
            "records": [asdict(r) for r in records],
        }
        print(json.dumps(out, ensure_ascii=False, indent=2))
        return

    print(f"# {args.file}")
    print(f"header (檔頭、緊接第一段文字前的數字欄位) = {header}")
    print()
    for r in records:
        text_disp = r.text if len(r.text) <= 70 else r.text[:67] + "..."
        prefix = f"[ctrl={r.control_prefix!r}]" if r.control_prefix else ""
        print(f"#{r.index:3d} {prefix} {text_disp!r}")
        if r.count is not None:
            names = f" -> {r.monster_names}" if r.monster_names else ""
            print(
                f"      count={r.count} monster_ids={r.monster_ids}{names} "
                f"terminator={r.terminator_present} trailer={r.trailer}"
            )
        else:
            print(f"      raw_numbers={r.raw_numbers}  (套不上 count-prefix 假說，原樣保留)")
    print()
    print(f"# 共 {len(records)} 筆記錄")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)

    p_dump = sub.add_parser("dump", help="原始 tokenize：列出每個欄位的型別/offset/內容，不做結構假設")
    p_dump.add_argument("file", help="要解析的 .TXT 檔路徑")
    p_dump.add_argument("--json", action="store_true", help="輸出 JSON 而非人類可讀文字")
    p_dump.set_defaults(func=cmd_dump)

    p_rec = sub.add_parser("records", help="套用已驗證的 DATA*.TXT 事件記錄結構做分解")
    p_rec.add_argument("file", help="要解析的 DATA*.TXT 檔路徑")
    p_rec.add_argument("--monster-dat", help="MONSTER.DAT 路徑，提供時會把怪物 ID 換成名字")
    p_rec.add_argument("--json", action="store_true", help="輸出 JSON 而非人類可讀文字")
    p_rec.set_defaults(func=cmd_records)

    args = ap.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
