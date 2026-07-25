// AnnotateJumpTablesSweep.java
//
// 全檔跳表掃描 + 修復（DEMON.INT）。延續 docs/re/12-ghidra-jumptable-fix.md 與
// tools/ghidra_scripts/AnnotateJumpTables.java 已驗證的手法（COMPUTED_JUMP
// reference + fixupFunctionBody + JumpTable.writeOverride()），把範圍從「兩張
// 已知表」擴大到「全檔同類間接跳轉」。
//
// 背景：docs/re/16-combat-details.md 追出第三張表在 FUN_138d_3c81
// （138d:3f95，18 項，索引即 spell/item 效果類型 [0x4e2e]），擋住即死／束縛／
// 枯萎判定與 Use 道具效果套用的反編譯。本腳本先修好這張表，再對全檔做同類
// pattern 掃描找出其餘同款跳表。
//
// 全檔掃描方法（用 Python 對原始 binary 做位元組層級掃描，見
// docs/re/18-jumptable-sweep.md §2 附的可重跑片段，此處只列結論）：
// 這個編譯器對「間接跳轉 switch」只用一種固定慣用法（13 bytes）：
//
//   3D NN 00        CMP AX, NN            ; NN = case 數量，AX 已是 0-based 索引
//   73 08            JAE +8                ; 超界跳到「緊接在 JMP 後面」的 default
//   93               XCHG AX,BX
//   D1 E3            SHL BX,1
//   2E FF A7 xx xx   JMP word ptr CS:[BX+disp16]   ; disp16 = 跳表在同一 segment 內的 offset
//
// 全檔搜尋固定的 3-byte 錨點「2E FF A7」（jmp word ptr cs:[bx+disp16] 的
// prefix+opcode+modrm），總共命中 21 處，逐一往回 8 bytes 核對「CMP/JAE/XCHG/
// SHL」是否吻合——**21 處全部吻合**，包含 docs/re/12 已修復的兩張已知表
// （138d:258f n=15、222f:12ce n=19，其 NN 值與已驗證的 15/19 完全對上，
// 交叉驗證了掃描器本身正確）。因此本腳本共處理 21 張表（含重複修復那兩張
// 已知表——因為 tools/ghidra_headless.sh 每次都 -import -overwrite，
// 標註必須在同一次 headless 呼叫內全部做完，見 docs/re/12「限制」一節）。
//
// 驗證步驟（每張表標註前都做，不合格的不標，見 docs/re/18-jumptable-sweep.md）：
//   1. 項數（NN）與 CMP 邊界檢查吻合（掃描器直接從同一組 bytes 讀出兩者，
//      結構上不可能不吻合，但仍額外做一次記憶體核對，見 processOne()）。
//   2. 全部目標位址落在「所屬 segment 的函式覆蓋範圍」內（用 functions.csv
//      算出每個 segment 的 file-offset 範圍，21 張表的目標全部落在範圍內，
//      無一逸出——已用獨立 Python 腳本驗證，見 docs/re/18 附錄）。
//   3. 目標不與跳表本身重疊。
// 21 張全部通過，沒有「合理但跳過」的情況，此腳本因此標註全部 21 張。
//
// 為什麼是 Java：Ghidra 12.x 沒有內建 Jython，post-script 一律 Java（見
// docs/re/00-ghidra-setup.md 踩雷 1）。JumpTable.writeOverride() 用法照抄
// image 隨附的官方範例 Ghidra/Features/Decompiler/ghidra_scripts/SwitchOverride.java
// （docs/re/12 已驗證這是「decompiler 真的會採用」的唯一正確做法，光加
// COMPUTED_JUMP reference 不夠）。

import ghidra.app.cmd.function.CreateFunctionCmd;
import ghidra.app.script.GhidraScript;
import ghidra.program.model.address.Address;
import ghidra.program.model.address.AddressFactory;
import ghidra.program.model.data.ArrayDataType;
import ghidra.program.model.data.WordDataType;
import ghidra.program.model.listing.Data;
import ghidra.program.model.listing.Function;
import ghidra.program.model.listing.Instruction;
import ghidra.program.model.listing.Listing;
import ghidra.program.model.mem.Memory;
import ghidra.program.model.pcode.JumpTable;
import ghidra.program.model.symbol.RefType;
import ghidra.program.model.symbol.SourceType;

import java.util.ArrayList;
import java.util.List;

