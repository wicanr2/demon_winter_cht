# 公式與數值工程索引

日期：2026-07-30

本頁提供公式的單一導覽；完整推導、位址與例外以連結文件為準。

| 系統 | 核心規則 | 證據 |
|---|---|---|
| 亂數 | DOS 浮點縮放與整數範圍等價；固定 seed 用於重播，不改正常遊戲分布 | [`docs/spec/01`](../spec/01-rng.md)、[`docs/re/14`](../re/14-rng-float-equivalence.md) |
| 命中／傷害 | 行動點、武器、護甲、姿勢、技能與傷害結算按 DOS 欄位順序處理 | [`docs/spec/02`](../spec/02-combat.md)、[`docs/re/06`](../re/06-combat-system.md)、[`docs/re/16`](../re/16-combat-details.md) |
| 法術量值 | 多數效果上限為 `K × 投入 SP ÷ M`；低於上限三分之一會重擲，形成偏向上段的分布 | [`docs/re/09`](../re/09-spells-and-actions.md)、[`docs/re/15`](../re/15-spell-constants.md) |
| 枯萎 | 速度、力量、技巧各自以目前值為上限重擲，最低為 3；不是固定扣值 | [`docs/re/18`](../re/18-jumptable-sweep.md)、`internal/game/spell_test.go` |
| 道具價值 | 基價、材質、附魔及效果共同決定；未知／保留欄位不發明用途 | [`docs/re/44`](../re/44-item-valuation.md)、[`docs/re/49`](../re/49-item-valuation-complete.md) |
| 附魔費用 | `(新附魔價值 − 舊附魔價值) × (20 − 材質) ÷ 10` | [`docs/re/102`](../re/102-enchant-cost-and-a-hand-rolled-pow.md) |
| 商人價格 | 商品價值再套設施／交涉路徑；原版 Seaside 匕首 2 Gold 已動態對拍 | [`docs/re/45`](../re/45-merchant-price.md)、[`docs/re/113`](../re/113-dosbox-price-and-battle-gold-oracle.md) |
| 掉寶 | 等級決定價格上限；效果最低法力若高於強度就整組重擲 | [`docs/re/30`](../re/30-loot-generation.md)、[`docs/re/48`](../re/48-loot-generator-reread.md) |
| 戰後獎勵 | 怪物資料決定 EXP 與金幣範圍；固定遭遇已與 DOS 結算量級對拍 | [`docs/re/56`](../re/56-battle-rewards.md)、[`docs/re/113`](../re/113-dosbox-price-and-battle-gold-oracle.md) |
| 時間 | DOS 日／月／年進位常數為 38／35／23；24 小時鐘與旅人之床另有獨立門檻 | [`docs/spec/06`](../spec/06-time.md)、[`docs/re/107`](../re/107-final-arena-time-and-stale-worklist-audit.md) |
| 海戰 | 航行點成本、砲擊命中／傷害、船體與 150–199 步遭遇重設均按原版 | [`docs/re/105`](../re/105-ida94-sea-combat.md) |
| 建角 | 初擲加兩次重擲，共三輪；一次可選多項屬性，6 是建議門檻而非限制 | [`docs/spec/05`](../spec/05-character.md)、`internal/game/create_test.go` |

所有整數除法均依 Go 正整數截斷語意實作；若原版可能出現負值或溢位，測試
必須另列 16-bit 邊界，不能只憑代數式推定。
