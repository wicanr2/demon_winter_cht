// ExportAnalysis.java
//
// Demon's Winter (DEMON.INT) Ghidra headless post-analysis 匯出腳本。
// 用 analyzeHeadless -postScript 呼叫，於自動分析跑完後執行。
//
// 為什麼是 Java 不是 Python：Ghidra 12.x 移除了內建 Jython script provider，
// .py post-script 一律要靠 PyGhidra（CPython + jpype，需另外 pip 環境配好版本）
// 才能跑。為了讓這個匯出環境不多背一層 Python/venv 相依，改用 Java GhidraScript
// —— analyzeHeadless 內建 javac 編譯執行，JDK image 本身就夠，零額外相依。
// 詳細踩雷記錄見 docs/re/00-ghidra-setup.md。
//
// 匯出到環境變數 DW_EXPORT_DIR 指定目錄（沒設就退回 /work/export）：
//   functions.csv        — 位址, 名稱, 大小(bytes), is_thunk
//   strings.csv           — 位址, 長度, 內容
//   disassembly.asm        — 全域反組譯（逐指令）
//   decompiled/<addr>_<name>.c — 每個函式各一個反編譯 .c 檔
//   decompiled_all.c        — 全部函式反編譯串接（方便 grep）
//
// 16-bit real mode 位址以 segment:offset 顯示（如 2037:0009），本腳本原樣輸出
// Ghidra 顯示的位址字串，不重算 linear address —— 對回檔案位移的換算公式見
// docs/re/00-ghidra-setup.md。

import ghidra.app.script.GhidraScript;
import ghidra.app.decompiler.DecompInterface;
import ghidra.app.decompiler.DecompileOptions;
import ghidra.app.decompiler.DecompileResults;
import ghidra.program.model.listing.Function;
import ghidra.program.model.listing.FunctionManager;
import ghidra.program.model.listing.Instruction;
import ghidra.program.model.listing.InstructionIterator;
import ghidra.program.model.listing.Listing;
import ghidra.program.model.data.StringDataInstance;
import ghidra.program.model.listing.Data;
import ghidra.program.util.DefinedDataIterator;
import ghidra.util.task.ConsoleTaskMonitor;

import java.io.File;
import java.io.FileWriter;
import java.io.PrintWriter;
import java.util.Iterator;

public class ExportAnalysis extends GhidraScript {

    private File exportDir;

    @Override
    protected void run() throws Exception {
        String dw = System.getenv("DW_EXPORT_DIR");
        exportDir = new File(dw != null ? dw : "/work/export");
        if (!exportDir.exists()) {
            exportDir.mkdirs();
        }
        println("[export] 匯出目錄: " + exportDir.getAbsolutePath());

        Listing listing = currentProgram.getListing();
        FunctionManager fm = currentProgram.getFunctionManager();

        exportFunctions(fm);
        exportStrings();
        exportDisassembly(listing);
        exportDecompiled(fm);

        println("[export] 完成");
    }

    private void exportFunctions(FunctionManager fm) throws Exception {
        File out = new File(exportDir, "functions.csv");
        try (PrintWriter w = new PrintWriter(new FileWriter(out))) {
            w.println("address,name,size_bytes,is_thunk");
            int count = 0;
            for (Function func : fm.getFunctions(true)) {
                String addr = func.getEntryPoint().toString();
                String name = csvEscape(func.getName());
                long size = func.getBody().getNumAddresses();
                boolean thunk = func.isThunk();
                w.println(addr + "," + name + "," + size + "," + thunk);
                count++;
            }
            println("[export] functions.csv 寫入 " + count + " 筆函式");
        }
    }

    private void exportStrings() throws Exception {
        File out = new File(exportDir, "strings.csv");
        try (PrintWriter w = new PrintWriter(new FileWriter(out))) {
            w.println("address,length,content");
            int count = 0;
            Iterator<Data> it = DefinedDataIterator.byDataInstance(currentProgram, Data::hasStringValue);
            while (it.hasNext()) {
                Data data = it.next();
                try {
                    StringDataInstance sdi = StringDataInstance.getStringDataInstance(data);
                    String s = sdi.getStringValue();
                    if (s == null) {
                        Object val = data.getValue();
                        if (val == null) {
                            continue;
                        }
                        s = val.toString();
                    }
                    String addr = data.getAddress().toString();
                    w.println(addr + "," + s.length() + "," + csvEscape(s));
                    count++;
                } catch (Exception e) {
                    // 個別字串解析失敗不中斷整體匯出
                }
            }
            println("[export] strings.csv 寫入 " + count + " 筆字串");
        }
    }

    private void exportDisassembly(Listing listing) throws Exception {
        File out = new File(exportDir, "disassembly.asm");
        try (PrintWriter w = new PrintWriter(new FileWriter(out))) {
            int count = 0;
            InstructionIterator it = listing.getInstructions(true);
            while (it.hasNext()) {
                Instruction instr = it.next();
                w.println(instr.getAddress().toString() + "\t" + instr.toString());
                count++;
            }
            println("[export] disassembly.asm 寫入 " + count + " 行指令");
        }
    }

    private void exportDecompiled(FunctionManager fm) throws Exception {
        File decompDir = new File(exportDir, "decompiled");
        if (!decompDir.exists()) {
            decompDir.mkdirs();
        }
        File allOut = new File(exportDir, "decompiled_all.c");

        DecompInterface ifc = new DecompInterface();
        DecompileOptions options = new DecompileOptions();
        ifc.setOptions(options);
        ifc.openProgram(currentProgram);

        ConsoleTaskMonitor monitor = new ConsoleTaskMonitor();
        int count = 0;
        try (PrintWriter allW = new PrintWriter(new FileWriter(allOut))) {
            for (Function func : fm.getFunctions(true)) {
                try {
                    DecompileResults res = ifc.decompileFunction(func, 60, monitor);
                    if (res == null || !res.decompileCompleted()) {
                        continue;
                    }
                    String code = res.getDecompiledFunction() != null
                            ? res.getDecompiledFunction().getC()
                            : null;
                    if (code == null) {
                        continue;
                    }
                    String addrStr = func.getEntryPoint().toString().replace(":", "_");
                    String safeName = func.getName().replaceAll("[^A-Za-z0-9_]", "_");
                    File fnFile = new File(decompDir, addrStr + "_" + safeName + ".c");
                    try (PrintWriter fw = new PrintWriter(new FileWriter(fnFile))) {
                        fw.print(code);
                    }

                    allW.println();
                    allW.println("// ==== " + func.getName() + " @ " + func.getEntryPoint() + " ====");
                    allW.print(code);
                    count++;
                } catch (Exception e) {
                    println("[export] decompile 失敗 " + func.getName() + ": " + e.getMessage());
                }
            }
        } finally {
            ifc.dispose();
        }
        println("[export] decompiled_all.c / decompiled/*.c 寫入 " + count + " 個函式");
    }

    private static String csvEscape(String s) {
        if (s == null) {
            return "";
        }
        String escaped = s.replace("\"", "\"\"");
        boolean needsQuote = escaped.contains(",") || escaped.contains("\"")
                || escaped.contains("\n") || escaped.contains("\r");
        if (needsQuote) {
            return "\"" + escaped + "\"";
        }
        return escaped;
    }
}
