package scenario

import (
	"fmt"
	"os"
	"path/filepath"
)

// 存檔寫回。
//
// 策略是**改寫而不是重建**：從 decode 時保留的原始 bytes 出發，只把已知欄位
// 蓋回去，其餘一個 byte 都不動。
//
// 理由是這份格式還有一整片未解區域（`Unknown0D`、10 個道具槽的內部語意、
// trailer 大半）。重建的話那些 bytes 只能填 0，存檔拿回原版就壞了，
// 而且壞在看不見的地方 —— 遊戲不會報錯，只會有某些狀態悄悄消失。
//
// 驗收標準：**讀出來再寫回去，byte-for-byte 相同**。
// 這條過不了就代表某個欄位的 encode 與 decode 不對稱，先修那個再談其他。

// Encode 把存檔編回 PARTY.DAT 的位元組內容。
func (s *SaveGame) Encode() ([]byte, error) {
	out := make([]byte, 0, trailerStart+trailerLen)

	for i := range s.Characters {
		rec, err := s.Characters[i].encode()
		if err != nil {
			return nil, fmt.Errorf("scenario: 第 %d 名角色編碼失敗: %w", i+1, err)
		}
		out = append(out, rec...)
	}

	trailer := s.TrailerRaw
	putByte(trailer[:], positionXOffset, s.PositionX)
	putByte(trailer[:], positionYOffset, s.PositionY)
	putByte(trailer[:], facingOffset, s.Facing)
	putByte(trailer[:], timeCounterOffset, s.TimeCounter)
	putByte(trailer[:], unknown9COffset, s.Unknown9C)
	putByte(trailer[:], hourOffset, s.Hour)
	putByte(trailer[:], dayOffset, s.Day)
	putByte(trailer[:], monthOffset, s.Month)
	putByte(trailer[:], mapIDOffset, s.MapID)
	putByte(trailer[:], lightSourceOffset, s.LightSource)
	putByte(trailer[:], merchantBaseOffset, s.MerchantBase)
	putByte(trailer[:], partySizeOffset, s.PartySize)
	putByte(trailer[:], rationsOffset, s.Rations)
	encodeShips(trailer[:], s.Ships)
	putBool(trailer[:], viewedLandTodayOffset, s.ViewedLandToday)
	putByte(trailer[:], boatOffset, s.Boat)
	putByte(trailer[:], unknownC1Offset, s.UnknownC1)
	copy(trailer[formationOffset:], s.Formation[:])
	putByte(trailer[:], unknown09Offset, s.Unknown09)
	copy(trailer[ldFlagsOffset:], s.LDFlags[:])
	putLE4(trailer[:], goldOffset, s.Gold)

	out = append(out, trailer[:]...)

	if len(out) != trailerStart+trailerLen {
		return nil, fmt.Errorf("scenario: 編碼後長度 %d，預期 %d",
			len(out), trailerStart+trailerLen)
	}
	return out, nil
}

