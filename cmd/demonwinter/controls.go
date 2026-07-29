package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/demon_winter_cht/internal/game"
)

// controlMode 只改玩家如何把按鍵翻成探索指令；移動、碰撞、事件、時間與
// 戰鬥規則仍走同一條既有路徑。
type controlMode uint8

const (
	controlsModern controlMode = iota
	controlsRetro
)

func parseControlMode(s string) (controlMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "modern":
		return controlsModern, nil
	case "retro":
		return controlsRetro, nil
	default:
		return controlsModern, fmt.Errorf("只接受 modern 或 retro，收到 %q", s)
	}
}

func (m controlMode) next() controlMode {
	if m == controlsModern {
		return controlsRetro
	}
	return controlsModern
}

func (m controlMode) labelKey() string {
	if m == controlsRetro {
		return "controls.mode.retro"
	}
	return "controls.mode.modern"
}

func turnFacing(f game.Facing, delta int) game.Facing {
	return game.Facing((int(f) + delta + 4) % 4)
}
