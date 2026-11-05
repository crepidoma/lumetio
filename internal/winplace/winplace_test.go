package winplace

import (
	"testing"

	"github.com/crepidoma/lumetio/internal/localconf"
)

func TestPlace(t *testing.T) {
	minSize := Size{720, 520}
	def := Size{1280, 800}
	// 主の画面 (1920×1040) と、その右の外付けの画面 (2560×1400)。
	primary := Screen{WorkArea: Rect{0, 0, 1920, 1040}, Primary: true}
	external := Screen{WorkArea: Rect{1920, 0, 2560, 1400}}
	both := []Screen{primary, external}

	tests := []struct {
		name      string
		saved     *localconf.Window
		screens   []Screen
		want      Rect
		maximized bool
	}{
		{"前回なし", nil, both, Rect{320, 120, 1280, 800}, false},
		{
			"前回の位置",
			&localconf.Window{Width: 1000, Height: 700, X: 2100, Y: 50, HasPosition: true},
			both, Rect{2100, 50, 1000, 700}, false,
		},
		{
			"外付けの画面を外した",
			&localconf.Window{Width: 1000, Height: 700, X: 2100, Y: 50, HasPosition: true, Maximized: true},
			[]Screen{primary}, Rect{460, 170, 1000, 700}, true,
		},
		{
			"位置がない",
			&localconf.Window{Width: 1000, Height: 700},
			both, Rect{460, 170, 1000, 700}, false,
		},
		{
			"画面より大きい",
			&localconf.Window{Width: 3000, Height: 2000, X: 100, Y: 100, HasPosition: true},
			[]Screen{primary}, Rect{0, 0, 1920, 1040}, false,
		},
		{
			"最小より小さい",
			&localconf.Window{Width: 300, Height: 200, X: 100, Y: 100, HasPosition: true},
			[]Screen{primary}, Rect{100, 100, 720, 520}, false,
		},
		{
			"右下にはみ出す",
			&localconf.Window{Width: 1000, Height: 700, X: 1300, Y: 900, HasPosition: true},
			[]Screen{primary}, Rect{920, 340, 1000, 700}, false,
		},
		{"画面の情報がない", nil, nil, Rect{320, 140, 1280, 800}, false},
	}
	for _, tt := range tests {
		got, maximized := Place(tt.saved, tt.screens, minSize, def)
		if got != tt.want || maximized != tt.maximized {
			t.Errorf("%s: %+v, %v, want %+v, %v", tt.name, got, maximized, tt.want, tt.maximized)
		}
	}
}
