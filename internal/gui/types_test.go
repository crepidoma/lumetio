package gui

import (
	"testing"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/effort"
	"github.com/crepidoma/lumetio/internal/timeexpr"
)

// 長さの表記を読めない形にしていても、直す欄にはそのまま読める形を入れる。
func TestTimeEntriesInput(t *testing.T) {
	es, err := dayfile.ParseTimes([]string{"90m", "09:00-10:30"})
	if err != nil {
		t.Fatal(err)
	}
	got := timeEntries(es, durationFormat("H時間m分"))
	if got[0].Label != "1時間30分" || got[1].Label != "09:00-10:30" {
		t.Errorf("labels = %q, %q", got[0].Label, got[1].Label)
	}
	for i, te := range got {
		parsed, err := timeexpr.Parse(te.Input, timeexpr.Context{})
		if err != nil || len(parsed) != 1 || parsed[0] != es[i] {
			t.Errorf("Input %q を読むと %v, %v", te.Input, parsed, err)
		}
	}
}

// 合計は、時間不定なら空にして Untimed で伝え、時間未入力なら 0 を出す。
func TestNewDayRecordTotals(t *testing.T) {
	day, err := dayfile.Parse([]byte("date: 2026-11-05\ntasks:\n  - title: 実装\n    time: [90m]\n  - title: 調査\n    untimed: true\n  - title: 設計\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := newDay(app.DayView{Day: day, Summary: effort.Summarize(day)}).Records
	want := []struct {
		total   string
		untimed bool
	}{{"1:30", false}, {"", true}, {"0:00", false}}
	for i, w := range want {
		if got[i].Total != w.total || got[i].Untimed != w.untimed {
			t.Errorf("%s = %q/%v, want %+v", got[i].Title, got[i].Total, got[i].Untimed, w)
		}
	}
}
