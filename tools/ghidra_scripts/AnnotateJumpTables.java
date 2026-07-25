// AnnotateJumpTables.java
//
// 修復 Ghidra 對 DEMON.INT 兩個已知 16-bit real mode 間接跳表 switch 的反組譯：
//   1. 138d:258f（15 項）—— 戰鬥動作分派，間接 JMP 在 138d:25b5
//   2. 222f:12ce（19 項）—— 主指令迴圈分派，間接 JMP 在 222f:12fc
//
// 背景：這兩個函式（FUN_138d_1ef8、FUN_222f_0b0e）Ghidra 自動分析只展開了前段
// （讀輸入/邊界檢查），跳表本身與幾乎所有 case body 落在自動分析完全沒碰過的
// 「間隙」裡（不屬於任何函式、disassembly.asm 也沒列出）。decompiler 對這種情況
// 不會報錯，反而會用殘缺的控制流編造出一套看似合理但錯誤的邏輯——這是本專案
// 目前為止最貴的一類錯誤，已造成三次錯誤斷言（見 docs/re/00-ghidra-setup.md 第 5 條、
// docs/re/06 §7、docs/re/08 §5.4、docs/re/09 §0）。
//
// 兩張跳表的位址與內容，已由協調者事先從原始 binary 用 struct.unpack_from('<NH', ...)
// 解出並驗證過，本腳本執行時會另外直接讀 Ghidra 記憶體裡的原始 bytes 做交叉核對
// （不是盲目相信寫死的清單），核對結果印在 log 裡。
//
// 修復手法：第一版只做了「加 COMPUTED_JUMP reference + 重建函式本體」，結果發現
// disassembly.asm 層級完全修好了（函式本體正確擴大、逐行反組譯跟 docs/re/08 手工
// objdump 解出的內容一致），但 decompiler 輸出的 C 碼對 FUN_222f_0b0e 完全沒有變好
// （行數/警告數幾乎和修復前一樣，還是編造出呼叫 FUN_25be_0263 的假控制流）。
//
// 原因：對 BRANCHIND 的解析，decompiler 走的是自己的 p-code 層級 jump table 分析，
// 不會單看 listing 上的 COMPUTED_JUMP reference。要讓 decompiler 真的採用「已知答案」，
// 要用 Ghidra 內建的 jump table override 機制——這不是憑記憶猜的，是直接讀這個 image
// 裡隨附的官方範例腳本 Ghidra/Features/Decompiler/ghidra_scripts/SwitchOverride.java
// 找到的正確用法（分析"Override indirect jump destinations"這個內建功能背後的實作）：
//
//   1. instr.addOperandReference(0, target, COMPUTED_JUMP, USER_DEFINED)  對每個 case
//      目標加 reference（跟第一版一樣，這步的用途是給函式本體重算用的 CFG 提示）。
//   2. CreateFunctionCmd.fixupFunctionBody(program, function, monitor)    用官方 API
//      重算函式本體（取代第一版 remove+createFunction 的土砲做法）。
//   3. new JumpTable(branchAddr, destList, true, 0).writeOverride(function)  ★ 這步
//      才是關鍵：把「這個 BRANCHIND 的目的地清單」寫成 decompiler 真的會讀的 override，
//      不是只有 listing reference。
//
// 完整流程：
//   1. 把跳表本身的記憶體區域標成 word 陣列資料（避免被誤當程式碼反組譯）。
//   2. 對每個 case 目標位址：若尚未反組譯，呼叫 disassemble() 展開。
//   3. 在間接 JMP 指令上加 COMPUTED_JUMP/USER_DEFINED reference（依 case 順序，
//      重複目標不去重——SwitchOverride.java 範例本身也是這樣做，重複加同一條
//      reference 沒有副作用）。
//   4. CreateFunctionCmd.fixupFunctionBody() 重算外層函式本體。
//   5. JumpTable.writeOverride() 寫入 decompiler 真正會用的跳表覆寫。
//   6. analyzeChanges() 讓其餘分析器（stack、data reference…）跟進。
//   7. 呼叫既有的 ExportAnalysis.java 重新匯出五種產出檔案。
//
// 為什麼是 Java 不是 Python：見 ExportAnalysis.java 開頭的說明，Ghidra 12.x 沒有
// 內建 Jython，本專案的 post-script 一律用 Java 寫。
//
// API 簽章全部用 javap 對這個 image 裡實際的 Base.jar / SoftwareModeling.jar
// 反查過，且核心的 JumpTable override 用法直接抄自 image 隨附的官方範例腳本
// SwitchOverride.java（不是憑記憶/猜測拼湊，避免重蹈 docs/re/00 踩雷表第 2 條的
// 覆轍）。

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

public class AnnotateJumpTables extends GhidraScript {

