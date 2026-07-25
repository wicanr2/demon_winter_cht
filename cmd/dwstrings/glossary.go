package main

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// 譯名一致性掃描。
//
// 就算發了統一譯名表，多人／多 agent 分頭翻仍然會漂移 ——
// 同一個人名出現三種音譯是這類專案最常見的品質破口，而且逐條肉眼看不出來
// （每一條單看都通順）。這裡把它變成可以自動擋下來的事。
//
// 只掃**專有名詞**：人名、地名、神名、怪物名。一般名詞的譯法本來就該隨語境調整，
// 拿它們比對只會製造大量假警報。

// glossaryTerm 是一條譯名。
type glossaryTerm struct {
	en string
	zh string
}

// properNounSections 是會收專有名詞的章節標題關鍵字。
var properNounSections = []string{"專有名詞", "神祇", "城鎮", "主要地點", "怪物", "NPC", "劇情道具"}

// notProperNounSections 蓋掉上面誤中的章節。
//
// 「### 城鎮設施」含「城鎮」但收的是介面指令（Pray、Heal…），
// 那些詞在敘述裡是普通動詞，拿去比對只會製造假警報。
var notProperNounSections = []string{"設施", "指令", "選單"}

var (
	headingRe = regexp.MustCompile(`^#{2,3}\s+(.*)$`)
	rowRe     = regexp.MustCompile(`^\|([^|]+)\|([^|]+)\|`)
)

// loadGlossary 讀出譯名表裡的專有名詞。
func loadGlossary(path string) ([]glossaryTerm, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []glossaryTerm
	inSection := false

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Text()

		if m := headingRe.FindStringSubmatch(line); m != nil {
			inSection = false
			for _, k := range properNounSections {
				if strings.Contains(m[1], k) {
					inSection = true
					break
				}
			}
			for _, k := range notProperNounSections {
				if strings.Contains(m[1], k) {
					inSection = false
					break
				}
			}
			continue
		}
		if !inSection {
			continue
		}

		m := rowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		en := strings.TrimSpace(m[1])
		zh := strings.TrimSpace(m[2])
		if en == "" || zh == "" || en == "原文" || strings.HasPrefix(en, "-") {
			continue
		}
		// 「Xxx（說明）」這種帶註記的原文欄取括號前的部分。
		if i := strings.IndexAny(en, "（("); i > 0 {
			en = strings.TrimSpace(en[:i])
		}
		// 「A / B」列多個詞。兩欄的項數相同時逐項配對
		// （`Fire dragon / Ice dragon` 對 `火龍／冰龍`），
		// 否則視為同一個詞的多種拼法，全部指向同一個譯名。
		ens, zhs := splitAlternatives(en), splitAlternatives(zh)
		for i, alt := range ens {
			if !isProperNoun(alt) {
				continue
			}
			target := zhs[0]
			if len(zhs) == len(ens) {
				target = zhs[i]
			}
			out = append(out, glossaryTerm{en: alt, zh: target})
		}
	}
	return out, sc.Err()
}

func splitAlternatives(s string) []string {
	s = strings.ReplaceAll(s, "／", "/")
	var out []string
	for _, p := range strings.Split(s, "/") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// isProperNoun 判斷是否為值得掃描的專有名詞。
//
// 判準：大寫開頭、至少四個字母、不含空白以外的符號。
// 這會漏掉少數多字專名（那些本來就不容易誤譯），但幾乎不會誤報。
func isProperNoun(s string) bool {
	if len(s) < 4 || s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	for _, r := range s {
		if r >= 0x80 {
			return false
		}
		if !(r == ' ' || r == '-' || r == '\'' ||
			(r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return false
		}
	}
	return true
}

// wordBoundary 建一個帶詞界的比對式，避免 `Orc` 命中 `Sorcerer`。
//
// **大小寫敏感。** 專有名詞在原文裡一律大寫開頭，普通名詞不是 ——
// 這一條就能分開「the Skeleton attacks」（怪物名，要照譯名表）與
// 「the skeletons of demons」（骸骨，一般名詞，譯法隨語境）。
// 寧可漏抓幾個，也不要讓假警報淹掉真正的譯名漂移。
func wordBoundary(term string) *regexp.Regexp {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(term) + `s?\b`)
}
