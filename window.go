package main

import (
	"log"
	"sync"

	"github.com/crepidoma/lumetio/internal/localconf"
	"github.com/crepidoma/lumetio/internal/winplace"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// 初めて開くときの大きさと、縮められる最小の大きさ。
// 最小の幅は、左ペインの最小、中央の最小、畳んだ右のバーが並ぶ幅。最小の高さは、記録の行が 3 つほど見える高さ。
var (
	defaultSize = winplace.Size{Width: 1280, Height: 800}
	minSize     = winplace.Size{Width: 720, Height: 520}
)

// normalBounds は、最大化も最小化もしていないときの、最後の大きさと位置。
// 最大化したまま閉じても、元に戻したときの大きさを残せるよう、普通の状態のときだけ覚える。
type normalBounds struct {
	mu   sync.Mutex
	rect application.Rect
}

func (b *normalBounds) set(r application.Rect) {
	b.mu.Lock()
	b.rect = r
	b.mu.Unlock()
}

func (b *normalBounds) get() application.Rect {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rect
}

// restoreWindow は、起動したときに前回の大きさと位置でウィンドウを見せ、閉じるときにそれを保存する。
// file が nil (ローカル設定を読めなかった) なら、保存しない。
func restoreWindow(wails *application.App, win *application.WebviewWindow, saved *localconf.Window, file *localconf.File) {
	var bounds normalBounds

	// 画面の並びは、アプリが動き始めてから分かる。
	wails.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		var screens []winplace.Screen
		for _, s := range wails.Screen.GetAll() {
			a := s.WorkArea
			screens = append(screens, winplace.Screen{WorkArea: winplace.Rect{X: a.X, Y: a.Y, Width: a.Width, Height: a.Height}, Primary: s.IsPrimary})
		}
		r, maximized := winplace.Place(saved, screens, minSize, defaultSize)
		rect := application.Rect{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height}
		win.SetBounds(rect)
		bounds.set(rect)
		win.Show()
		if maximized {
			win.Maximise()
		}
	})

	track := func(*application.WindowEvent) {
		if !win.IsMaximised() && !win.IsMinimised() {
			bounds.set(win.Bounds())
		}
	}
	win.OnWindowEvent(events.Common.WindowDidMove, track)
	win.OnWindowEvent(events.Common.WindowDidResize, track)

	if file == nil {
		return
	}
	win.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) {
		r := bounds.get()
		w := localconf.Window{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height, HasPosition: true, Maximized: win.IsMaximised()}
		if err := file.SaveWindow(w); err != nil {
			// 閉じる途中なので、画面には出せない。次に開くときは前回の値か初期値で開く。
			log.Printf("ウィンドウの大きさを保存できません: %v", err)
		}
	})
}