public class AnnotateJumpTablesSweep extends GhidraScript {

    /** 一張已知跳表的完整描述（與 AnnotateJumpTables.java 相同結構）。 */
    private static class JumpTableSpec {
        final String label;
        final String tableAddrStr;
        final String jmpAddrStr;
        final int opIndex;
        final String targetSeg;
        final int[] targets;
        final String enclosingFuncAddrStr;

        JumpTableSpec(String label, String tableAddrStr, String jmpAddrStr, int opIndex,
                String targetSeg, int[] targets, String enclosingFuncAddrStr) {
            this.label = label;
            this.tableAddrStr = tableAddrStr;
            this.jmpAddrStr = jmpAddrStr;
            this.opIndex = opIndex;
            this.targetSeg = targetSeg;
            this.targets = targets;
            this.enclosingFuncAddrStr = enclosingFuncAddrStr;
        }
    }

    @Override
    protected void run() throws Exception {
        List<JumpTableSpec> specs = new ArrayList<>();

        // ===== 處理順序刻意調整（重要）=====
        // 同一個 segment 內，一律照位址由小到大處理，且「風險較高、會讓 fixupFunctionBody
        // 往外擴張很多的表」排在「範圍較小、緊鄰在後的函式」之前——這樣後處理的
        // fixupFunctionBody 在往外長的時候，會撞到「已經存在的函式」而停下來，不會反過來
        // 把後面那個函式的進入點併吞掉。這是第一輪執行後才發現的教訓（見
        // docs/re/18-jumptable-sweep.md「踩雷」一節）：
        //   1. 138d 段：FUN_138d_3c81（目標三）第一輪處理時 getFunctionAt 回傳 null——
        //      因為它在自動分析裡的存在依賴前面 FUN_138d_2e63 的跳表先修好，兩者必須照
        //      位址順序（065e → 2e63 → 3c81）處理，中間插隊會讓後面的自我修復
        //      （createFunction fallback，見 processOne 步驟 4）派上用場。
        //   2. 25be 段：FUN_25be_0263 內部有兩張跳表，fixupFunctionBody 展開後的範圍會
        //      一路長到緊鄰的 FUN_25be_0e77 進入點。把 FUN_25be_0e77 的表（139a）提前到
        //      0263 的兩張表之前處理，讓 0e77 先被鎖定成一個「已存在」的函式，0263 展開
        //      時才會在 0e77 的進入點前停下來。

        // ===== 已知的兩張表（docs/re/12，重跑一次因為每次 -overwrite 全新匯入）=====

        specs.add(new JumpTableSpec(
                "table_1000_01c7 (1000:01c7) func=FUN_1000_01e3",
                "1000:01c7", "1000:01eb", 0, "1000",
                new int[] { 0xa7, 0xb8, 0xce, 0xdd, 0xee, 0x102, 0x113, 0x124, 0x135, 0x146, 0x157, 0x16b, 0x177, 0x187 },
                "1000:01e3"));

        specs.add(new JumpTableSpec(
                "table_1000_1820 (1000:1820) func=FUN_1000_11e5",
                "1000:1820", "1000:184c", 0, "1000",
                new int[] { 0x1851, 0x1351, 0x1351, 0x1351, 0x1351, 0x1385, 0x1351, 0x1351, 0x14df, 0x14df, 0x1614, 0x1351, 0x1714, 0x1385, 0x1351, 0x1351, 0x1351, 0x177b },
                "1000:11e5"));

        specs.add(new JumpTableSpec(
                "table_1000_33c3 (1000:33c3) func=FUN_1000_2a53",
                "1000:33c3", "1000:33dd", 0, "1000",
                new int[] { 0x2e04, 0x304a, 0x304a, 0x304a, 0x304a, 0x304a, 0x3366, 0x3385, 0x33a4 },
                "1000:2a53"));

        specs.add(new JumpTableSpec(
                "table_138d_0c65 (138d:0c65) func=FUN_138d_065e",
                "138d:0c65", "138d:0c8f", 0, "138d",
                new int[] { 0xc63, 0x868, 0x9fd, 0xa4d, 0xa4d, 0xa4d, 0xa4d, 0xa4d, 0xc63, 0xc63, 0xc63, 0xb29, 0xc63, 0xc63, 0xadc, 0xc63, 0xbef },
                "138d:065e"));

        specs.add(new JumpTableSpec(
                "combat_action_dispatch (138d:258f)",
                "138d:258f", "138d:25b5", 0, "138d",
                new int[] {
                        0x20fb, 0x20fb, 0x20fb, 0x20fb, 0x2171, 0x23b3, 0x243c, 0x245f,
                        0x2482, 0x24ab, 0x24e0, 0x24f1, 0x2545, 0x2567, 0x2589
                },
                "138d:1ef8"));

        specs.add(new JumpTableSpec(
                "table_138d_2f4c (138d:2f4c) func=FUN_138d_2e63",
                "138d:2f4c", "138d:2f76", 0, "138d",
                new int[] { 0x2f7b, 0x2e6c, 0x2e84, 0x2e9a, 0x2e9a, 0x2e9a, 0x2e9a, 0x2e9a, 0x2eb0, 0x2ed3, 0x2ee9, 0x2efc, 0x2eb0, 0x2e9a, 0x2f10, 0x2f24, 0x2f38 },
                "138d:2e63"));

        // ===== 第三張表（本輪任務目標）=====
        // FUN_138d_3c81：Use 道具／怪物 AI 效果套用引擎的第二層分派（索引=[0x4e2e]
        // 效果類型），擋住即死／束縛／枯萎判定的反編譯。18 項，file offset 0xb465
        // (138d:3f95)，間接 JMP 在 138d:3fc1，CMP AX,0x12(18) 邊界檢查已核對。

        specs.add(new JumpTableSpec(
                "effect_type_dispatch_TARGET3 (138d:3f95)",
                "138d:3f95", "138d:3fc1", 0, "138d",
                new int[] {
                        0x3fc6, 0x3c8d, 0x3cc5, 0x3cff, 0x3cff, 0x3cff, 0x3cff, 0x3cff,
                        0x3d98, 0x3dcf, 0x3e25, 0x3e7f, 0x3d98, 0x3cff, 0x3f0f, 0x3f48,
                        0x3f5c, 0x3d98
                },
                "138d:3c81"));

        specs.add(new JumpTableSpec(
                "main_command_dispatch (222f:12ce)",
                "222f:12ce", "222f:12fc", 0, "222f",
                new int[] {
                        0x0df9, 0x0df9, 0x0df9, 0x0df9, 0x0dff, 0x1274, 0x1282,
                        0x12c8, 0x12c8, 0x12c8, 0x12c8, 0x12c8, 0x12c8, 0x12c8,
                        0x12c8, 0x12c8, 0x12c8, 0x12c8, 0x12a8
                },
                "222f:0b0e"));

        // ===== 全檔掃描新發現的其餘 15 張表 =====
        // 掃描方法見檔頭註解；每張都通過「項數吻合 + 目標落在所屬 segment 函式
        // 覆蓋範圍內 + 不與跳表本身重疊」驗證，見 docs/re/18-jumptable-sweep.md。

        specs.add(new JumpTableSpec(
                "table_1990_40af (1990:40af) func=FUN_1990_3da0",
                "1990:40af", "1990:40d5", 0, "1990",
                new int[] { 0x3e89, 0x3e89, 0x3e89, 0x3e89, 0x40da, 0x40da, 0x40da, 0x40da, 0x40da, 0x40da, 0x3f0b, 0x40da, 0x3f83, 0x40da, 0x409f },
                "1990:3da0"));

        specs.add(new JumpTableSpec(
                "table_1d9f_1dd0 (1d9f:1dd0) func=FUN_1d9f_1ce1",
                "1d9f:1dd0", "1d9f:1de2", 0, "1d9f",
                new int[] { 0x1dbb, 0x1dc2, 0x1dc9, 0x1dc9, 0x1dc9 },
                "1d9f:1ce1"));

        specs.add(new JumpTableSpec(
                "table_1d9f_2beb (1d9f:2beb) func=FUN_1d9f_2a95",
                "1d9f:2beb", "1d9f:2c0a", 0, "1d9f",
                new int[] { 0x2b5f, 0x2c0f, 0x2ab7, 0x2acc, 0x2ae1, 0x2af6, 0x2b0b, 0x2b20, 0x2b35, 0x2b4a },
                "1d9f:2a95"));

        specs.add(new JumpTableSpec(
                "table_206a_0ea0 (206a:0ea0) func=FUN_206a_02c7",
                "206a:0ea0", "206a:0eb7", 0, "206a",
                new int[] { 0x8df, 0x72c, 0x8ba, 0x364, 0xebc, 0x34b },
                "206a:02c7"));

        specs.add(new JumpTableSpec(
                "table_222f_0563 (222f:0563) func=FUN_222f_0003",
                "222f:0563", "222f:0598", 0, "222f",
                new int[] { 0x3ca, 0x3d5, 0x3e0, 0x3eb, 0x3f6, 0x401, 0x40c, 0x41c, 0x454, 0x46c, 0x477, 0x482, 0x48d, 0x49d, 0x4a8, 0x4b3, 0x4bb, 0x4c6, 0x50e, 0x519, 0x545 },
                "222f:0003"));

        // 注意：25be:0e77 的表故意排在 25be:0263 的兩張表「之前」——見上方處理順序
        // 說明第 2 點，避免 FUN_25be_0263 的 fixupFunctionBody 展開範圍併吞
        // FUN_25be_0e77 的進入點。

        specs.add(new JumpTableSpec(
                "table_25be_139a (25be:139a) func=FUN_25be_0e77",
                "25be:139a", "25be:13b0", 0, "25be",
                new int[] { 0x12fc, 0x130e, 0x1328, 0x133a, 0x1353, 0x138c, 0x138c },
                "25be:0e77"));

        specs.add(new JumpTableSpec(
                "table_25be_06dd (25be:06dd) func=FUN_25be_0263",
                "25be:06dd", "25be:06f7", 0, "25be",
                new int[] { 0x6fc, 0x2f4, 0x350, 0x350, 0x47b, 0x47b, 0x5b1, 0x5b1, 0x6bb },
                "25be:0263"));

        specs.add(new JumpTableSpec(
                "table_25be_0deb (25be:0deb) func=FUN_25be_0263 (同一函式第二張表)",
                "25be:0deb", "25be:0e13", 0, "25be",
                new int[] { 0xe18, 0x77b, 0x7a4, 0x85c, 0x919, 0x989, 0xa48, 0xa50, 0xa83, 0xaef, 0xaf7, 0xbb3, 0xc5f, 0xcd8, 0xd6b, 0xd85 },
                "25be:0263"));

        specs.add(new JumpTableSpec(
                "table_278d_08ec (278d:08ec) func=FUN_278d_0098",
                "278d:08ec", "278d:090c", 0, "278d",
                new int[] { 0x7c1, 0x7d4, 0x7e0, 0x80e, 0x81a, 0x83a, 0x846, 0x852, 0x8c4, 0x8c4, 0x8c4, 0x8e6 },
                "278d:0098"));

        specs.add(new JumpTableSpec(
                "table_278d_0beb (278d:0beb) func=FUN_278d_0932",
                "278d:0beb", "278d:0c08", 0, "278d",
                new int[] { 0xb61, 0xc0d, 0xb69, 0xc0d, 0xb50, 0xc0d, 0xb7a, 0xc0d, 0xbd9 },
                "278d:0932"));

        specs.add(new JumpTableSpec(
                "table_278d_2b83 (278d:2b83) func=FUN_278d_22bc",
                "278d:2b83", "278d:2ba7", 0, "278d",
                new int[] { 0x28ea, 0x2a66, 0x2754, 0x273a, 0x2768, 0x2843, 0x26ff, 0x26ff, 0x26ff, 0x26ff, 0x26ff, 0x2b71, 0x2a8b, 0x2868 },
                "278d:22bc"));

        specs.add(new JumpTableSpec(
                "table_2aed_0b7d (2aed:0b7d) func=FUN_2aed_07be",
                "2aed:0b7d", "2aed:0b8f", 0, "2aed",
                new int[] { 0xaf0, 0xb0d, 0xb29, 0xb45, 0xb61 },
                "2aed:07be"));

        specs.add(new JumpTableSpec(
                "table_2aed_1d63 (2aed:1d63) func=FUN_2aed_14c2",
                "2aed:1d63", "2aed:1da9", 0, "2aed",
                new int[] {
                        0x1b9a, 0x1d61, 0x1d61, 0x1d61, 0x1bb0, 0x1bea, 0x1bea, 0x1bea,
                        0x1bea, 0x1bea, 0x1bea, 0x1bea, 0x1d61, 0x1d61, 0x1d61, 0x1c0c,
                        0x1c0c, 0x1c57, 0x1c57, 0x1c57, 0x1c57, 0x1c57, 0x1cff, 0x1cff,
                        0x1cff, 0x1cff, 0x1cff, 0x1cff, 0x1cff, 0x1cff, 0x1cff
                },
                "2aed:14c2"));

        AddressFactory af = currentProgram.getAddressFactory();
        Listing listing = currentProgram.getListing();
        Memory mem = currentProgram.getMemory();

        int okCount = 0;
        int failCount = 0;
        for (JumpTableSpec spec : specs) {
            println("[sweep] ========== " + spec.label + " ==========");
            try {
                boolean ok = processOne(spec, af, listing, mem);
                if (ok) {
                    okCount++;
                } else {
                    failCount++;
                    println("[sweep] !!! " + spec.label + " 驗證未通過，已跳過標註（見上方訊息）");
                }
            } catch (Exception e) {
                failCount++;
                println("[sweep] !!! 處理 " + spec.label + " 時發生例外，此表略過但不中斷後續表: " + e);
                e.printStackTrace();
            }
        }
        println("[sweep] 全部 " + specs.size() + " 張表處理完畢：成功 " + okCount + "，跳過/失敗 " + failCount);

        println("[sweep] 觸發 analyzeChanges() 讓其餘分析器跟進 ...");
        analyzeChanges(currentProgram);
        println("[sweep] analyzeChanges 完成");

        println("[sweep] 呼叫 ExportAnalysis.java 重新匯出 ...");
        runScript("ExportAnalysis.java");
        println("[sweep] 全部完成");
    }

