package gui

import (
	"fmt"

	"github.com/crepidoma/lumetio/internal/watch"
)

// ChangedEvent は、データフォルダのファイルが変わったことを画面に知らせるイベントの名前。中身は watch.Change。
const ChangedEvent = "data-changed"

// Watch は、今のデータフォルダの監視を始める。前の監視があれば止める。
// 他の PC などがファイルを書き換えたら、Env.Notify で画面に知らせる。
// 起動と終了のときにアプリが呼ぶもので、画面からは呼ばないので、bindings に出さない。
//
//wails:ignore
func (s *Service) Watch() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.watchLocked()
}

// StopWatch は監視をやめる。
//
//wails:ignore
func (s *Service) StopWatch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.watcher != nil {
		_ = s.watcher.Close()
		s.watcher = nil
	}
}

// watchLocked は、s.mu を持ったまま、今のデータフォルダの監視を始め直す。
// ローカル設定を読めなかったときは、どの操作も誤りを返すので、監視もしない。
func (s *Service) watchLocked() error {
	if s.watcher != nil {
		_ = s.watcher.Close()
		s.watcher = nil
	}
	if s.startErr != nil || s.env.Notify == nil {
		return nil
	}
	// 自分で書いたファイルの知らせは捨てる。書いた結果はもう画面に返しているので、読み直しは要らないため。
	w, err := watch.Start(s.locations.DataDir, s.env.Notify, s.app().WrittenHere)
	if err != nil {
		return fmt.Errorf("%s の監視を始められません: %w", s.locations.DataDir, err)
	}
	s.watcher = w
	return nil
}
