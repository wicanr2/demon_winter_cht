#!/usr/bin/env python3
"""DEMON.INT 的位址換算與交叉引用掃描。

這支工具做的是這個專案反組譯時**重複最多次的三個動作**：

1. `seg:off` ↔ 檔案位移的換算（每次手算都可能算錯）
2. 「誰呼叫這支函式」「誰指到這個位址」「誰讀寫這個全域」
3. 找 Turbo C 的 switch 跳表並把它解出來

第 3 項是 `docs/re/50` §1.1 與 `docs/re/52` §2 的關鍵手法：
**Turbo C 把 switch 的分派放在函式最後面**，從 case body 往前找永遠找不到；
掃 `jmp word ptr cs:[bx+d16]` 的指令碼 `2e ff a7` 一次命中。

# 位址模型

    檔案位移 = 段 × 16 − 0xC400

而**指令與遠指標裡存的段值要 +0x1000 才是真的段**（連結期原始值，
載入時才加基底）。所以組合語言裡看到的 `call 0x178d:0x1fd1`
實際上是 `278d:1fd1`。見 `docs/re/08` §1.2。

# 用法

    tools/xref.py addr 278d:1fd1          # 換算成檔案位移
    tools/xref.py addr 0x1d4a1 --seg 278d # 反過來算
    tools/xref.py call 278d:1fd1          # 誰 far call 這支
    tools/xref.py ptr 278d:1fd1           # 誰的遠指標指到這裡
    tools/xref.py global 0x4e2e           # 誰讀寫這個 DS 全域
    tools/xref.py dis 278d:1fd1 -n 0x40   # 反組譯（自動換算）
    tools/xref.py str 0x38c9              # 讀 DS 上的字串
    tools/xref.py table 0x16488           # 解 switch 跳表
    tools/xref.py findtables 0x15ef0 0x17000   # 掃這個範圍裡的跳表
"""

import argparse
import re
import struct
import subprocess
import sys
import tempfile
from pathlib import Path

# 預設的執行檔。原版資料不進版控，路徑照 CLAUDE.md 的 workplace 慣例。
DEFAULT_BINARY = Path("workplace/orig/demwin/DEMON.INT")

# 檔案位移 = 段 × 16 − SEGMENT_BIAS。
SEGMENT_BIAS = 0xC400
# 指令／遠指標裡存的段值 + ENCODED_BIAS 才是真的段。
ENCODED_BIAS = 0x1000
# DS 基底（檔案位移）。全域 `ds:0xNNNN` 的資料在這裡 + 偏移。
DS_BASE = 0x25B00


def seg_to_file(seg: int, off: int = 0) -> int:
    """段:偏移 → 檔案位移。"""
    return seg * 16 - SEGMENT_BIAS + off


def file_to_off(file_off: int, seg: int) -> int:
    """檔案位移 → 指定段裡的偏移。"""
    return file_off - seg_to_file(seg)


def parse_addr(text: str) -> tuple[int, int | None]:
    """吃 `278d:1fd1` 或 `0x1d4a1`，回傳 (檔案位移, 段或 None)。"""
    if ":" in text:
        seg_s, off_s = text.split(":", 1)
        seg, off = int(seg_s, 16), int(off_s, 16)
        return seg_to_file(seg, off), seg
    return int(text, 0), None


def load(path: Path) -> bytes:
    if not path.exists():
        sys.exit(f"找不到 {path}（原版執行檔不進版控，見 CLAUDE.md）")
    return path.read_bytes()


def objdump(data: bytes, file_off: int, length: int) -> str:
    """把一段位元組當 16-bit x86 反組譯，位址標成檔案位移。

    與 `docs/re/08` §1.2 同一套參數：`--adjust-vma` 直接設成檔案位移，
    印出來的位址就是本專案慣用的座標。

    **一定要落地成檔案** —— objdump 讀 `/dev/stdin` 會失敗（它要 seek）。
    """
    chunk = data[file_off:file_off + length]
    with tempfile.NamedTemporaryFile(suffix=".bin") as tmp:
        tmp.write(chunk)
        tmp.flush()
        proc = subprocess.run(
            ["objdump", "-D", "-b", "binary", "-m", "i386",
             "-Maddr16,data16,intel", f"--adjust-vma={file_off}", tmp.name],
            capture_output=True, check=True)
    out = proc.stdout.decode("utf-8", "replace")
    # 前七行是 objdump 的檔頭，沒有資訊。
    return "\n".join(out.splitlines()[7:])


# --- 各子命令 ---

def cmd_addr(args, data):
    file_off, seg = parse_addr(args.addr)
    if seg is not None:
        print(f"{seg:04x}:{file_off - seg_to_file(seg):04x}  →  檔案位移 0x{file_off:05x}")
        return
    if args.seg is None:
        sys.exit("給檔案位移時要用 --seg 指定段")
    s = int(args.seg, 16)
    print(f"檔案位移 0x{file_off:05x}  →  {s:04x}:{file_to_off(file_off, s):04x}"
          f"（組語裡寫成 {s - ENCODED_BIAS:04x}:{file_to_off(file_off, s):04x}）")