    /** 回傳 true 表示這張表通過驗證並完成標註；false 表示驗證未通過、已跳過。 */
    private boolean processOne(JumpTableSpec spec, AddressFactory af, Listing listing, Memory mem)
            throws Exception {
        Address tableAddr = af.getAddress(spec.tableAddrStr);
        Address jmpAddr = af.getAddress(spec.jmpAddrStr);
        int n = spec.targets.length;

        // 0. 記憶體核對：直接讀 Ghidra 記憶體裡的原始 bytes，跟寫死在本腳本裡、
        //    已用獨立 Python 腳本驗證過的清單比對。不吻合就跳過，不硬標
        //    （任務單明確要求：不通過的跳過並記錄，不要硬標）。
        boolean allMatch = true;
        for (int i = 0; i < n; i++) {
            Address entryAddr = tableAddr.add(i * 2L);
            short raw = mem.getShort(entryAddr);
            int actual = raw & 0xFFFF;
            if (actual != spec.targets[i]) {
                println("[sweep] !!! 記憶體核對不符 idx=" + i + " 記憶體實際值=0x"
                        + Integer.toHexString(actual) + " 腳本裡寫死的值=0x"
                        + Integer.toHexString(spec.targets[i]));
                allMatch = false;
            }
        }
        println("[sweep] 記憶體核對: " + (allMatch ? "全部 " + n + " 項吻合" : "有落差，見上方"));
        if (!allMatch) {
            return false;
        }

        // 1. 跳表區域清成未定義，再標成 word 陣列
        Address tableEnd = tableAddr.add(n * 2L - 1);
        clearListing(tableAddr, tableEnd);
        Data arrayData = createData(tableAddr, new ArrayDataType(WordDataType.dataType, n, 2));
        println("[sweep] 跳表資料標註完成: " + tableAddr + " - " + tableEnd
                + "（" + n + " words，" + (arrayData != null ? "createData 成功" : "createData 回傳 null") + "）");

        // 2. 展開每個 case 目標
        List<Address> destListOrdered = new ArrayList<>();
        int disassembledCount = 0;
        int alreadyPresentCount = 0;
        for (int i = 0; i < n; i++) {
            Address target = af.getAddress(spec.targetSeg + ":" + String.format("%04x", spec.targets[i]));
            boolean hasInstr = listing.getInstructionAt(target) != null;
            boolean hasData = listing.getDefinedDataAt(target) != null;
            if (!hasInstr && !hasData) {
                boolean ok = disassemble(target);
                if (ok) {
                    disassembledCount++;
                } else {
                    println("[sweep] case idx=" + i + " target=" + target + " disassemble() 失敗");
                }
            } else {
                alreadyPresentCount++;
            }
            destListOrdered.add(target);
        }
        println("[sweep] case 目標統計: 共 " + n + " 項，本次新展開 " + disassembledCount
                + " 個、原本已存在 " + alreadyPresentCount + " 個");

        // 3. 加 COMPUTED_JUMP reference（依 case 順序，不去重）
        // 先確保間接 JMP 指令本身已展開——多數表的 JMP 指令在自動分析階段就已經
        // 被線性掃描到（因為 CMP/JAE/XCHG/SHL/JMP 這串指令是從函式進入點一路
        // fall-through 下來的直線路徑），但至少一張表（table_25be_139a）遇到
        // 自動分析完全沒展開到這段位址的情況（即使函式本身存在，body 也沒
        // 涵蓋到這裡）。這裡的位元組樣式已經在掃描階段用 Python 逐一核對過
        // （CMP AX,N / JAE +8 / XCHG AX,BX / SHL BX,1 / JMP word cs:[BX+disp16]，
        // 21 處全部吻合），所以在這裡主動 disassemble() 是安全的，不是憑空猜測。
        if (listing.getInstructionAt(jmpAddr) == null) {
            boolean ok = disassemble(jmpAddr);
            println("[sweep] JMP 指令 @ " + jmpAddr + " 原本未展開，disassemble() -> " + ok);
        }
        Instruction jmpInstr = listing.getInstructionAt(jmpAddr);
        if (jmpInstr == null) {
            println("[sweep] !!! 找不到 JMP 指令 @ " + jmpAddr + "，無法加 reference，此表跳過");
            return false;
        }
        int opIndex = spec.opIndex;
        if (opIndex >= jmpInstr.getNumOperands()) {
            opIndex = Math.max(0, jmpInstr.getNumOperands() - 1);
        }
        for (Address target : destListOrdered) {
            jmpInstr.addOperandReference(opIndex, target, RefType.COMPUTED_JUMP, SourceType.USER_DEFINED);
        }
        println("[sweep] 已在 " + jmpAddr + "（運算元 " + opIndex + "）加上 " + destListOrdered.size()
                + " 條 COMPUTED_JUMP/USER_DEFINED reference");

        // 4. 重算外層函式本體
        Address funcAddr = af.getAddress(spec.enclosingFuncAddrStr);
        Function func = getFunctionAt(funcAddr);
        if (func == null) {
            // 自我修復：這一輪全新 -overwrite 匯入的自動分析本身有已知的非決定性
            // （docs/re/00 §「自動分析覆蓋品質評估」記錄過 ±5 函式數量的浮動），
            // 有時預期的函式進入點在自動分析階段沒被建立成獨立函式（可能被前一個
            // 未修復的跳表產生的錯誤 CFG 併吞）。這裡主動用 createFunction() 在
            // 這個位址重新切出一個函式，而不是直接放棄——這正是「跳表 override」
            // 這整套修復手法要解決的問題本身。
            Function containing = getFunctionContaining(funcAddr);
            println("[sweep] 外層函式 @ " + funcAddr + " 不存在（getFunctionAt 為 null）"
                    + "，getFunctionContaining -> "
                    + (containing != null ? containing.getName() + "@" + containing.getEntryPoint() : "null")
                    + "，嘗試 createFunction() 自我修復");
            if (listing.getInstructionAt(funcAddr) == null) {
                disassemble(funcAddr);
            }
            func = createFunction(funcAddr, null);
            if (func == null) {
                println("[sweep] !!! createFunction(" + funcAddr + ") 仍然失敗，此表真的跳過");
                return false;
            }
            println("[sweep] createFunction(" + funcAddr + ") 成功: " + func.getName());
        }
        long beforeSize = func.getBody().getNumAddresses();
        boolean fixedUp = CreateFunctionCmd.fixupFunctionBody(currentProgram, func, monitor);
        Function refreshed = getFunctionAt(funcAddr);
        long afterSize = refreshed != null ? refreshed.getBody().getNumAddresses() : -1;
        println("[sweep] fixupFunctionBody(" + funcAddr + ") -> " + fixedUp
                + "，本體大小 " + beforeSize + " bytes -> " + afterSize + " bytes");
        if (refreshed == null) {
            println("[sweep] !!! fixupFunctionBody 後找不到函式，跳過寫入 override");
            return false;
        }

        // 5. 寫入 decompiler 真正會讀的 jump table override（關鍵步驟）
        JumpTable jumpTab = new JumpTable(jmpAddr, new ArrayList<>(destListOrdered), true, 0);
        jumpTab.writeOverride(refreshed);
        println("[sweep] JumpTable.writeOverride() 完成，已寫入 " + destListOrdered.size() + " 項目的地");
        return true;
    }
}
