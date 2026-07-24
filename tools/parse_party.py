#!/usr/bin/env python3
"""解析 Demon's Winter 的 PARTY.DAT / PARTY.BAK（隊伍存檔，固定長度二進位格式）。

完整欄位表與驗證狀態見 docs/formats/game-data-tables.md 的「PARTY.DAT 完整欄位位移表」。
本檔只實作已驗證或高信心假設的欄位；未解欄位以 raw hex 附上，供後續比對。

record 結構（每名角色 0x104 = 260 bytes，共 5 名角色，之後接 194 bytes 隊伍共用資料）：
    0x000            姓名（NUL 結尾字串，欄位保留 12 bytes）
    0x00c            種族索引（1 byte；0xFF = 尚未設定）
    0x00d - 0x019    未知（13 bytes，本檔 5 名角色全為 0）
    0x01a - 0x0c3    裝備／道具欄位區（10 個 slot，每 slot 17 bytes；欄位內部語意未解，
                     只驗證了 slot 邊界與「空 slot」規律：slot 內第 4 byte 常為 0xFF
                     代表空 slot；有裝備的 slot 開頭常見 0x0a 0x01，語意未定）
    0x0c4            經驗值（3 bytes，little-endian／攻略稱「反序」）
    0x0e8 - 0x0ff    屬性區（見 ATTR_OFFSETS，相對於經驗值欄位 0x0c4 的位移沿用攻略寫法）
    0x100 - 0x103    未知（4 bytes）

隊伍共用資料（record 5 之後，abs 0x514 起，194 bytes）：
    0x51e (trailer 相對 0x0a)   隊伍金幣，3 或 4 bytes little-endian（長度未能從現有存檔消歧，
                                 兩份存檔尾端都剛好是 0，見文件）
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
RACE_OFFSET = 0x0c
EXP_OFFSET = 0xc4  # 相對於 record 起始
INVENTORY_START = 0x1a
INVENTORY_SLOT_LEN = 17
INVENTORY_SLOT_COUNT = 10

TRAILER_START = NUM_CHARACTERS * RECORD_LEN  # 0x514
GOLD_TRAILER_OFFSET = 0x0a  # abs 0x51e

# 相對於 EXP 欄位（0xc4）的屬性位移，沿用 docs/walkthrough/part-6.md 已知/反推的寫法。
# 型別皆為 1 byte，除了 EXP 本身是 3 bytes。
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

# 種族索引 -> 名稱：採 translations/glossary.md 第 3 節（原版手冊順序）的假設映射。
# 驗證狀態：本檔案 5 名角色只出現索引 0/1/2，用「屬性上限刪去法」
# （docs/walkthrough/part-2.md 種族屬性上限表）排除了索引 2 不可能是黑暗精靈或巨魔或精靈
# （Norman 的天生速度/耐力/力量超過那三個種族的上限），縮小到人類或矮人，
# 再依此處假設的序列順序（人類=0, 精靈=1, 矮人=2, 黑暗精靈=3, 巨魔=4）採用矮人。
# 索引 3、4（黑暗精靈、巨魔）在這份存檔完全沒出現，尚無法驗證。
RACE_NAMES = {
    0xFF: "（未設定）",
    0: "人類（假設）",
    1: "精靈（假設，弱驗證）",
    2: "矮人（假設，用屬性上限刪去法部分驗證）",
    3: "黑暗精靈（假設，本存檔未出現，無法驗證）",
    4: "巨魔（假設，本存檔未出現，無法驗證）",
}


def le_bytes_to_int(b: bytes) -> int:
    return int.from_bytes(b, byteorder="little", signed=False)


def parse_character(rec: bytes) -> dict:
    name = rec[0:NAME_FIELD_LEN].split(b"\x00")[0].decode("latin1")
    race_byte = rec[RACE_OFFSET]
    exp = le_bytes_to_int(rec[EXP_OFFSET : EXP_OFFSET + 3])

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
        "race_byte": race_byte,
        "race_guess": RACE_NAMES.get(race_byte, f"未知索引({race_byte})"),
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
    if len(trailer) >= GOLD_TRAILER_OFFSET + 3:
        gold = le_bytes_to_int(trailer[GOLD_TRAILER_OFFSET : GOLD_TRAILER_OFFSET + 3])

    return {
        "characters": characters,
        "party_gold_3byte_guess": gold,
        "trailer_raw": trailer.hex(" "),
        "file_len": len(data),
    }


def print_report(result: dict):
    print(f"檔案長度: {result['file_len']} bytes\n")
    for i, ch in enumerate(result["characters"], 1):
        print(f"=== {i} 號角色: {ch['name']} ===")
        print(f"  種族 byte = {ch['race_byte']:#04x} -> {ch['race_guess']}")
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
        print("  裝備欄（10 slot，raw hex，欄位語意未解）：")
        for i2, slot in enumerate(ch["inventory_slots_raw"]):
            print(f"    slot{i2}: {slot}")
        print()

    print(f"隊伍金幣（3-byte little-endian 假設值） = {result['party_gold_3byte_guess']}")
    print("（金幣欄位實際長度未能從單一存檔消歧，見文件說明；此處僅取 3 bytes 供參考）")


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
