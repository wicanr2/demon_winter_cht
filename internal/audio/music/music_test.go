package music

import "testing"

func TestTracksAreNonEmptyPCM(t *testing.T) {
	for _, scene := range []Scene{Exploration, Sanctuary, Battle, Finale} {
		got := Render(scene, .2)
		if len(got) < SampleRate*2 || len(got)%2 != 0 {
			t.Errorf("scene %d PCM 長度不合法：%d", scene, len(got))
		}
	}
}

func TestSilentAndZeroVolume(t *testing.T) {
	if Render(Silent, .2) != nil || Render(Exploration, 0) != nil {
		t.Error("靜音場景或零音量不應產生 PCM")
	}
}
