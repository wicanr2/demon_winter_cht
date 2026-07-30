# 不要再生成同一首「史詩管弦樂」：經典 RPG remake 的配樂提示詞方法

> 適用範圍：1980–1990 年代 D&D 類桌上角色扮演、Gold Box、迷宮探索、
> 回合制戰鬥與西方奇幻電腦 RPG 的 remake、中文化與宣傳片。
>
> 本文談的是如何把遊戲研究轉成可執行的作曲提示詞。它不鼓勵模仿特定遊戲、
> 電影或作曲家的可辨識風格，也不把生成平台的輸出權利視為理所當然。

## 為什麼「D&D、史詩、管弦樂」幾乎一定失敗

把下面這句丟給音樂生成模型：

```text
Epic D&D fantasy RPG soundtrack, cinematic orchestra, powerful choir.
```

通常會得到沒有遊戲身分的「預告片音樂」：低弦持續音、戰鼓漸強、銅管三連音、
混聲合唱、巨大撞擊。單獨聽或許有氣勢，放進十款 remake 卻像同一首曲子換標題。

問題不是模型不知道「史詩」，而是提示詞只告訴它**市場分類**，沒有告訴它：

- 這個世界由什麼材質與文化構成；
- 玩家此刻在做什麼；
- 音樂應如何隨玩法改變；
- 哪個短動機代表這款遊戲；
- 什麼聲音絕對不屬於它；
- 這是 1988 年原音還是 2026 年新編曲。

Udio 的官方指南建議明確提供情緒、類型、速度與樂器；其「積木法」
（Brick Method）再把提示拆成類型、聲音特徵、用途等區塊。Suno 的官方音樂詞彙
也強調以樂器密度與曲式詞彙控制結構。這些是必要條件，但對 remake 還不夠：
我們還必須把**遊戲考古證據**放進提示詞。

## 先寫音樂設計簡報，不要先寫提示詞

每個專案先完成下面十格。答不出來的格子就是研究缺口，不要用「史詩感」掩蓋。

| 欄位 | 要回答的問題 | 《冬之魔》示例 |
|---|---|---|
| 年代身分 | 原作年份與聲音硬體是什麼？ | 1988 DOS、PC speaker、原版無 BGM |
| 核心行動 | 玩家最常做的三件事？ | 世界探索、管理五人隊伍、戰術戰鬥 |
| 世界材質 | 能聽見哪些自然或建築材質？ | 寒風、凍土、石造神殿、海浪 |
| 文化聲色 | 哪些樂器能暗示地域／勢力？ | 低弦與框鼓；惡魔方使用低銅管與金屬 |
| 原作聲音 DNA | 有沒有可驗證的短旋律或音效？ | 11 音陣亡旋律、八個 C 大調方波單音 |
| 主題動機 | 3–7 個音如何代表本作？ | B–A–B–C–G–C 的輪廓 |
| 互動狀態 | 音樂必須支援哪些狀態？ | 探索、危險、戰鬥、魔王、勝敗 |
| 技術輸出 | 線性影片還是遊戲內循環？ | 宣傳片 72 秒；遊戲內則要分 stem／loop |
| 誠實標示 | 哪些是原作，哪些是新作？ | 「宣傳片專用編曲；原版沒有 BGM」 |
| 禁用清單 | 哪些俗套會破壞身分？ | 通用紫金奇幻、EDM drop、全程合唱 |

這一步的價值是把「我想要 D&D 音樂」改寫成：

> 我需要一段能陪伴某種玩法、屬於某個世界、可被某套音訊系統切換，而且不會
> 偽裝成原版資產的音樂。

## 可重用的提示詞七層結構

一份穩定提示詞依序寫七層：

```text
1. 用途與長度
2. 世界／年代身分
3. 玩家狀態與情緒弧線
4. 主題動機
5. 配器及每件樂器的動作
6. 曲式、循環、轉場與混音交付
7. 禁用項與權利標示
```

### 1. 用途與長度

「遊戲音樂」仍太寬。要寫：

- 90 秒可無縫循環的戶外探索；
- 45 秒城鎮日間循環，另交付 2 秒入城短句；
- 三層可疊加戰鬥 stem；
- 72 秒固定時間點的宣傳片；
- 6 秒魔王現身轉場。

