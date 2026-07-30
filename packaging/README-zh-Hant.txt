《Demon's Winter 冬之魔》繁體中文 remake

本發行包只包含乾淨重寫的引擎、繁中翻譯與專案自製文件。
基於著作權與字型授權，它不包含：

  1. 原版遊戲資料檔；
  2. 倚天 16×15 字型。

啟動前請準備自己的合法 DOS 版資料目錄，以及含 STDFONT.15、
SPCFONT.15 的倚天字型目錄：

Linux／macOS：

  ./demonwinter -data /path/to/DEM_DATA -eten /path/to/etan_font

Windows（PowerShell）：

  .\demonwinter.exe -data C:\path\to\DEM_DATA -eten C:\path\to\etan_font

可用 -video ega、-video cga 或 -video modern 選開場主題；遊戲中 F8
依序切換三套主題。Modern Icon 自製素材已隨包附上，無須另外指定目錄。
F1 開說明，F6 切換復古／現代操作，F10 離開並詢問是否存檔。

開發者可用：

  ./demonwinter -list-scenes
  ./demonwinter -scene armory -seed 11 \
      -data /path/to/DEM_DATA -eten /path/to/etan_font

Windows 請把上述命令的 `./demonwinter` 改成 `.\demonwinter.exe`。
Linux 執行環境需要 X11、OpenGL 與 ALSA 相容函式庫。macOS 第一次啟動時
若 Gatekeeper 阻擋未簽章的社群建置，請在「系統設定 → 隱私權與安全性」
確認你下載的檔案後允許開啟。

完整說明、還原差異與研究文件見 README.md。
