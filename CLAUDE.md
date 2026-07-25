# Demon's Winter SSI 冬之魔 中文化

## ⚡ 動手前先讀 CONTEXT.md

**[`CONTEXT.md`](./CONTEXT.md) 是全專案的單一入口。**
對話被壓縮、或新 session 接手時，先讀它就能重建完整全局：
現況一覽、全部文件索引、術語表、oracle 優先序、**已被推翻的斷言清單**、工作紀律。

累積的逆向筆記已超過 7,700 行（`docs/re/`、`docs/formats/`、`docs/spec/`），
不要憑印象重推 —— 先查 CONTEXT.md 的索引。

## 原始目標

1. 我想透過反組譯還原冬之魔遊戲引擎, 建立類似 scummvm-like VM bytes code, 建立冬之魔引擎  
 - 希望在 golang /ebitain 環境下可以還原執行
 - 音樂 / 音效 都要還原
2. 基於先前 VM bytecode引擎 完成 中文化, 讓界面與 message 都能用中文顯示, 讓經典用中文說明

請先評估建立 PLAN.md , 簡易的工作請安排 subagent 執行

> **兩處已依實際發現修正**（詳見 `CONTEXT.md` §5）：
> - **這遊戲沒有 bytecode VM**。`DEMON.INT` 是原生 8086 機器碼，`.INT` 只是 SSI 的
>   命名慣例。控制流全寫死在機器碼裡，要做的是「事件表直譯器」而非 VM。
> - **原版沒有配樂**，只有 PC speaker 音效（AdLib/MIDI 在反組譯中零命中）。
>   「音樂還原」的正確範圍是忠實重現音效序列，不是自製配樂。

## 工作方式：SDD

反組譯 → 收攏成規格（`docs/spec/`）→ 才實作。
**只有標 READY 的規格可以動手。** 目前刻意還沒開始寫引擎本體。

每一輪都要：更新 markdown → **清掉被推翻的斷言** → commit + push → 更新 CONTEXT.md 現況。

# 遊戲 image
@./'Demons Winter (1988).zip'

# github repo
https://github.com/wicanr2/demon_winter_cht.git

# 工作目錄
@./workplace