### 2. 世界／年代身分

不要只寫 fantasy。從遊戲本身挑三個名詞：

```text
冰封群島／海港木材／異教石造神殿
沙漠商路／青銅天文儀／失落星圖
邊境泥濘／破損軍旗／低魔法傭兵團
```

若要有復古感，也要說清楚是：

- 真正硬體限制的 chiptune；
- 現代管弦重編，但保留原作短動機；
- 現代音色，節奏與和聲向 1980 年代電腦 RPG 致意；
- 只在音效層保留 PC speaker／FM 合成器。

### 3. 玩家狀態與情緒弧線

遊戲配樂不是場景桌布。Berklee 對遊戲作曲工作的說明強調，音樂需對探索、戰鬥、
解謎與勝利等狀態立即反應；同一曲目要能拆成循環、片段或生成式素材。

提示詞應使用動詞：

```text
低弦維持警戒但不催促玩家
木管在發現道路時短暫向上回答
戰鼓只在敵人進入視野後加入
戰鬥結束時保留和聲，不要播放永久勝利終止式
```

「dark, mysterious」描述情緒；「戰鼓在敵人出現後加入」才描述遊戲行為。

### 4. 主題動機

每個 remake 至少要有一個自己的 3–7 音動機。來源優先序：

1. 原作可合法使用、已反組譯確認的旋律；
2. 標題畫面或重要音效的節奏輪廓；
3. 角色姓名、地圖形狀或故事選擇轉成的自創音型；
4. 完全新作，但在專案文件中記錄音高與用途。

不要只說「memorable leitmotif」。要寫實際行為：

```text
六音動機 B–A–B–C–G–C；探索時由大提琴低八度、每音拉長，
戰鬥時縮成兩拍節奏，魔王現身時由法國號提高八度完整回答。
```

若生成平台不穩定接受音名，先用 MIDI 或人工哼唱提供動機，再要求延伸；不能依靠
文字模型猜出相同旋律。

### 5. 配器不是清單，要說每件樂器做什麼

較弱：

```text
strings, brass, choir, drums
```

較強：

```text
低音提琴維持稀疏五度持續音；大提琴演奏六音主題；
框鼓以不規則弱拍模仿長途行軍；法國號只在地標揭露時出現；
混聲合唱在魔王階段才進入，沒有歌詞，不從開頭鋪滿。
```

樂器一多，模型容易把所有聲部同時開滿。每個聲部都要有**進場時間、角色與密度**。

### 6. 曲式、互動與交付

Wwise 官方文件把互動音樂大致分為：

- **水平重排**：探索 A → 危險橋段 → 戰鬥 B；
- **垂直分層**：同一時間軸上增加打擊、銅管或合唱層。

實際專案常混合兩者。即使還沒有 Wwise／FMOD，提示詞也應要求分軌：

```text
交付相同長度、相同起點的四個 stem：
1 ambience（風與環境）
2 harmony（低弦與和聲）
3 pulse（節奏與打擊）
4 threat（銅管、合唱與衝擊）
每軌皆可單獨循環；任意相加不得削波；循環點前後不可有殘響斷裂。
另交付 2 秒 danger、victory、defeat stinger。
```

對宣傳片則要寫時間碼：

```text
0–8 秒只有環境；42 秒開始加速；57 秒全編制驟停；
64 秒最大重拍；69–72 秒淡出但保留一次辨識音效。
```

### 7. 禁用項與權利標示

禁用項不是負面情緒宣洩，而是防止類型平均化：

```text
不要 EDM drop；不要電吉他；不要全程 ostinato；不要從第一秒就有合唱；
不要好萊塢大調英雄終止；不要使用其他遊戲、電影或作曲家的可辨識旋律；
不要模仿具名在世作曲家。
```

remake 還要加：

```text
輸出標示為「remake 新作配樂／宣傳片專用編曲」；
不得宣稱為原版未公開音軌。
```

## 六種 RPG 場景提示詞模板

方括號內容由各專案的音樂設計簡報填入。英文通常較容易被現有音樂平台解析，
但專案內應同時保存繁體中文意圖。

### 戶外探索

