// Package winplace は、前回のウィンドウと今つながっている画面から、ウィンドウを開く大きさと位置を決める。
package winplace

import "github.com/crepidoma/lumetio/internal/localconf"

// Rect は画面上の長方形。
type Rect struct {
	X, Y, Width, Height int
}

// Size は大きさ。
type Size struct {
	Width, Height int
}

// Screen は、つながっている画面 1 つの、ウィンドウを置ける範囲 (タスクバーなどを除いた範囲)。
type Screen struct {
	WorkArea Rect
	Primary  bool
}

// Place は、ウィンドウを開く大きさと位置と、最大化するかを返す。
// saved が nil なら、初期の大きさで主の画面の中央に開く。
// 前回の位置が今の画面のどれにも入らなければ (外付けの画面を外したときなど)、主の画面の中央に開く。
// 大きさは、最小の大きさより小さくせず、置く画面に収める。
func Place(saved *localconf.Window, screens []Screen, minSize, defaultSize Size) (Rect, bool) {
	primary := primaryArea(screens)
	size := defaultSize
	maximized := false
	if saved != nil {
		if saved.Width > 0 && saved.Height > 0 {
			size = Size{saved.Width, saved.Height}
		}
		maximized = saved.Maximized
		if saved.HasPosition {
			if area, ok := screenOf(screens, saved.X, saved.Y, size.Width); ok {
				s := fit(size, minSize, area)
				// 収めた大きさがはみ出さないよう、位置も画面の中に寄せる。
				x := clamp(saved.X, area.X, area.X+area.Width-s.Width)
				y := clamp(saved.Y, area.Y, area.Y+area.Height-s.Height)
				return Rect{x, y, s.Width, s.Height}, maximized
			}
		}
	}
	s := fit(size, minSize, primary)
	return Rect{
		X:      primary.X + (primary.Width-s.Width)/2,
		Y:      primary.Y + (primary.Height-s.Height)/2,
		Width:  s.Width,
		Height: s.Height,
	}, maximized
}

func primaryArea(screens []Screen) Rect {
	for _, s := range screens {
		if s.Primary {
			return s.WorkArea
		}
	}
	if len(screens) > 0 {
		return screens[0].WorkArea
	}
	// 画面の情報が取れないときは、よくある大きさの画面とみなす。
	return Rect{0, 0, 1920, 1080}
}

// screenOf は、位置 (x, y) に幅 width で置いたウィンドウのタイトルバーの真ん中が入る画面を探す。
// タイトルバーが見えていれば、ドラッグして動かせるため。
func screenOf(screens []Screen, x, y, width int) (Rect, bool) {
	px, py := x+width/2, y+8
	for _, s := range screens {
		a := s.WorkArea
		if px >= a.X && px < a.X+a.Width && py >= a.Y && py < a.Y+a.Height {
			return a, true
		}
	}
	return Rect{}, false
}

// fit は、size を最小の大きさより小さくせず、area に収める。
// area が最小の大きさより小さいときは、最小の大きさにする。
func fit(size, minSize Size, area Rect) Size {
	return Size{
		Width:  max(minSize.Width, min(size.Width, area.Width)),
		Height: max(minSize.Height, min(size.Height, area.Height)),
	}
}

// clamp は v を lo から hi のあいだに収める。hi が lo より小さいときは lo にする。
func clamp(v, lo, hi int) int {
	return max(lo, min(v, hi))
}
