#!/usr/bin/env python3
"""解析 Demon's Winter 的 ITEMS.DAT。

格式（已驗證，見 docs/formats/game-data-tables.md）：
    與 MONSTER.DAT 同款：整檔是 ASCII 文字，NUL（0x00）分隔，不是固定長度二進位記錄。
    每個道具 = 1 個名字字串 + 7 個數字欄位，共 8 個 token。
    724 bytes 切成 240 個 token = 30 個道具 × 8。

前 13 個道具的順序與 translations/glossary.md 第 8 節（武器 8 種 + 護甲 5 種）的原版遊戲內順序
完全一致，可視為強驗證錨點：
    0 dagger, 1 small axe, 2 short sword, 3 mace, 4 morning star, 5 broad sword,
    6 battle axe, 7 2-hand sword,
    8 cloth, 9 leather, 10 chain, 11 scale, 12 plate

其餘 17 個是任務／特殊道具（crown、vial×2、ring、wand、staff、rod、gem、amulet、
medallion、figurine、talisman、salve、torch、lantern、Demon Crystal、Orb/Evertime）。

7 個數字欄位的語意（驗證狀態）：
    0 price        售價（金幣）        已驗證：武器/護甲售價順序與 walkthrough part-2/part-5
                                        的裝備價格量級一致（dagger 最便宜，2-hand sword 最貴，
                                        plate 護甲最貴）
    1 weapon_slot  是否佔用「武器手」欄位（1/0）   假設：武器與魔杖/法杖/權杖/戒指等飾品類為 1，
                                        護甲與純道具（寶石/火把/任務物品）為 0
    2 category     道具分類索引            假設：武器+護甲+冠冕+項鍊同為 3，
                                        法杖類道具為 1，藥水為 2，飾品/任務物品為 0
                                        （分類邊界不完全乾淨，需要動態驗證）
    3-6 未定欄位   同一大類（8 把武器 / 5 件護甲）內數值完全相同，
                                        推測是圖像/圖示參照或其他共用參數，非個別武器的
                                        傷害骰／力量需求（那些數值查無其一一對應，
                                        很可能是寫死在 DEMON.EXE 裡，不在本檔）

用法：
    python3 tools/parse_items.py [ITEMS.DAT路徑]
"""
import sys
import json
from pathlib import Path

DEFAULT_PATH = "workplace/orig/demwin/DEM_DATA/ITEMS.DAT"

FIELD_NAMES = ["price", "weapon_slot", "category", "f3", "f4", "f5", "f6"]

WEAPON_NAMES = {
    "dagger", "small axe", "short sword", "mace", "morning star",
    "broad sword", "battle axe", "2-hand sword",
}
ARMOR_NAMES = {"cloth", "leather", "chain", "scale", "plate"}


def parse_items_dat(path):
    data = Path(path).read_bytes()
    tokens = data.split(b"\x00")
    if tokens and tokens[-1] == b"":
        tokens = tokens[:-1]
    if len(tokens) % 8 != 0:
        raise ValueError(
            f"token 數 {len(tokens)} 不是 8 的倍數，格式假設可能不適用於這個檔案"
        )
    items = []
    for i in range(len(tokens) // 8):
        chunk = tokens[i * 8 : (i + 1) * 8]
        name = chunk[0].decode("latin1")
        nums = [int(x) for x in chunk[1:]]
        category = (
            "weapon" if name in WEAPON_NAMES
            else "armor" if name in ARMOR_NAMES
            else "misc"
        )
        items.append((name, category, dict(zip(FIELD_NAMES, nums))))
    return items


def main():
    path = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_PATH
    items = parse_items_dat(path)
    print(f"# ITEMS.DAT 共 {len(items)} 個道具\n")
    header = f"{'名稱':<20}{'分類':<8}" + "".join(f"{f:>10}" for f in FIELD_NAMES)
    print(header)
    print("-" * len(header))
    for name, cat, fields in items:
        row = f"{name:<20}{cat:<8}" + "".join(f"{fields[f]:>10}" for f in FIELD_NAMES)
        print(row)

    if "--json" in sys.argv:
        out = [{"name": n, "category": c, **f} for n, c, f in items]
        print(json.dumps(out, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
