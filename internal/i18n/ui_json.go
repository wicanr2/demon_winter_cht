package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// UICatalogJSON 是 remake 自有介面文字的資料格式。原版索引型散文仍使用
// 易於校稿的純文字 catalog；沒有 legacy 索引的玩家介面改用 JSON，讓引擎
// 只保留穩定 key，不再在 Go 裡藏一份中文 fallback。
type UICatalogJSON struct {
	Locale         string                         `json:"locale"`
	CommandLayouts map[string]UICommandLayoutJSON `json:"commandLayouts,omitempty"`
	Entries        []UIJSONEntry                  `json:"entries"`
}

type UIJSONEntry struct {
	Key  string `json:"key"`
	EN   string `json:"en,omitempty"`
	Text string `json:"text"`
}

type UICommandLayoutJSON struct {
	Retro  []string             `json:"retro"`
	Groups []UICommandGroupJSON `json:"groups"`
}

type UICommandGroupJSON struct {
	TitleKey string   `json:"titleKey"`
	Column   int      `json:"column"`
	Items    []string `json:"items"`
}

func LoadUICatalogJSON(path string) (*UICatalogJSON, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c UICatalogJSON
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("解析 UI JSON %s：%w", path, err)
	}
	seen := map[string]bool{}
	for i, e := range c.Entries {
		if e.Key == "" {
			return nil, fmt.Errorf("UI JSON 第 %d 筆缺少 key", i+1)
		}
		if seen[e.Key] {
			return nil, fmt.Errorf("UI JSON 重複 key %q", e.Key)
		}
		if e.Text == "" {
			return nil, fmt.Errorf("UI JSON 的 %q 沒有 text", e.Key)
		}
		seen[e.Key] = true
	}
	for name, layout := range c.CommandLayouts {
		if name == "" {
			return nil, fmt.Errorf("UI JSON 有空白 command layout 名稱")
		}
		retro := map[string]bool{}
		for _, key := range layout.Retro {
			if !seen[key] {
				return nil, fmt.Errorf("command layout %q 引用不存在的 key %q", name, key)
			}
			if retro[key] {
				return nil, fmt.Errorf("command layout %q 的 retro 重複 key %q", name, key)
			}
			retro[key] = true
		}
		grouped := map[string]bool{}
		for _, group := range layout.Groups {
			if !seen[group.TitleKey] {
				return nil, fmt.Errorf("command layout %q 引用不存在的 titleKey %q",
					name, group.TitleKey)
			}
			if group.Column < 0 || group.Column > 1 {
				return nil, fmt.Errorf("command layout %q 的 column %d 超出 0–1",
					name, group.Column)
			}
			for _, key := range group.Items {
				if !retro[key] {
					return nil, fmt.Errorf("command layout %q 的分組 key %q 不在 retro 清單",
						name, key)
				}
				if grouped[key] {
					return nil, fmt.Errorf("command layout %q 的分組重複 key %q", name, key)
				}
				grouped[key] = true
			}
		}
		if len(grouped) != len(retro) {
			return nil, fmt.Errorf("command layout %q：retro 有 %d 項，分組只有 %d 項",
				name, len(retro), len(grouped))
		}
	}
	return &c, nil
}

func WriteUICatalogJSON(path, locale string, c *Catalog) error {
	out := &UICatalogJSON{Locale: locale}
	for _, e := range c.Entries {
		if e.Name == "" || !e.Translated() {
			continue
		}
		out.Entries = append(out.Entries, UIJSONEntry{
			Key: e.Name, EN: e.Source, Text: e.Target,
		})
	}
	sort.Slice(out.Entries, func(i, j int) bool {
		return out.Entries[i].Key < out.Entries[j].Key
	})
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