```text
Instrumental seamless exploration loop, [90] seconds. A [年代／硬體] fantasy RPG
world defined by [三個世界材質]. Mood: patient discovery with distant danger, never
urgent. [主奏樂器] states a [音數] note motif [音型或輪廓] once every [秒數];
[低音樂器] sustains open fifths; [環境／文化樂器] answers sparsely. No full drum
kit; percussion enters only as a faint travel pulse after the midpoint. Deliver a
clean loop with a two-second reverb-safe tail and separate ambience/harmony stems.
Avoid heroic major cadences, constant ostinato, choir, EDM, and recognizable themes.
```

### 城鎮

```text
Instrumental town loop for a [港口／邊境／沙漠／地下] settlement, [60–90] seconds.
The town feels [安全但貧困／繁忙但受威脅／神聖但腐敗]. [兩件小編制樂器] trade
short phrases over [節奏樂器] at [BPM] BPM. Keep the arrangement intimate enough for
dialogue and menu sounds. Include an eight-bar A section, contrasting B section, and
return without a final cadence. No tavern cliché unless the actual town is a tavern;
no large orchestra, no choir, no trailer drums.
```

### 地城

```text
Dark dungeon ambience-music hybrid, seamless [90] second loop. The space is made of
[石材／水／金屬／生物質], and the danger is [宗教／亡靈／機械／惡魔]. Use [低音
樂器] as irregular breath-like swells, isolated [敲擊音色] with long non-rhythmic
decay, and fragments of the game motif with missing final notes. Preserve silence:
no more than [聲部數] simultaneous layers. No steady combat beat, no jump-scare
every eight bars, no generic horror riser.
```

### 一般戰鬥

```text
Layered turn-based RPG combat cue, [96] BPM, [60] second seamless loop. Tactical,
decisive, and readable rather than frantic. Strings perform a syncopated reduction of
[主題動機]; low drums mark action decisions, not every beat; brass answers only at
round boundaries. Deliver four synchronized stems: pulse, harmony, threat, and impact.
The loop must sustain repeated battles without listener fatigue. Avoid nonstop cymbal
crashes, double-time trailer ostinato, power metal, and permanent maximum intensity.
```

### 魔王決戰

```text
Final boss music for [魔王身分與核心矛盾], [長度／循環需求]. Begin with the player's
established motif in [熟悉樂器], then let the antagonist distort it through [音程、
節拍或配器變形]. Reserve choir and low brass for phase [數字]; create a genuine
[1–3] second void before the irreversible decision or phase transition. The climax
should sound costly, not automatically victorious. End on [未解決／悲劇／勝利但有代價]
harmony. No generic Latin syllables, no borrowed liturgical text, no constant wall of
sound, no recognizable franchise themes.
```

### 宣傳片

```text
[60–75] second instrumental RPG remake trailer score synchronized to picture.
0–[A] historical restraint; [A]–[B] world reveal; [B]–[C] gameplay acceleration;
[C]–[D] silence before the decision; [D] climax; final [秒數] unresolved logo tail.
Build the entire score from the project's [主題動機／原作音效節奏], transformed across
[低弦、文化樂器、銅管等]. Leave spectral space for dialogue and authentic game SFX.
Deliver full mix plus music-only and four stems. Target [LUFS] integrated loudness,
true peak below [-1 dBTP]. Label as new remake/trailer music, not original soundtrack.
```

## 讓每款 remake 聽起來不同：五軸差異化

新專案不得直接複製上一款的 prompt，只能複製七層結構。至少改變下列五軸中的
三軸：

| 軸 | 問題 | 可選方向 |
|---|---|---|
| 音色地理 | 世界由什麼材料與文化發聲？ | 木、冰、沙、銅、海、機械、宗教空間 |
| 節奏倫理 | 玩家是征服、求生、調查還是遠征？ | 堅定行軍、不規則呼吸、儀式循環、航海擺動 |
| 主題來源 | 什麼只屬於這款遊戲？ | 原作音效、標題旋律、故事密語、自創音型 |
| 技術年代 | 如何承認原作硬體？ | 方波點音、FM 音色、MIDI 配器、純現代聲學 |
| 情緒代價 | 勝利究竟意味什麼？ | 凱旋、倖存、悲劇、未解、道德不安 |

例如兩款都是「黑暗奇幻」：

