package gui

import (
	"testing"

	"github.com/crepidoma/lumetio/internal/dayfile"
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
