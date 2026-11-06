package effort

import (
	"testing"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/timeexpr"
)

func times(t *testing.T, ss ...string) []timeexpr.Entry {
	t.Helper()
	var es []timeexpr.Entry
	for _, s := range ss {
		e, err := timeexpr.ParseStored(s)
		if err != nil {
			t.Fatal(err)
		}
		es = append(es, e)
	}
	return es
}

func TestSummarize(t *testing.T) {
	dev := dayfile.Project{Code: "D-1", Name: "開発"}
	day := dayfile.Day{Tasks: []dayfile.Task{
		{Title: "改修", Project: dev, Time: times(t, "09:30-11:30")},
		{Title: "精算", Project: dayfile.Project{Code: "X-1", Name: "事務"}, Time: times(t, "30m", "60m", "16:00-17:00")},
		{Title: "調べもの", Project: dev, Untimed: true},
		{Title: "勉強会", Project: dev, Exclude: true, Time: times(t, "12:00-13:00")},
		{Title: "夜間作業", Project: dayfile.Project{Code: "M-1"}, Time: times(t, "23:00-05:00")},
		{Title: "雑談", Project: dayfile.Project{Code: "Z-1", Name: "雑務"}, Exclude: true, Untimed: true},
		{Title: "付け先なし", Time: times(t, "15m")},
		// 時間未入力 (時間の入れ忘れ) は、時間不定として数えない。
		{Title: "入れ忘れ", Project: dayfile.Project{Code: "Y-1"}},
	}}
	s := Summarize(day)

	if s.TotalMinutes != 120+150+360+15 {
		t.Errorf("TotalMinutes = %d", s.TotalMinutes)
	}

	wantTasks := []struct {
		minutes          int
		untimed, counted bool
	}{
		{120, false, true},
		{150, false, true},
		{0, true, false},
		{60, false, false},
		{360, false, true},
		{0, true, false},
		{15, false, true},
		{0, false, false},
	}
	for i, w := range wantTasks {
		got := s.Tasks[i]
		if got.Minutes != w.minutes || got.Untimed != w.untimed || got.Counted != w.counted {
			t.Errorf("Tasks[%d] (%s) = %d/%v/%v, want %d/%v/%v", i, got.Title, got.Minutes, got.Untimed, got.Counted, w.minutes, w.untimed, w.counted)
		}
	}

	wantProjects := []struct {
		name, code          string
		minutes             int
		counted, hasUntimed bool
		tasks               int
	}{
		{"開発", "D-1", 120, true, true, 3},
		{"事務", "X-1", 150, true, false, 1},
		{"M-1", "M-1", 360, true, false, 1},
		{"雑務", "Z-1", 0, false, false, 1},
		{"", "", 15, true, false, 1},
		{"Y-1", "Y-1", 0, false, false, 1},
	}
	if len(s.Projects) != len(wantProjects) {
		t.Fatalf("len(Projects) = %d", len(s.Projects))
	}
	for i, w := range wantProjects {
		got := s.Projects[i]
		if got.Name != w.name || got.Code != w.code || got.Minutes != w.minutes || got.Counted != w.counted || got.HasUntimed != w.hasUntimed || len(got.Tasks) != w.tasks {
			t.Errorf("Projects[%d] = %q/%q/%d/%v/%v/%d, want %+v", i, got.Name, got.Code, got.Minutes, got.Counted, got.HasUntimed, len(got.Tasks), w)
		}
	}
}

func TestSummarizeGroupsByCode(t *testing.T) {
	s := Summarize(dayfile.Day{Tasks: []dayfile.Task{
		{Title: "a", Project: dayfile.Project{Code: "D-1", Name: "開発"}, Time: times(t, "30m")},
		{Title: "b", Project: dayfile.Project{Code: "D-2", Name: "開発"}, Time: times(t, "45m")},
		{Title: "c", Project: dayfile.Project{Code: "D-1", Name: "開発(新)"}, Time: times(t, "15m")},
		{Title: "d", Project: dayfile.Project{Code: "D-2"}, Time: times(t, "10m")},
	}})
	want := []struct {
		name, code string
		minutes    int
	}{
		{"開発", "D-1", 45},
		{"開発", "D-2", 55},
	}
	if len(s.Projects) != len(want) {
		t.Fatalf("Projects = %+v", s.Projects)
	}
	for i, w := range want {
		got := s.Projects[i]
		if got.Name != w.name || got.Code != w.code || got.Minutes != w.minutes {
			t.Errorf("Projects[%d] = %q/%q/%d, want %+v", i, got.Name, got.Code, got.Minutes, w)
		}
	}
}

func TestSummarizeEmptyDay(t *testing.T) {
	s := Summarize(dayfile.Day{})
	if len(s.Tasks) != 0 || len(s.Projects) != 0 || s.TotalMinutes != 0 {
		t.Errorf("Summary = %+v", s)
	}
}

func TestCombine(t *testing.T) {
	first := Summarize(dayfile.Day{Tasks: []dayfile.Task{
		{Title: "改修", Project: dayfile.Project{Code: "D-1", Name: "開発"}, Time: times(t, "60m")},
		{Title: "勉強会", Project: dayfile.Project{Code: "S-1"}, Exclude: true, Time: times(t, "30m")},
	}})
	second := Summarize(dayfile.Day{Tasks: []dayfile.Task{
		{Title: "調べもの", Project: dayfile.Project{Code: "R-1"}, Untimed: true},
		{Title: "改修", Project: dayfile.Project{Code: "D-1", Name: "開発 (改)"}, Time: times(t, "90m")},
	}})
	got := Combine([]Summary{first, second})
	if got.TotalMinutes != 150 || len(got.Projects) != 3 || got.Tasks != nil {
		t.Fatalf("Combine = %+v", got)
	}
	// コードでまとめ、名前は期間の中で最初に出てきたもの。
	if p := got.Projects[0]; p.Code != "D-1" || p.Name != "開発" || p.Minutes != 150 || !p.Counted || p.HasUntimed {
		t.Errorf("D-1 = %+v", p)
	}
	if p := got.Projects[1]; p.Code != "S-1" || p.Minutes != 0 || p.Counted || p.HasUntimed {
		t.Errorf("S-1 = %+v", p)
	}
	if p := got.Projects[2]; p.Code != "R-1" || p.Counted || !p.HasUntimed {
		t.Errorf("R-1 = %+v", p)
	}
}