// encode 把一名角色編回 260 bytes。
func (c *Character) encode() ([]byte, error) {
	if len(c.Raw) != recordLen {
		return nil, fmt.Errorf("原始記錄長度 %d，預期 %d", len(c.Raw), recordLen)
	}
	rec := make([]byte, recordLen)
	copy(rec, c.Raw)

	putName(rec, c.Name)

	putByte(rec, levelOffset, c.Level)
	putByte(rec, raceOffset, c.RaceByte)
	putByte(rec, classOffset, c.ClassByte)
	copy(rec[skillFlagsOffset:], c.SkillFlags[:])

	// 道具槽：從原始 bytes 出發，把已解欄位蓋回去。
	//
	// **換上另一件東西就整格重造。** 沿用舊 bytes 會讓前一件的次數、附魔、
	// 武器特效黏在新道具身上。
	//
	// 清空則相反 —— 只寫 0xFF，其餘留原樣，照原版的作法
	// （兩份原版存檔裡交出去的那幾格都還留著前一件的殘值）。
	for i, slot := range c.InventorySlotsRaw {
		if len(slot) != inventorySlotLen {
			continue
		}
		raw := make([]byte, inventorySlotLen)
		if it := c.Inventory[i]; !it.Empty() && it.Type != slot[slotType] {
			raw = newSlotRaw()
		} else {
			copy(raw, slot)
		}
		c.Inventory[i].encodeInto(raw)
		copy(rec[inventoryStart+i*inventorySlotLen:], raw)
	}

	putLE4(rec, expOffset, c.Experience)

	attr := func(off int, v byte) { putByte(rec, expOffset+off, v) }
	attr(attrStrengthNaturalOffset, c.StrengthNatural)
	attr(attrSkillNaturalOffset, c.SkillNatural)
	attr(attrMaxSPNaturalOffset, c.MaxSPNatural)
	attr(attrSpeedNaturalOffset, c.SpeedNatural)
	attr(attrSpeedBonusOffset, c.SpeedBonus)
	attr(attrStrengthBonusOffset, c.StrengthBonus)
	attr(attrIntellectOffset, c.Intellect)
	attr(attrEnduranceOffset, c.Endurance)
	attr(attrSkillBonusOffset, c.SkillBonus)
	attr(attrMaxHPOffset, c.MaxHP)
	attr(attrCurrentHPOffset, c.CurrentHP)
	attr(attrMaxSPBonusOffset, c.MaxSPBonus)
	attr(attrCurrentSPOffset, c.CurrentSP)

	putBool(rec, identifiedTodayOffset, c.IdentifiedToday)
	putBool(rec, worshipedTodayOffset, c.WorshipedToday)
	putBool(rec, exorcisedTodayOffset, c.ExorcisedToday)

	putByte(rec, prayChanceOffset, c.PrayChance)
	putByte(rec, bindLevelOffset, c.BindLevel)
	putByte(rec, deityOffset, c.Deity)

	putByte(rec, weaponSlotOffset, c.WeaponSlotIndex)
	putByte(rec, armorSlotOffset, c.ArmorSlotIndex)
	putByte(rec, combatFlagsOffset, byte(c.CombatStatus))
	putByte(rec, unknown103Offset, c.Unknown103)

	return rec, nil
}

// putName 寫入 NUL 結尾的姓名。
//
// **只覆蓋到字串本身與那個 NUL，後面的 bytes 原封不動。**
// 原版在姓名欄後半留了什麼沒人知道；把整段清成 0 是「看起來比較乾淨」
// 的破壞，會讓存檔與原版不再等價。
func putName(rec []byte, name string) {
	b := []byte(name)
	if len(b) > nameFieldLen-1 {
		b = b[:nameFieldLen-1]
	}
	copy(rec, b)
	rec[len(b)] = 0
}

// putBool 把旗標寫成 0／1。
func putBool(buf []byte, off int, v bool) {
	var x byte
	if v {
		x = 1
	}
	putByte(buf, off, x)
}

func putByte(buf []byte, off int, v byte) {
	if off >= 0 && off < len(buf) {
		buf[off] = v
	}
}

func putLE4(buf []byte, off, v int) {
	if off < 0 || off+4 > len(buf) {
		return
	}
	buf[off] = byte(v)
	buf[off+1] = byte(v >> 8)
	buf[off+2] = byte(v >> 16)
	buf[off+3] = byte(v >> 24)
}

// SaveTo 把存檔寫到指定路徑。
//
// **先寫暫存檔再改名。** 存檔寫到一半當掉會讓玩家連舊進度一起失去，
// 那比存檔失敗嚴重得多；改名在同一個檔案系統上是原子操作。
func (s *SaveGame) SaveTo(path string) error {
	data, err := s.Encode()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".party-*.tmp")
	if err != nil {
		return fmt.Errorf("scenario: 建立暫存檔失敗: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // 成功改名後這一步是 no-op

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("scenario: 寫入暫存檔失敗: %w", err)
	}
	// CreateTemp 給 0600，存檔沒有敏感內容，比照一般檔案的 0644。
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fmt.Errorf("scenario: 設定存檔權限失敗: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("scenario: 同步暫存檔失敗: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("scenario: 關閉暫存檔失敗: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("scenario: 換上新存檔失敗: %w", err)
	}
	return nil
}