    /** 一張已知跳表的完整描述。 */
    private static class JumpTableSpec {
        final String label;
        final String tableAddrStr;   // 跳表資料本身的位址，如 "138d:258f"
        final String jmpAddrStr;     // 間接 JMP 指令本身的位址，如 "138d:25b5"
        final int opIndex;           // JMP 指令運算元索引（單運算元指令通常是 0）
        final String targetSeg;      // case 目標使用的段（跟跳表同段，近位址）
        final int[] targets;         // 每項目標 offset（協調者已驗證，本腳本會再核對一次）
        final String enclosingFuncAddrStr; // 外層函式進入點，修復後要重算它的函式本體

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

        // 表一：戰鬥動作分派表（docs/re/09 §0，15 項，file offset 0x9a5f）
        // case 0-3 共用 20fb（移動/轉向），4=前進，5=Attack，6=Cast，7=Use，
        // 8=Turn Undead，9=Dodge，10=Examine，11=Sound，12=Pray，13=Leech，14=保留
        specs.add(new JumpTableSpec(
                "combat_action_dispatch (138d:258f)",
                "138d:258f", "138d:25b5", 0, "138d",
                new int[] {
                        0x20fb, 0x20fb, 0x20fb, 0x20fb, 0x2171, 0x23b3, 0x243c, 0x245f,
                        0x2482, 0x24ab, 0x24e0, 0x24f1, 0x2545, 0x2567, 0x2589
                },
                "138d:1ef8"));

        // 表二：主指令迴圈分派表（docs/re/08 §5.4，19 項，file offset 0x171be）
        // case 0-3=方向鍵，4=Walk，5=PartyInfo，6=SaveGame(?)，7-17=共用「回傳 值-1」，
        // 18=Quit，>=19=default（與 7-17 共用同一個 handler，已用 objdump 交叉核對確認）
        specs.add(new JumpTableSpec(
                "main_command_dispatch (222f:12ce)",
                "222f:12ce", "222f:12fc", 0, "222f",
                new int[] {
                        0x0df9, 0x0df9, 0x0df9, 0x0df9, 0x0dff, 0x1274, 0x1282,
                        0x12c8, 0x12c8, 0x12c8, 0x12c8, 0x12c8, 0x12c8, 0x12c8,
                        0x12c8, 0x12c8, 0x12c8, 0x12c8, 0x12a8
                },
                "222f:0b0e"));

        AddressFactory af = currentProgram.getAddressFactory();
        Listing listing = currentProgram.getListing();
        Memory mem = currentProgram.getMemory();

        for (JumpTableSpec spec : specs) {
            println("[jumptable] ========== " + spec.label + " ==========");
            try {
                processOne(spec, af, listing, mem);
            } catch (Exception e) {
                println("[jumptable] !!! 處理 " + spec.label + " 時發生例外，此表略過但不中斷後續表: "
                        + e);
                e.printStackTrace();
            }
        }

        println("[jumptable] 全部表處理完，觸發 analyzeChanges() 讓其餘分析器跟進 ...");
        analyzeChanges(currentProgram);
        println("[jumptable] analyzeChanges 完成");

        // 沿用既有匯出腳本，重新匯出五種產出檔案（functions.csv / strings.csv /
        // disassembly.asm / decompiled_all.c / decompiled/*.c）到同一個
        // DW_EXPORT_DIR。這裡直接呼叫既有腳本而不重複實作，避免匯出邏輯分裂成兩份。
        println("[jumptable] 呼叫 ExportAnalysis.java 重新匯出 ...");
        runScript("ExportAnalysis.java");
        println("[jumptable] 全部完成");
    }