def cmd_call(args, data):
    """找 far call 到指定 seg:off 的地方。

    機器碼是 `9a <off16> <seg16>`，而 seg 存的是**編碼過的**值（真段 − 0x1000）。
    """
    file_off, seg = parse_addr(args.addr)
    if seg is None:
        sys.exit("call 要用 seg:off 的形式（far call 的運算元就是段與偏移）")
    off = file_to_off(file_off, seg)
    pat = bytes([0x9A]) + struct.pack("<HH", off, seg - ENCODED_BIAS)
    hits = [m.start() for m in re.finditer(re.escape(pat), data)]
    print(f"far call {seg:04x}:{off:04x}（機器碼 {pat.hex(' ')}）：{len(hits)} 處")
    for h in hits:
        print(f"  0x{h:05x}")
        if args.context:
            print(objdump(data, max(0, h - args.context), args.context + 8))


def cmd_ptr(args, data):
    """找指到某個位址的遠指標（4 bytes 的 `(off, seg)`）。

    段值同樣是編碼過的，所以判準是 `off + 16 × 存的段 == 線性位址 − 0x10000`。
    """
    file_off, _ = parse_addr(args.addr)
    target = file_off + SEGMENT_BIAS - (ENCODED_BIAS * 16)
    hits = []
    for i in range(len(data) - 3):
        off, seg = struct.unpack_from("<HH", data, i)
        if seg and off + 16 * seg == target:
            hits.append((i, off, seg))
    print(f"指到 0x{file_off:05x} 的遠指標：{len(hits)} 處")
    for i, off, seg in hits:
        print(f"  0x{i:05x}  →  {seg:04x}:{off:04x}"
              f"（真段 {seg + ENCODED_BIAS:04x}）")


# DS 全域的常見定址編碼。key 是前綴位元組，value 是給人看的說明。
GLOBAL_OPCODES = {
    b"\xa0": "mov al, ds:X",
    b"\xa1": "mov ax, ds:X",
    b"\xa2": "mov ds:X, al",
    b"\xa3": "mov ds:X, ax",
    b"\xc6\x06": "mov byte ds:X, imm",
    b"\xc7\x06": "mov word ds:X, imm",
    b"\x8b\x16": "mov dx, ds:X",
    b"\x8b\x1e": "mov bx, ds:X",
    b"\x8b\x0e": "mov cx, ds:X",
    b"\xff\x36": "push ds:X",
    b"\xff\x06": "inc word ds:X",
    b"\xff\x0e": "dec word ds:X",
    b"\x83\x3e": "cmp word ds:X, imm8",
    b"\x81\x3e": "cmp word ds:X, imm16",
    b"\x80\x3e": "cmp byte ds:X, imm8",
    b"\x01\x06": "add word ds:X, ax",
    b"\x11\x16": "adc word ds:X, dx",
    b"\x29\x06": "sub word ds:X, ax",
    b"\x19\x16": "sbb word ds:X, dx",
    b"\x83\x06": "add word ds:X, imm8",
    b"\x83\x2e": "sub word ds:X, imm8",
    b"\xc4\x1e": "les bx, ds:X",
    b"\xc5\x1e": "lds bx, ds:X",
}


def cmd_global(args, data):
    """找讀寫某個 DS 全域的地方。

    只認直接定址那幾種編碼（涵蓋這個專案實際遇到的所有情形）。
    `[bx+disp]` 那種靠基底暫存器的形式抓不到 —— 那類要靠 `--raw` 掃原始位移。
    """
    addr = int(args.offset, 0)
    disp = struct.pack("<H", addr)
    print(f"ds:0x{addr:04x}（資料在檔案位移 0x{DS_BASE + addr:05x}）")
    total = 0
    for prefix, desc in GLOBAL_OPCODES.items():
        pat = prefix + disp
        hits = [m.start() for m in re.finditer(re.escape(pat), data)]
        if hits:
            total += len(hits)
            print(f"  {desc:26s} {' '.join(f'0x{h:05x}' for h in hits)}")
    if args.raw:
        hits = [m.start() for m in re.finditer(re.escape(disp), data)]
        print(f"  {'（原始位移出現處）':26s} {len(hits)} 處")
    if total == 0:
        print("  沒有直接定址的引用 —— 可能是走 [bx+disp] 這類基底定址，試試 --raw")


def cmd_dis(args, data):
    file_off, _ = parse_addr(args.addr)
    print(objdump(data, file_off, args.n))


def cmd_str(args, data):
    """讀 DS 上的一條 NUL 結尾字串。

    這個專案裡**讀字串常常比讀組合語言快** —— `docs/re/52` 的選項表、
    `docs/re/53` 的 " LIE "、`docs/re/50` 的 "The sun is rising"
    都是字串自己招供的。
    """
    off = int(args.offset, 0)
    p = DS_BASE + off
    end = data.index(b"\x00", p)
    print(f"ds:0x{off:04x} = {data[p:end].decode('latin1')!r}")


