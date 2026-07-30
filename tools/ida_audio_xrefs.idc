#include <idc.idc>

// IDA 9.4：列出 DEMON.INT 音效選擇器 sub_20485 的所有直接程式交叉引用。
//
// docker run --rm ... ida-pro-9.4-ver2 \
//   idat -A -S/work/tools/ida_audio_xrefs.idc DEMON.INT.i64
static main()
{
  auto target, ea, count, p1, p2, p3;

  target = 0x20485;
  count = 0;
  ea = RfirstB(target);
  while (ea != BADADDR) {
    count = count + 1;
    p1 = prev_head(ea, 0);
    p2 = prev_head(p1, 0);
    p3 = prev_head(p2, 0);
    msg("AUDIO_XREF call=%05X func=%s before=%05X:%s | %05X:%s | %05X:%s\n",
        ea, get_func_name(ea),
        p3, generate_disasm_line(p3, 0),
        p2, generate_disasm_line(p2, 0),
        p1, generate_disasm_line(p1, 0));
    ea = RnextB(target, ea);
  }
  msg("AUDIO_XREF target=%05X count=%d\n", target, count);
  qexit(0);
}