    private void processOne(JumpTableSpec spec, AddressFactory af, Listing listing, Memory mem)
            throws Exception {
        Address tableAddr = af.getAddress(spec.tableAddrStr);
        Address jmpAddr = af.getAddress(spec.jmpAddrStr);
        int n = spec.targets.length;

        // 0. 交叉驗證：直接讀 Ghidra 記憶體裡的原始 bytes，跟寫死在本腳本裡、
        //    協調者已驗證過的清單比對。兩邊都不吻合就是真的有問題，要如實印出來，
        //    不能悶著頭繼續套用寫死的清單（見 rulebook 62：斷言前先驗證）。
        boolean allMatch = true;
        for (int i = 0; i < n; i++) {
            Address entryAddr = tableAddr.add(i * 2L);
            short raw = mem.getShort(entryAddr);
            int actual = raw & 0xFFFF;
            if (actual != spec.targets[i]) {
                println("[jumptable] !!! 記憶體核對不符 idx=" + i + " 記憶體實際值=0x"
                        + Integer.toHexString(actual) + " 腳本裡寫死的值=0x"
                        + Integer.toHexString(spec.targets[i]));
                allMatch = false;
            }
        }
        println("[jumptable] 記憶體核對(vs 協調者提供清單): " + (allMatch ? "全部 " + n + " 項吻合" : "有落差，見上方"));

        // 1. 跳表區域清成未定義，再標成 word 陣列（避免被誤判成程式碼）
        Address tableEnd = tableAddr.add(n * 2L - 1);
        clearListing(tableAddr, tableEnd);
        Data arrayData = createData(tableAddr, new ArrayDataType(WordDataType.dataType, n, 2));
        println("[jumptable] 跳表資料標註完成: " + tableAddr + " - " + tableEnd
                + "（" + n + " words，" + (arrayData != null ? "createData 成功" : "createData 回傳 null") + "）");

        // 2. 對每個 case 目標：若尚未反組譯就展開。同時依 case 順序（保留重複）
        //    組出目的地清單，供步驟 3/5 使用。
        List<Address> destListOrdered = new ArrayList<>();
        int disassembledCount = 0;
        int alreadyPresentCount = 0;
        for (int i = 0; i < n; i++) {
            Address target = af.getAddress(spec.targetSeg + ":" + String.format("%04x", spec.targets[i]));
            boolean hasInstr = listing.getInstructionAt(target) != null;
            boolean hasData = listing.getDefinedDataAt(target) != null;
            if (!hasInstr && !hasData) {
                boolean ok = disassemble(target);
                println("[jumptable] case idx=" + i + " target=" + target
                        + " 原本未展開，disassemble() -> " + ok);
                if (ok) {
                    disassembledCount++;
                }
            } else {
                alreadyPresentCount++;
            }
            destListOrdered.add(target);
        }
        println("[jumptable] case 目標統計: 共 " + n + " 項，其中本次新展開 " + disassembledCount
                + " 個、原本已存在 " + alreadyPresentCount + " 個");

        // 3. 在間接 JMP 指令上，依 case 順序加 COMPUTED_JUMP reference（不去重，
        //    照抄 Ghidra 官方範例 SwitchOverride.java 的做法——重複加同一條
        //    reference 沒有副作用）。
        Instruction jmpInstr = listing.getInstructionAt(jmpAddr);
        if (jmpInstr == null) {
            println("[jumptable] !!! 找不到 JMP 指令 @ " + jmpAddr + "，無法加 reference，此表其餘步驟略過");
            return;
        }
        int opIndex = spec.opIndex;
        if (opIndex >= jmpInstr.getNumOperands()) {
            println("[jumptable] !!! opIndex=" + opIndex + " 超出 JMP 指令運算元數("
                    + jmpInstr.getNumOperands() + ")，改用最後一個運算元");
            opIndex = Math.max(0, jmpInstr.getNumOperands() - 1);
        }
        for (Address target : destListOrdered) {
            jmpInstr.addOperandReference(opIndex, target, RefType.COMPUTED_JUMP, SourceType.USER_DEFINED);
        }
        println("[jumptable] 已在 " + jmpAddr + "（運算元 " + opIndex + "）加上 " + destListOrdered.size()
                + " 條 COMPUTED_JUMP/USER_DEFINED reference（依 case 順序，含重複目標）");

        // 4. 用官方 API 重算外層函式本體（取代土砲的 remove+createFunction）。
        //    現在 CFG 已經包含新加的 computed-jump 邊，本體會正確納入 case body，
        //    遇到下一個已存在的函式進入點（如 FUN_222f_1321）時仍會正確停止。
        Address funcAddr = af.getAddress(spec.enclosingFuncAddrStr);
        Function func = getFunctionAt(funcAddr);
        if (func == null) {
            println("[jumptable] !!! 外層函式 @ " + funcAddr + " 不存在，略過重算函式本體與寫入 jump table override");
            return;
        }
        long beforeSize = func.getBody().getNumAddresses();
        boolean fixedUp = CreateFunctionCmd.fixupFunctionBody(currentProgram, func, monitor);
        Function refreshed = getFunctionAt(funcAddr);
        long afterSize = refreshed != null ? refreshed.getBody().getNumAddresses() : -1;
        println("[jumptable] fixupFunctionBody(" + funcAddr + ") -> " + fixedUp
                + "，本體大小 " + beforeSize + " bytes -> " + afterSize + " bytes");
        if (refreshed == null) {
            println("[jumptable] !!! fixupFunctionBody 後找不到函式，略過寫入 jump table override");
            return;
        }
        boolean containsJmp = refreshed.getBody().contains(jmpAddr);
        println("[jumptable] 重算後的函式本體是否已包含 JMP 指令位址 " + jmpAddr + " ？ " + containsJmp);

        // 5. 寫入 decompiler 真正會讀的 jump table override（關鍵步驟，第一版漏掉
        //    這步，只加 listing reference 不夠——decompiler 對 BRANCHIND 的解析走
        //    自己的 p-code 層級分析，不會單看 listing reference）。
        //    用法照抄 image 隨附的官方範例 SwitchOverride.java：
        //      new JumpTable(branchAddr, destList, true, 0).writeOverride(function)
        JumpTable jumpTab = new JumpTable(jmpAddr, new ArrayList<>(destListOrdered), true, 0);
        jumpTab.writeOverride(refreshed);
        println("[jumptable] JumpTable.writeOverride() 完成，已寫入 " + destListOrdered.size() + " 項目的地");
    }
}