def cmd_table(args, data):
    """解 Turbo C 的 switch 跳表。

    位址給 `jmp word ptr cs:[bx+d16]` 那一行（機器碼 `2e ff a7`）。
    往前找 `cmp ax, imm16` 取 case 數、找 `sub ax, imm16` 取起始值。
    """
    jmp_off, _ = parse_addr(args.addr)
    if data[jmp_off:jmp_off + 3] != b"\x2e\xff\xa7":
        sys.exit(f"0x{jmp_off:05x} 不是 `jmp word ptr cs:[bx+d16]`"
                 f"（機器碼應該是 2e ff a7，實際是 {data[jmp_off:jmp_off+3].hex(' ')}）")
    table_disp = struct.unpack_from("<H", data, jmp_off + 3)[0]

    # 往前 16 bytes 找 `cmp ax,imm16`（3d）與 `sub ax,imm16`（2d）。
    window = data[max(0, jmp_off - 16):jmp_off]
    count = base = None
    for i in range(len(window) - 2):
        if window[i] == 0x3D and count is None:
            count = struct.unpack_from("<H", window, i + 1)[0]
        if window[i] == 0x2D and base is None:
            base = struct.unpack_from("<H", window, i + 1)[0]
    if count is None:
        sys.exit("找不到 `cmp ax,imm16`，case 數量無法判定")
    base = base or 0

    if args.seg:
        seg = int(args.seg, 16)
    else:
        sys.exit("要用 --seg 指定這段程式的段（跳表偏移是 CS 相對的）")
    table = seg_to_file(seg, table_disp)

    print(f"跳表在 {seg:04x}:{table_disp:04x} = 檔案位移 0x{table:05x}")
    print(f"case 數 {count}，選擇子先減 {base}"
          f" → 原始值 {base}–{base + count - 1}")
    print()
    print("原始值  case  偏移    檔案位移   第一行")
    for i in range(count):
        w = struct.unpack_from("<H", data, table + i * 2)[0]
        body = seg_to_file(seg, w)
        first = objdump(data, body, 12).splitlines()
        text = ""
        for line in first:
            if "\t" in line:
                text = line.split("\t")[-1].strip()
                break
        print(f"  0x{base + i:02x}  {i:4d}  0x{w:04x}  0x{body:05x}   {text}")


def cmd_findtables(args, data):
    """掃一段範圍裡所有的 switch 跳表分派點。"""
    lo, _ = parse_addr(args.start)
    hi, _ = parse_addr(args.end)
    hits = [i for i in range(lo, min(hi, len(data) - 3))
            if data[i:i + 3] == b"\x2e\xff\xa7"]
    print(f"0x{lo:05x}–0x{hi:05x} 裡的 switch 分派：{len(hits)} 處")
    for h in hits:
        disp = struct.unpack_from("<H", data, h + 3)[0]
        print(f"  0x{h:05x}  跳表在 CS:0x{disp:04x}"
              f"（用 `table 0x{h:05x} --seg <段>` 解開）")


def main():
    ap = argparse.ArgumentParser(
        description="DEMON.INT 位址換算與交叉引用",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__)
    ap.add_argument("--binary", type=Path, default=DEFAULT_BINARY)
    sub = ap.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("addr", help="seg:off ↔ 檔案位移")
    p.add_argument("addr")
    p.add_argument("--seg", help="給檔案位移時，要換算到哪個段")
    p.set_defaults(fn=cmd_addr)

    p = sub.add_parser("call", help="誰 far call 這支函式")
    p.add_argument("addr")
    p.add_argument("-c", "--context", type=lambda s: int(s, 0), default=0,
                   help="順便反組譯呼叫點前面幾個 bytes")
    p.set_defaults(fn=cmd_call)

    p = sub.add_parser("ptr", help="誰的遠指標指到這裡")
    p.add_argument("addr")
    p.set_defaults(fn=cmd_ptr)

    p = sub.add_parser("global", help="誰讀寫這個 DS 全域")
    p.add_argument("offset")
    p.add_argument("--raw", action="store_true", help="連原始位移出現處都數")
    p.set_defaults(fn=cmd_global)

    p = sub.add_parser("dis", help="反組譯（自動換算位址）")
    p.add_argument("addr")
    p.add_argument("-n", type=lambda s: int(s, 0), default=0x40)
    p.set_defaults(fn=cmd_dis)

    p = sub.add_parser("str", help="讀 DS 上的字串")
    p.add_argument("offset")
    p.set_defaults(fn=cmd_str)

    p = sub.add_parser("table", help="解 switch 跳表")
    p.add_argument("addr", help="`jmp word ptr cs:[bx+d16]` 那一行的位址")
    p.add_argument("--seg", required=True, help="這段程式的段（十六進位）")
    p.set_defaults(fn=cmd_table)

    p = sub.add_parser("findtables", help="掃範圍裡的 switch 分派點")
    p.add_argument("start")
    p.add_argument("end")
    p.set_defaults(fn=cmd_findtables)

    args = ap.parse_args()
    args.fn(args, load(args.binary))


if __name__ == "__main__":
    main()
