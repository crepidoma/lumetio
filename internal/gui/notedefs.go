package gui

import "github.com/crepidoma/lumetio/internal/settings"

// NoteDefRow は、メモ欄の定義を直すときの 1 行。
type NoteDefRow struct {
	Name string `json:"name"`
	// Type は text か bool。
	Type      string `json:"type"`
	CarryOver bool   `json:"carryOver"`
}

// NoteDefRows は、メモ欄の定義を画面に出す順で返す。
func (s *Service) NoteDefRows() ([]NoteDefRow, error) {
	if s.startErr != nil {
		return nil, s.startErr
	}
	defs, err := s.app().NoteDefs()
	if err != nil {
		return nil, err
	}
	out := []NoteDefRow{}
	for _, d := range defs {
		out = append(out, NoteDefRow{Name: d.Name, Type: string(d.Type), CarryOver: d.CarryOver})
	}
	return out, nil
}

// SaveNoteDefs はメモ欄の定義を書く。base は読んだときの定義。
func (s *Service) SaveNoteDefs(base, rows []NoteDefRow) error {
	if s.startErr != nil {
		return s.startErr
	}
	return s.app().SaveNoteDefs(toNoteDefs(base), toNoteDefs(rows))
}

func toNoteDefs(rows []NoteDefRow) []settings.NoteDef {
	out := []settings.NoteDef{}
	for _, r := range rows {
		out = append(out, settings.NoteDef{Name: r.Name, Type: settings.NoteType(r.Type), CarryOver: r.CarryOver})
	}
	return out
}
