#!/usr/bin/env python3
"""掃某個 disp16 欄位的所有存取端（byte-level，不靠線性反組譯對齊）。

用法： fieldscan.py 0xa9 [0xbe ...]

掃法：ModRM 的 mod=10（disp16）有八種 rm。指令碼只取「會碰記憶體的
單/雙位元組 mov / cmp / test / inc / dec / and / or」那一組，
再比對後兩個位元組是不是目標 disp16（little-endian）。

# ⚠ 限制：只看得到「位移是常數」的存取

這支工具最有用的地方是**普查**：「`+0xbe` 十一處存取沒有一處寫 0
→ 它是單向閂鎖」「`+0xd7`／`+0xd8` 各六處存取只有一處寫入
→ 全遊戲只有那裡會清掉薩滿與司祭」——這種結論比讀邏輯強。

**但它只掃常數位移。** 偏移放在暫存器的存取（`mov es:[bx+si], al`，
執行時才決定要寫哪個欄位）一個都掃不到 —— 光是隊伍結構就有 20 處
這種寫入端（`docs/re/80` §3）。

所以：
  * 「這個欄位被這樣用」——可以靠它
  * **「這個欄位沒有寫入端」——不可以只靠它**

`docs/re/80` 就踩到這一點：`+0xb9` 的 19 處存取沒有一處寫 1，
看起來整條劇情鏈到不了，但攻略明確說那段劇情會發生。
依 oracle 優先序（攻略 > 反組譯推論），結論是掃描器有盲點，不是劇情不存在。
"""
import sys
from pathlib import Path

BIN = Path("workplace/orig/demwin/DEMON.INT")
SEGMENT_BIAS = 0xC400

# opcode → (助憶, 有無立即值長度)
OPS = {
    0x88: ("mov r/m8,r8", 0),
    0x89: ("mov r/m16,r16", 0),
    0x8A: ("mov r8,r/m8", 0),
    0x8B: ("mov r16,r/m16", 0),
    0xC6: ("mov r/m8,imm8", 1),
    0xC7: ("mov r/m16,imm16", 2),
    0x38: ("cmp r/m8,r8", 0),
    0x39: ("cmp r/m16,r16", 0),
    0x3A: ("cmp r8,r/m8", 0),
    0x3B: ("cmp r16,r/m16", 0),
    0x84: ("test r/m8,r8", 0),
    0x85: ("test r/m16,r16", 0),
    0x00: ("add r/m8,r8", 0),
    0x01: ("add r/m16,r16", 0),
    0x02: ("add r8,r/m8", 0),
    0x03: ("add r16,r/m16", 0),
    0x28: ("sub r/m8,r8", 0),
    0x2A: ("sub r8,r/m8", 0),
    0x08: ("or r/m8,r8", 0),
    0x0A: ("or r8,r/m8", 0),
    0x20: ("and r/m8,r8", 0),
    0x22: ("and r8,r/m8", 0),
}
# group opcodes：reg 欄是子運算
GRP1 = {0x80: ("<grp1> r/m8,imm8", 1), 0x81: ("<grp1> r/m16,imm16", 2),
        0x83: ("<grp1> r/m16,imm8", 1)}
GRP1_OPS = ["add", "or", "adc", "sbb", "and", "sub", "xor", "cmp"]
GRP_FE = {0xFE: ("<inc/dec> r/m8", 0), 0xFF: ("<grp5> r/m16", 0)}
FE_OPS = ["inc", "dec"]
FF_OPS = ["inc", "dec", "call", "callf", "jmp", "jmpf", "push", "?"]

RM = ["[bx+si", "[bx+di", "[bp+si", "[bp+di", "[si", "[di", "[bp", "[bx"]
REG8 = ["al", "cl", "dl", "bl", "ah", "ch", "dh", "bh"]
REG16 = ["ax", "cx", "dx", "bx", "sp", "bp", "si", "di"]


def scan(data, disp):
    lo, hi = disp & 0xFF, (disp >> 8) & 0xFF
    out = []
    for i in range(len(data) - 5):
        op = data[i]
        table = None
        if op in OPS:
            table = OPS
        elif op in GRP1:
            table = GRP1
        elif op in GRP_FE:
            table = GRP_FE
        if table is None:
            continue
        modrm = data[i + 1]
        if modrm >> 6 != 0b10:
            continue
        if data[i + 2] != lo or data[i + 3] != hi:
            continue
        name, immlen = table[op]
        reg = (modrm >> 3) & 7
        rm = modrm & 7
        if op in GRP1:
            name = name.replace("<grp1>", GRP1_OPS[reg])
        elif op == 0xFE:
            name = name.replace("<inc/dec>", FE_OPS[reg] if reg < 2 else f"?{reg}")
        elif op == 0xFF:
            name = name.replace("<grp5>", FF_OPS[reg])
        imm = None
        if immlen == 1:
            imm = data[i + 4]
        elif immlen == 2:
            imm = data[i + 4] | (data[i + 5] << 8)
        # 組出可讀的運算元
        wide = op in (0x89, 0x8B, 0xC7, 0x39, 0x3B, 0x85, 0x01, 0x03, 0x81, 0x83, 0xFF)
        regname = (REG16 if wide else REG8)[reg]
        mem = f"{RM[rm]}+0x{disp:04x}]"
        out.append((i, name, mem, regname, imm))
    return out


def main():
    data = BIN.read_bytes()
    for arg in sys.argv[1:]:
        disp = int(arg, 0)
        hits = scan(data, disp)
        print(f"=== disp16 = 0x{disp:04x} —— {len(hits)} 處 ===")
        for off, name, mem, regname, imm in hits:
            immtxt = f"  imm=0x{imm:x}({imm})" if imm is not None else ""
            print(f"0x{off:05x}  {name:<22} {mem:<20} reg={regname}{immtxt}")


main()
