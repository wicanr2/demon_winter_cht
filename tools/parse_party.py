#!/usr/bin/env python3
"""解析 Demon's Winter 的 PARTY.DAT / PARTY.BAK（隊伍存檔，固定長度二進位格式）。

完整欄位表與驗證狀態見 docs/formats/game-data-tables.md 的「PARTY.DAT 完整欄位位移表」。
本檔只實作已驗證或高信心假設的欄位；未解欄位以 raw hex 附上，供後續比對。

record 結構（每名角色 0x104 = 260 bytes，共 5 名角色，之後接 194 bytes 隊伍共用資料）：
    0x000            姓名（NUL 結尾字串，欄位保留 12 bytes）
    0x00c - 0x0b5    裝備／道具欄位區（10 個 slot，每 slot 17 bytes；第 1 byte
                     是 ITEMS.DAT 型別，0xFF 代表空 slot）
    0x0c4            經驗值（4 bytes little-endian；遊戲規則另封頂 0x00FFFFFF）
    0x0e8 - 0x0ff    屬性區（見 ATTR_OFFSETS，相對於經驗值欄位 0x0c4 的位移沿用攻略寫法）
    0x0f4            等級
    0x0f5            種族索引
    0x0f6            職業索引
    0x100 / 0x101    已裝備武器／護甲的 slot 索引（0xFF = 無）

隊伍共用資料（record 5 之後，abs 0x514 起，194 bytes）：
    0x51e (trailer 相對 0x0a)   隊伍金幣，4 bytes little-endian
    其餘 trailer 欄位大多未解，dump_trailer() 會印出 raw bytes 供比對。

用法：
    python3 tools/parse_party.py [PARTY.DAT路徑]
    python3 tools/parse_party.py --diff PARTY.DAT PARTY.BAK   # 逐 byte диff 兩個存檔
"""
import sys
from pathlib import Path

DEFAULT_PATH = "workplace/orig/demwin/DEM_DATA/PARTY.DAT"

RECORD_LEN = 0x104
NUM_CHARACTERS = 5
NAME_FIELD_LEN = 12
LEVEL_OFFSET = 0xf4
RACE_OFFSET = 0xf5
CLASS_OFFSET = 0xf6
WEAPON_SLOT_OFFSET = 0x100
ARMOR_SLOT_OFFSET = 0x101
EXP_OFFSET = 0xc4  # 相對於 record 起始
INVENTORY_START = 0x0c
INVENTORY_SLOT_LEN = 17
INVENTORY_SLOT_COUNT = 10

TRAILER_START = NUM_CHARACTERS * RECORD_LEN  # 0x514
GOLD_TRAILER_OFFSET = 0x0a  # abs 0x51e
EXP_LEN = 4

# 相對於 EXP 欄位（0xc4）的屬性位移，沿用 docs/walkthrough/part-6.md 已知/反推的寫法。
# 型別皆為 1 byte，除了 EXP 本身是 4 bytes。
ATTR_OFFSETS = {
    "strength_natural": 0x24,
    "skill_natural": 0x25,
    "max_sp_natural": 0x26,
    "speed_natural": 0x2f,  # 攻略在 3 號角色誤標為「最大法力值」，已驗證應為速度，見文件
    "speed_bonus": 0x33,
    "strength_bonus": 0x34,
    "intellect": 0x35,
    "endurance": 0x36,
    "skill_bonus": 0x37,
    "max_hp": 0x38,
    "current_hp": 0x39,
    "max_sp_bonus": 0x3a,
    "current_sp": 0x3b,
}

# 種族索引已由 FILES.DAT 0x422 的 5×5 屬性上限表與手冊附錄 B 交叉驗證。
RACE_NAMES = {
    0xFF: "（未設定）",
    0: "人類",
    1: "精靈",
    2: "矮人",
    3: "黑暗精靈",
    4: "巨魔",
}


def le_bytes_to_int(b: bytes) -> int:
    return int.from_bytes(b, byteorder="little", signed=False)


