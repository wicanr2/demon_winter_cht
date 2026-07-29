《Demon's Winter 冬之魔》繁體中文 remake

本發行包只包含乾淨重寫的引擎、繁中翻譯與專案自製文件。
基於著作權與字型授權，它不包含：

  1. 原版遊戲資料檔；
  2. 倚天 16×15 字型。

啟動前請準備自己的合法 DOS 版資料目錄，以及含 STDFONT.15、
SPCFONT.15 的倚天字型目錄：

  ./demonwinter -data /path/to/DEM_DATA -eten /path/to/etan_font

可用 -video ega、-video cga 或 -video modern 選開場主題；遊戲中 F8
依序切換三套主題。F2 開手札，F10 離開並詢問是否存檔。

開發者可用：

  ./demonwinter -list-scenes
  ./demonwinter -scene armory -seed 11 \
      -data /path/to/DEM_DATA -eten /path/to/etan_font

Linux 執行環境需要 X11、OpenGL 與 ALSA 相容函式庫。若執行檔無法啟動，
可在原始碼目錄使用專案提供的 Docker Go 工具鏈自行編譯。

完整說明、還原差異與研究文件見 README.md。
