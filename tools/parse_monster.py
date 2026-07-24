#!/usr/bin/env python3
"""解析 Demon's Winter 的 MONSTER.DAT。

格式（已驗證，見 docs/formats/game-data-tables.md）：
    整個檔案是純 ASCII 文字，用 0x00（NUL）當分隔符，不是固定長度的二進位記錄。
    每一隻怪物 = 1 個名字字串 + 11 個數字欄位（皆為 ASCII 十進位文字，含負號），
    以 NUL 分隔，共 12 個 token。3853 bytes 的檔案剛好切成 1188 個 token = 99 隻怪物 × 12。

11 個數字欄位的語意（驗證狀態見文件，這裡只標簡稱）：
    0 speed          速度                      已驗證（跨等級序列遞增、與 PC 屬性同義）
    1 strength       力量                      已驗證
    2 skill          技巧                      已驗證
    3 hp             生命值                    假設（數值隨怪物強度遞增，量級合理）
    4 attack_type    攻擊/武器類型索引（負值＝帶毒）  已驗證（與 DEMON.INT 武器類型字串表對照，
                                                 13=Bite；蛇類全部帶負號，對應「毒咬」）
    5 sprite_index   圖像索引（MONSTER.SHP/SHE）      已驗證（同種怪物群組數值完全一致，
                                                 如四種熊、三種狼、七種龍）
    6 num_attacks    每回合攻擊次數                假設（隨戰士等級序列遞增，量級合理）
    7 experience     擊殺經驗值                    已驗證（隨怪物強度單調遞增）
    8 level          怪物等級/難度分級（1–10）       假設（非等於名稱中的數字，是另一套內部分級）
    9 sp             法力值上限                    已驗證（隨「法師」序列遞增，與 PC 法力值同義）
    10 special       特殊能力/元素索引，-1＝無         假設（可能是咒語/吐息類型索引，未完全解出）

用法：
    python3 tools/parse_monster.py [MONSTER.DAT路徑]
"""
import sys
import json
from pathlib import Path

DEFAULT_PATH = "workplace/orig/demwin/DEM_DATA/MONSTER.DAT"

FIELD_NAMES = [
    "speed", "strength", "skill", "hp", "attack_type", "sprite_index",
    "num_attacks", "experience", "level", "sp", "special",
]


def parse_monster_dat(path):
    """回傳 [(name, {field: value, ...}), ...]。"""
    data = Path(path).read_bytes()
    tokens = data.split(b"\x00")
    if tokens and tokens[-1] == b"":
        tokens = tokens[:-1]
    if len(tokens) % 12 != 0:
        raise ValueError(
            f"token 數 {len(tokens)} 不是 12 的倍數，格式假設可能不適用於這個檔案"
        )
    monsters = []
    for i in range(len(tokens) // 12):
        chunk = tokens[i * 12 : (i + 1) * 12]
        name = chunk[0].decode("latin1")
        nums = [int(x) for x in chunk[1:]]
        monsters.append((name, dict(zip(FIELD_NAMES, nums))))
    return monsters


def main():
    path = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_PATH
    monsters = parse_monster_dat(path)
    print(f"# MONSTER.DAT 共 {len(monsters)} 隻怪物\n")
    header = f"{'名稱':<20}" + "".join(f"{f:>10}" for f in FIELD_NAMES)
    print(header)
    print("-" * len(header))
    for name, fields in monsters:
        row = f"{name:<20}" + "".join(f"{fields[f]:>10}" for f in FIELD_NAMES)
        print(row)

    if "--json" in sys.argv:
        out = [{"name": n, **f} for n, f in monsters]
        print(json.dumps(out, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
