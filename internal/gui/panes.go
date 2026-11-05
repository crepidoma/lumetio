package gui

import "github.com/crepidoma/lumetio/internal/localconf"

// Panes はペインの幅 (px) と、右ペインを手で畳んでいるか。
type Panes struct {
	// Left と Right は、保存されていなければ 0。範囲に収めるのは画面で行う。
	Left  int `json:"left"`
	Right int `json:"right"`
	// RightCollapsed は、右ペインを閉じるボタンで畳んでいるか。
	RightCollapsed bool `json:"rightCollapsed"`
}

// Panes は、保存されているペインの幅と、右ペインを畳んでいるかを返す。
func (s *Service) Panes() Panes {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Panes{Left: s.panes.Left, Right: s.panes.Right, RightCollapsed: s.panes.RightCollapsed}
}

// SavePanes は、ペインの幅と、右ペインを畳んでいるかを local.yaml に書く。
func (s *Service) SavePanes(p Panes) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panes = localconf.Panes{Left: p.Left, Right: p.Right, RightCollapsed: p.RightCollapsed}
	if s.env.Local == nil {
		return nil
	}
	return s.env.Local.SavePanes(s.panes)
}