def parse_character(rec: bytes) -> dict:
    name = rec[0:NAME_FIELD_LEN].split(b"\x00")[0].decode("latin1")
    level = rec[LEVEL_OFFSET]
    race_byte = rec[RACE_OFFSET]
    class_byte = rec[CLASS_OFFSET]
    exp = le_bytes_to_int(rec[EXP_OFFSET : EXP_OFFSET + EXP_LEN])

    attrs = {}
    for field, off in ATTR_OFFSETS.items():
        attrs[field] = rec[EXP_OFFSET + off]

    inventory_slots = []
    for i in range(INVENTORY_SLOT_COUNT):
        s = INVENTORY_START + i * INVENTORY_SLOT_LEN
        slot = rec[s : s + INVENTORY_SLOT_LEN]
        inventory_slots.append(slot.hex(" "))

    return {
        "name": name,
        "level": level,
        "race_byte": race_byte,
        "race_guess": RACE_NAMES.get(race_byte, f"未知索引({race_byte})"),
        "class_byte": class_byte,
        "equipped_weapon": rec[WEAPON_SLOT_OFFSET],
        "equipped_armor": rec[ARMOR_SLOT_OFFSET],
        "experience": exp,
        **attrs,
        "inventory_slots_raw": inventory_slots,
    }


def parse_party_dat(path) -> dict:
    data = Path(path).read_bytes()
    if len(data) < TRAILER_START:
        raise ValueError(f"檔案長度 {len(data)} 小於預期的 5 個角色記錄長度")

    characters = []
    for c in range(NUM_CHARACTERS):
        rec = data[c * RECORD_LEN : (c + 1) * RECORD_LEN]
        characters.append(parse_character(rec))

    trailer = data[TRAILER_START:]
    gold = None
    if len(trailer) >= GOLD_TRAILER_OFFSET + 4:
        gold = le_bytes_to_int(trailer[GOLD_TRAILER_OFFSET : GOLD_TRAILER_OFFSET + 4])

    return {
        "characters": characters,
        "party_gold": gold,
        "trailer_raw": trailer.hex(" "),
        "file_len": len(data),
    }


def print_report(result: dict):
    print(f"檔案長度: {result['file_len']} bytes\n")
    for i, ch in enumerate(result["characters"], 1):
        print(f"=== {i} 號角色: {ch['name']} ===")
        print(
            f"  等級 = {ch['level']}  種族 byte = {ch['race_byte']:#04x} -> {ch['race_guess']}"
            f"  職業 byte = {ch['class_byte']}"
        )
        print(
            f"  裝備索引：武器 {ch['equipped_weapon']}／護甲 {ch['equipped_armor']}"
        )
        print(f"  經驗值 = {ch['experience']}")
        print(
            "  力量(天生/含加成) = {}/{}   技巧(天生/含加成) = {}/{}   "
            "速度(天生/含加成) = {}/{}".format(
                ch["strength_natural"], ch["strength_bonus"],
                ch["skill_natural"], ch["skill_bonus"],
                ch["speed_natural"], ch["speed_bonus"],
            )
        )
        print(
            "  智力 = {}   耐力 = {}   最大法力(天生/含加成) = {}/{}".format(
                ch["intellect"], ch["endurance"],
                ch["max_sp_natural"], ch["max_sp_bonus"],
            )
        )
        print(
            "  生命值(目前/上限) = {}/{}   法力值(目前) = {}".format(
                ch["current_hp"], ch["max_hp"], ch["current_sp"]
            )
        )
        print("  道具欄（10 slot，raw hex；第 1 byte 是型別）：")
        for i2, slot in enumerate(ch["inventory_slots_raw"]):
            print(f"    slot{i2}: {slot}")
        print()

    print(f"隊伍金幣（4-byte little-endian） = {result['party_gold']}")


def diff_files(path_a, path_b):
    a = Path(path_a).read_bytes()
    b = Path(path_b).read_bytes()
    n = min(len(a), len(b))
    print(f"{path_a}: {len(a)} bytes, {path_b}: {len(b)} bytes")
    diffs = 0
    for i in range(n):
        if a[i] != b[i]:
            diffs += 1
            rec = i // RECORD_LEN if i < TRAILER_START else None
            rel = i - rec * RECORD_LEN if rec is not None else i - TRAILER_START
            where = f"char{rec+1} rel=0x{rel:03x}" if rec is not None else f"trailer rel=0x{rel:03x}"
            print(f"  abs 0x{i:04x} ({where}): {path_a}={a[i]:3d}(0x{a[i]:02x})  {path_b}={b[i]:3d}(0x{b[i]:02x})")
    print(f"共 {diffs} bytes 不同")


def main():
    args = sys.argv[1:]
    if args and args[0] == "--diff":
        diff_files(args[1], args[2])
        return
    path = args[0] if args else DEFAULT_PATH
    result = parse_party_dat(path)
    print_report(result)


if __name__ == "__main__":
    main()
