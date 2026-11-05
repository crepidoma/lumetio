package items

import (
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
)

const sample = `- title: 朝会
  project:
    code: MEETING
    name: 会議
  time: ['09:00-09:15']
  repeat:
    freq: weekly
    by_day: [mo, tu, we, th, fr]
- title: 隔週の打ち合わせ
  time: ['13:00-14:00']
  repeat:
    freq: weekly
    interval: 2
    by_day: [we]
    start: 2026-11-04
    until: 2026-12-31
- title: 月例報告会
  repeat:
    freq: monthly
    by_day: [2mo]
- title: 月末の振り返り
  repeat:
    freq: monthly
    by_day: [-1fr]
- title: 経費の精算
  repeat:
    freq: monthly
    by_month_day: [25]
- title: 日次の確認
  repeat:
    freq: daily
    start: 2026-11-01
- title: 問い合わせ対応
  tags: [サポート]
  exclude: true
  time: [30m, '17:00-17:30']
  favorite: true
`

func TestRoundTrip(t *testing.T) {
	items, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != sample {
		t.Errorf("Marshal(Parse(sample)) =\n%s\nwant\n%s", out, sample)
	}
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestParse(t *testing.T) {
	items, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 7 {
		t.Fatalf("len(items) = %d", len(items))
	}

	morning := items[0]
	if morning.Project != (dayfile.Project{Code: "MEETING", Name: "会議"}) || len(morning.Time) != 1 || morning.Time[0].Minutes != 15 {
		t.Errorf("items[0] = %+v", morning)
	}
	if r := morning.Repeat; r.Freq != Weekly || r.Interval != 1 || len(r.ByDay) != 5 || r.ByDay[0] != (DaySpec{Weekday: time.Monday}) {
		t.Errorf("items[0].Repeat = %+v", r)
	}

	biweekly := items[1].Repeat
	if biweekly.Interval != 2 || !biweekly.Start.Equal(date(2026, 11, 4)) || !biweekly.Until.Equal(date(2026, 12, 31)) {
		t.Errorf("items[1].Repeat = %+v", biweekly)
	}
	if got := items[2].Repeat.ByDay; len(got) != 1 || got[0] != (DaySpec{N: 2, Weekday: time.Monday}) {
		t.Errorf("items[2].Repeat.ByDay = %+v", got)
	}
	if got := items[3].Repeat.ByDay; len(got) != 1 || got[0] != (DaySpec{N: -1, Weekday: time.Friday}) {
		t.Errorf("items[3].Repeat.ByDay = %+v", got)
	}
	if got := items[4].Repeat.ByMonthDay; got != 25 {
		t.Errorf("items[4].Repeat.ByMonthDay = %d", got)
	}

	favorite := items[6]
	if !favorite.Favorite || !favorite.Exclude || favorite.Repeat != nil || len(favorite.Time) != 2 || len(favorite.Tags) != 1 {
		t.Errorf("items[6] = %+v", favorite)
	}
}

// interval: 1 は省略時の値なので、書き出すときは書かない。
func TestMarshalOmitsDefaultInterval(t *testing.T) {
	items, err := Parse([]byte("- title: a\n  repeat:\n    freq: daily\n    interval: 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	if want := "- title: a\n  repeat:\n    freq: daily\n"; string(out) != want {
		t.Errorf("Marshal =\n%s\nwant\n%s", out, want)
	}
}

// 設定をすべて外したタスクは、items.yaml に残さない。
func TestMarshalDropsItemsWithoutSettings(t *testing.T) {
	out, err := Marshal([]Item{
		{Title: "設定なし"},
		// プロジェクト、タグ、計上しないは設定ではない。
		{Title: "プロジェクトだけ", Project: dayfile.Project{Code: "DEV-001"}, Tags: []string{"調査"}, Exclude: true},
		{Title: "よく使う", Favorite: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "- title: よく使う\n  favorite: true\n"; string(out) != want {
		t.Errorf("Marshal =\n%s\nwant\n%s", out, want)
	}

	out, err = Marshal([]Item{{Title: "設定なし"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("Marshal = %q, want 空", out)
	}
}

func TestParseEmpty(t *testing.T) {
	items, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("items = %+v", items)
	}
}

func TestParseErrors(t *testing.T) {
	tests := map[string]string{
		"一覧でない":                     "title: a\n",
		"知らないキー":                    "- title: a\n  done: 2026-10-20\n",
		"title がない":                 "- favorite: true\n",
		"タイトルが重なる":                  "- title: a\n  favorite: true\n- title: a\n  exclude: true\n",
		"プロジェクトに code がない":          "- title: a\n  project:\n    name: b\n",
		"時間が保存形式でない":                "- title: a\n  time: [9-10]\n",
		"繰り返しの知らないキー":               "- title: a\n  repeat:\n    freq: daily\n    count: 3\n",
		"freq がない":                  "- title: a\n  repeat:\n    by_day: [mo]\n",
		"freq が使えない":                "- title: a\n  repeat:\n    freq: yearly\n",
		"daily に by_day":            "- title: a\n  repeat:\n    freq: daily\n    by_day: [mo]\n",
		"weekly に by_day がない":       "- title: a\n  repeat:\n    freq: weekly\n",
		"weekly の by_day に週の番号":     "- title: a\n  repeat:\n    freq: weekly\n    by_day: [2mo]\n",
		"weekly の by_day が重なる":      "- title: a\n  repeat:\n    freq: weekly\n    by_day: [mo, mo]\n",
		"weekly に by_month_day":     "- title: a\n  repeat:\n    freq: weekly\n    by_day: [mo]\n    by_month_day: [1]\n",
		"曜日が読めない":                   "- title: a\n  repeat:\n    freq: weekly\n    by_day: [MO]\n",
		"monthly の by_day に週の番号がない": "- title: a\n  repeat:\n    freq: monthly\n    by_day: [mo]\n",
		"monthly の by_day が 2 つ":    "- title: a\n  repeat:\n    freq: monthly\n    by_day: [1mo, 3mo]\n",
		"monthly の週の番号が範囲外":         "- title: a\n  repeat:\n    freq: monthly\n    by_day: [6mo]\n",
		"monthly に両方":               "- title: a\n  repeat:\n    freq: monthly\n    by_day: [1mo]\n    by_month_day: [1]\n",
		"monthly にどちらもない":           "- title: a\n  repeat:\n    freq: monthly\n",
		"by_month_day が 2 つ":        "- title: a\n  repeat:\n    freq: monthly\n    by_month_day: [1, 15]\n",
		"by_month_day が範囲外":         "- title: a\n  repeat:\n    freq: monthly\n    by_month_day: [32]\n",
		"by_month_day が負":           "- title: a\n  repeat:\n    freq: monthly\n    by_month_day: [-1]\n",
		"interval が 3":              "- title: a\n  repeat:\n    freq: weekly\n    interval: 3\n    by_day: [mo]\n    start: 2026-11-02\n",
		"monthly で interval 2":      "- title: a\n  repeat:\n    freq: monthly\n    interval: 2\n    by_month_day: [1]\n    start: 2026-11-01\n",
		"隔週に start がない":             "- title: a\n  repeat:\n    freq: weekly\n    interval: 2\n    by_day: [we]\n",
		"start が読めない":               "- title: a\n  repeat:\n    freq: daily\n    start: 2026/11/01\n",
		"until が start より前":         "- title: a\n  repeat:\n    freq: daily\n    start: 2026-11-02\n    until: 2026-11-01\n",
		"YAML として壊れている":             "- title: [\n",
	}
	for name, input := range tests {
		if _, err := Parse([]byte(input)); err == nil {
			t.Errorf("%s: Parse succeeded, want error", name)
		}
	}
}
