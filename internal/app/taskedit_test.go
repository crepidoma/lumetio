package app

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
)

func titlesOfItems(list []items.Item) []string {
	return titlesOf(list, func(it items.Item) string { return it.Title })
}

func TestItem(t *testing.T) {
	f := newFixture(t)
	it, err := f.app.Item("朝会")
	if err != nil {
		t.Fatal(err)
	}
	if it.Project.Code != "MTG" || it.Repeat == nil {
		t.Errorf("朝会 = %+v", it)
	}
	// 編集するのは items.yaml にあるタスクだけ。
	if _, err := f.app.Item("調査"); !IsInputError(err) {
		t.Errorf("items.yaml にない: err = %v, want input error", err)
	}
}

// base は、編集を開いたときに読む値。
func (f *fixture) base(title string) *items.Item {
	f.t.Helper()
	it, err := f.app.Item(title)
	if err != nil {
		f.t.Fatal(err)
	}
	return &it
}

func TestSaveTask(t *testing.T) {
	f := newFixture(t)

	// 新しく作ると items.yaml の最後に足す。プロジェクトは、コードだけなら一覧から名前を入れる。
	err := f.app.SaveTask("", nil, TaskInput{
		Title:   "調査",
		Project: &dayfile.Project{Code: "DEV-001"},
		Tags:    []string{" ふりかえり ", "ふりかえり", ""},
		Time:    "9-10 30",
		Repeat:  &items.Repeat{Freq: items.Weekly, Interval: 2, ByDay: []items.DaySpec{{Weekday: time.Wednesday}}, Start: today},
	})
	if err != nil {
		t.Fatal(err)
	}
	list := f.items()
	it := list[len(list)-1]
	if it.Title != "調査" || it.Project != (dayfile.Project{Code: "DEV-001", Name: "新機能"}) || !slices.Equal(it.Tags, []string{"ふりかえり"}) || len(it.Time) != 2 || it.Repeat == nil || it.Repeat.Interval != 2 {
		t.Errorf("調査 = %+v", it)
	}

	// タイトルを変えると、items.yaml の同じ位置のタスクを書き換える。
	if err := f.app.SaveTask("朝会", f.base("朝会"), TaskInput{Title: "朝の会", Time: "9:00-9:15", Favorite: true}); err != nil {
		t.Fatal(err)
	}
	if got := titlesOfItems(f.items()); !slices.Equal(got, []string{"朝の会", "週次ミーティング", "問い合わせ対応", "調査"}) {
		t.Errorf("items = %v", got)
	}

	// 設定をすべて外すと、プロジェクトなどが残っていても消す。
	if err := f.app.SaveTask("朝の会", f.base("朝の会"), TaskInput{Title: "朝の会", Project: &dayfile.Project{Code: "MTG"}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := findItem(f.items(), "朝の会"); ok {
		t.Error("朝の会が items.yaml に残っている")
	}
}

// 編集を開いているあいだに他でそのタスクが変わったら、他の変更を上書きしない。
func TestSaveTaskConflict(t *testing.T) {
	f := newFixture(t)
	base := f.base("週次ミーティング")
	changed := *base
	changed.Favorite = true
	in := TaskInput{Title: "週次ミーティング", Time: "13:00-15:00"}

	stale := *base
	stale.Time = nil
	if err := f.app.SaveTask("週次ミーティング", &stale, in); !errors.Is(err, ErrConflict) {
		t.Errorf("変わっていた: err = %v, want conflict", err)
	}
	if err := f.app.SaveTask("ない", &changed, in); !errors.Is(err, ErrConflict) {
		t.Errorf("消されていた: err = %v, want conflict", err)
	}
	if got, _ := findItem(f.items(), "週次ミーティング"); len(got.Time) != 1 || got.Time[0].String() != "13:00-14:00" {
		t.Errorf("書き換わった: %+v", got)
	}
}

func TestSaveTaskInputErrors(t *testing.T) {
	f := newFixture(t)
	weekly := func(r items.Repeat) *items.Repeat { r.Freq = items.Weekly; return &r }
	for name, tc := range map[string]struct {
		original string
		in       TaskInput
	}{
		"名前がない":      {"", TaskInput{Title: " ", Favorite: true}},
		"名前が重なる":     {"朝会", TaskInput{Title: "週次ミーティング", Favorite: true}},
		"新しく設定なし":    {"", TaskInput{Title: "新しいタスク"}},
		"直前の記録から":    {"", TaskInput{Title: "新しいタスク", Time: "-12:00"}},
		"今まで":        {"", TaskInput{Title: "新しいタスク", Time: "13:00-"}},
		"曜日がない":      {"", TaskInput{Title: "新しいタスク", Repeat: weekly(items.Repeat{Interval: 1})}},
		"毎月の日がない":    {"", TaskInput{Title: "新しいタスク", Repeat: &items.Repeat{Freq: items.Monthly, Interval: 1}}},
		"隔週にいつからがない": {"", TaskInput{Title: "新しいタスク", Repeat: weekly(items.Repeat{Interval: 2, ByDay: []items.DaySpec{{Weekday: time.Monday}}})}},
		"期間が逆":       {"", TaskInput{Title: "新しいタスク", Repeat: &items.Repeat{Freq: items.Daily, Interval: 1, Start: today, Until: today.AddDate(0, 0, -1)}}},
	} {
		var base *items.Item
		if tc.original != "" {
			base = f.base(tc.original)
		}
		if err := f.app.SaveTask(tc.original, base, tc.in); !IsInputError(err) {
			t.Errorf("%s: err = %v, want input error", name, err)
		}
	}
}

func TestCompleteTask(t *testing.T) {
	f := newFixture(t)
	f.write("days/2026/2026-11-03.yaml", "date: 2026-11-03\ntasks:\n  - title: 調査\n    time: [30m]\n")
	f.write("days/2026/2026-11-06.yaml", "date: 2026-11-06\ntasks:\n  - title: 調査\n    time: [30m]\n")

	// 開いている日までで最後に記録した日に付ける。後の日の記録は書き換えない。
	if _, err := f.app.CompleteTask(today, "調査"); err != nil {
		t.Fatal(err)
	}
	if !task(f.day(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)), "調査").Done {
		t.Error("11/3 の記録に完了の印がない")
	}
	if task(f.day(time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)), "調査").Done {
		t.Error("11/6 の記録に完了の印が付いた")
	}

	// 記録のないタスクは完了にできない。
	if _, err := f.app.CompleteTask(today, "朝会"); !IsInputError(err) {
		t.Errorf("記録なし: err = %v, want input error", err)
	}
}