- 冰封航海 RPG：開放五度、框鼓、寒風、低銅管，節奏像划槳；
- 地下宗教 RPG：狹窄半音、無拍長音、石室殘響、低聲無詞合唱；
- 邊境傭兵 RPG：乾燥小鼓、中提琴、木笛、短促不對稱樂句。

三者不需要引用任何具名作曲家，也能建立清楚差異。

## 生成平台不是授權捷徑

生成之前先保存：

- 平台、模型版本與日期；
- 完整 prompt、seed、延伸／重混鏈；
- 使用的方案層級；
- 當日服務條款與輸出授權；
- 人工修改的 MIDI、音訊工程與 stem；
- 原作動機的權利依據與專案持有人的決定。

以 Suno 目前官方說明為例，免費方案與付費方案的輸出用途不同，而且後來訂閱不會
自動回溯授權先前免費生成的歌曲。這類條款會變動，不能把本文寫死成永久法律結論。
公開或商業發行前必須重新查當日條款。若專案需要最可控的來源，優先使用：

1. 人工作曲／委製並簽清楚授權；
2. 專案自行撰寫 MIDI、以可再散布 SoundFont 或自有音源渲染；
3. 已核實授權的音樂生成服務與付費方案；
4. 明確允許該用途的曲庫。

## 驗收：好聽之外還要能用

每首至少通過：

- **身分盲測**：不看檔名，能否說出它屬於哪個世界，而非只說「奇幻」？
- **狀態測試**：探索、危險、戰鬥三段是否真的不同，還是只有音量變大？
- **疲勞測試**：循環三次後，固定高頻打擊或 ostinato 是否令人煩躁？
- **對白測試**：繁中旁白與 UI 音效是否仍清楚？
- **循環測試**：頭尾波形、殘響、節拍是否無跳點？
- **分層測試**：任意 stem 組合是否同拍、無削波、無和聲衝突？
- **真實性測試**：原版聲音與 remake 新作是否標示清楚？
- **權利測試**：能否提出來源、方案、條款快照及專案使用依據？

## 《冬之魔》如何套用

《冬之魔》的獨特答案不是「更大的管弦樂」，而是：

- 原版**沒有 BGM**，安靜本身就是歷史事實；
- PC speaker 的乾燥方波是原作聲音身分；
- `B–A–B–C–G–C` 來自已反組譯的陣亡旋律輪廓；
- 寒風、凍土、航海與惡魔神殿構成世界材質；
- 澤瑞斯之戰不是單純凱旋，而是「殺了他，毀滅也不會結束」的決斷。

因此宣傳片應先保留大量空間，讓原作方波穿透；低弦逐步延伸六音動機，戰鼓到中後段
才進場，魔王台詞前必須抽空，決戰和聲不能以明亮大調完全解決。這些限制共同構成
《冬之魔》的聲音，而不是在通用奇幻模板上更換片名。

## 來源與延伸閱讀

- [Berklee：遊戲作曲需因應探索、戰鬥等狀態，並製作 loops、chunks 與主題](https://www.berklee.edu/careers/roles/composer-video-games)
- [Berklee：互動媒體作曲包含 RPG 類型、動態配器與音樂敘事](https://college.berklee.edu/courses/gaim-341)
- [Berklee：互動遊戲配樂與可變性](https://online.berklee.edu/takenote/scoring-for-games-top-techniques-for-composing-music-for-interactive-media/)
- [Audiokinetic Wwise：建立互動音樂](https://www.audiokinetic.com/en/public-library/2025.1.4_9062/?id=creating_interactive_music&source=Help)
- [Audiokinetic Wwise：水平與垂直互動音樂結構](https://www.audiokinetic.com/en/public-library/2025.1.3_9039/?id=interactive_music_project_structure&source=Help)
- [Udio 官方提示指南](https://help.udio.com/en/articles/10716541-prompt-like-a-master)
- [Udio 官方積木提示法](https://help.udio.com/en/articles/12232112-the-brick-method-making-udio-work-for-you)
- [Suno 官方音樂詞彙與曲式提示](https://help.suno.com/en/articles/9010177)
- [Suno 官方輸出權利說明](https://help.suno.com/en/articles/2746945)
- [Suno：付費方案不回溯授權免費方案舊作品](https://help.suno.com/en/articles/2425729)

