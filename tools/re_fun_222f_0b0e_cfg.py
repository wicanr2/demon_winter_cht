#!/usr/bin/env python3
"""
逐行分析 FUN_222f_0b0e（Demon's Winter 主指令迴圈）的輔助工具。

背景：這個函式的 decompiler 輸出膨脹到 6557 行、59 個 unreachable 警告，不可信。
必須改讀 workplace/ghidra/export/disassembly.asm 的原始反組譯。但這份反組譯裡
control-flow 指令（Jcc/JMP/CALLF）的目標位址，Ghidra 用的「顯示 segment」不一定
等於該位址「正規」所屬的 segment（例如函式內部一個近跳轉，目標可能顯示成
"0x2000:2e4a" 而不是 "222f:0b5a"）——這是 16-bit real mode segment:offset 定址法
的本質（同一個實體位址有無限多組 segment:offset 表示法），16-bit 編譯器/組譯器
在產生 CALLF/JMP 的立即數運算元時，用的是「當時已知的某個 segment 值」，不保證
跟 Ghidra 後續替目標位址取的函式名稱 segment 一致。

驗證過的解法（見 docs/re/08 §1）：一律先換算成 file_offset
（file_offset = segment*16 + offset - 0xC400，跟 docs/re/00 的公式完全一致，
因為 linear = segment*16+offset，只是這裡把它用在「目標位址」而非「函式進入點」），
再拿 file_offset 去比對 functions.csv 裡各函式的 [start, start+size) 區間，
或比對本函式自己指令清單的 file_offset，就能穩定地把任何顯示 segment 的位址
正確歸位。

本工具只做「輔助換算 + 列出 control flow 指令」，不做自動反編譯、不猜測語意——
語意判讀仍由人工逐行完成，寫進 docs/re/08-movement-and-modes.md。

用法：
    python3 tools/re_fun_222f_0b0e_cfg.py
（純標準庫，讀 workplace/ghidra/export/disassembly.asm 與 functions.csv，
   不需要 docker、不需要額外套件）
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EXPORT = ROOT / "workplace" / "ghidra" / "export"
DISASM = EXPORT / "disassembly.asm"
FUNCS = EXPORT / "functions.csv"

FILE_OFFSET_CONST = 0xC400

LINE_RE = re.compile(r"^([0-9a-fA-F]{4}):([0-9a-fA-F]{4})\t(.*)$")
SEGOFF_RE = re.compile(r"\b0x([0-9a-fA-F]{1,4}):([0-9a-fA-F]{1,4})\b")
# JMP 0x1234 或 JZ 0x1234 這種"隱式同 segment"的近跳轉（沒有 0x seg:off，只有純位移）
NEARJMP_RE = re.compile(r"^(J[A-Z]+|LOOP\w*)\s+0x([0-9a-fA-F]{1,4})$")

CTRLFLOW_MNEM = re.compile(
    r"^(JMP|JZ|JNZ|JE|JNE|JL|JLE|JG|JGE|JB|JBE|JA|JAE|JC|JNC|JO|JNO|JS|JNS|"
    r"JP|JNP|JPE|JPO|JCXZ|LOOP|LOOPZ|LOOPNZ|LOOPE|LOOPNE|CALLF|CALLN|CALL|"
    r"RETF|RET)\b"
)


def seg_off_to_file_offset(seg: int, off: int) -> int:
    return seg * 16 + off - FILE_OFFSET_CONST


def load_functions():
    funcs = []  # (file_offset_start, file_offset_end, name, seg, off, size)
    for line in FUNCS.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("address"):
            continue
        parts = line.split(",")
        if len(parts) < 4:
            continue
        addr, name, size_s, _thunk = parts[0], parts[1], parts[2], parts[3]
        m = re.match(r"^([0-9a-fA-F]{4}):([0-9a-fA-F]{4})$", addr)
        if not m:
            continue
        seg = int(m.group(1), 16)
        off = int(m.group(2), 16)
        try:
            size = int(size_s)
        except ValueError:
            continue
        fo_start = seg_off_to_file_offset(seg, off)
        funcs.append((fo_start, fo_start + size, name, seg, off, size))
    funcs.sort()
    return funcs


def resolve_function(file_offset: int, funcs):
    for fo_start, fo_end, name, seg, off, size in funcs:
        if fo_start <= file_offset < fo_end:
            return name, fo_start, seg, off, size
    return None


def main():
    funcs = load_functions()

    # 讀取整份 disassembly.asm，建立「file_offset -> (seg,off,text)」索引
    # （只需要一次，供近跳轉/遠呼叫目標查表用）
    all_lines = []
    for raw in DISASM.read_text(errors="replace").splitlines():
        m = LINE_RE.match(raw)
        if not m:
            continue
        seg = int(m.group(1), 16)
        off = int(m.group(2), 16)
        text = m.group(3)
        fo = seg_off_to_file_offset(seg, off)
        all_lines.append((fo, seg, off, text))

    fo_index = {fo: (seg, off, text) for fo, seg, off, text in all_lines}
    fo_sorted = sorted(fo_index.keys())

    # 找 FUN_222f_0b0e 本身的邊界
    target = None
    for fo_start, fo_end, name, seg, off, size in funcs:
        if name == "FUN_222f_0b0e":
            target = (fo_start, fo_end, seg, off, size)
            break
    if not target:
        print("找不到 FUN_222f_0b0e", file=sys.stderr)
        sys.exit(1)
    fo_start, fo_end, seg0, off0, size0 = target
    print(f"# FUN_222f_0b0e: file_offset [{fo_start:#x}, {fo_end:#x}), "
          f"seg:off = {seg0:04x}:{off0:04x}, size={size0}")
    print(f"# 下一個函式邊界（含間隙資料）：查 functions.csv 確認")
    print()

    # 收集函式本體範圍內的所有行（含間隙，若 Ghidra 沒展開成指令則不會出現在 disassembly.asm）
    body = [(fo, seg, off, text) for fo, seg, off, text in all_lines
            if fo_start <= fo < fo_end]
    body.sort()

    print(f"# 本函式範圍內共有 {len(body)} 行反組譯（disassembly.asm 有展開的部分）")
    covered = {fo for fo, _, _, _ in body}
    # 找出宣告範圍內「沒有對應反組譯行」的 gap（可能是資料/跳表/未展開）
    gaps = []
    prev = None
    for fo in sorted(covered):
        if prev is not None and fo - prev > 1:
            gaps.append((prev, fo))
        prev = fo
    if gaps:
        print(f"# 發現 {len(gaps)} 個位址不連續的 gap（可能是跳表資料或多 byte 指令間距，正常）")
    print()

    # 逐行印出，並解析 control-flow 目標
    for fo, seg, off, text in body:
        mnem = text.split()[0] if text.split() else ""
        out_line = f"{seg:04x}:{off:04x}\t{text}"
        if CTRLFLOW_MNEM.match(mnem):
            targets = SEGOFF_RE.findall(text)
            resolved = []
            for tseg_s, toff_s in targets:
                tseg = int(tseg_s, 16)
                toff = int(toff_s, 16)
                tfo = seg_off_to_file_offset(tseg, toff)
                if fo_start <= tfo < fo_end:
                    # 落在本函式自己範圍內：換算回本函式的 seg:off 表示（用函式自己的 seg0）
                    local_off = off0 + (tfo - fo_start)
                    resolved.append(f"=> 本函式內 {seg0:04x}:{local_off:04x} (file_off {tfo:#x})")
                else:
                    r = resolve_function(tfo, funcs)
                    if r:
                        name, rstart, rseg, roff, rsize = r
                        delta = tfo - rstart
                        resolved.append(
                            f"=> {name} ({rseg:04x}:{roff:04x}) +{delta:#x} (file_off {tfo:#x})"
                        )
                    else:
                        resolved.append(f"=> 未知位址 file_off {tfo:#x} (原始 {tseg_s}:{toff_s})")
            if resolved:
                out_line += "\t\t" + " | ".join(resolved)
        print(out_line)


if __name__ == "__main__":
    main()
