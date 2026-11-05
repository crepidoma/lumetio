package gui

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/crepidoma/lumetio/internal/items"
)

// 画面と受け渡す繰り返しは、items の形と行き来しても変わらない。
func TestRepeatRoundTrip(t *testing.T) {
	for _, r := range []Repeat{
		{Freq: "daily", Interval: 1, ByDay: []string{}},
		{Freq: "weekly", Interval: 1, ByDay: []string{"mo", "tu", "we", "th", "fr"}},
		{Freq: "weekly", Interval: 2, ByDay: []string{"we"}, Start: "2026-11-04", Until: "2026-12-31"},
		{Freq: "monthly", Interval: 1, ByDay: []string{"-1fr"}},
		{Freq: "monthly", Interval: 1, ByDay: []string{}, ByMonthDay: 25},
	} {
		it, err := r.toItems()
		if err != nil {
			t.Fatalf("%+v: %v", r, err)
		}
		if got := repeatOf(it); !reflect.DeepEqual(*got, r) {
			t.Errorf("%+v, want %+v", *got, r)
		}
	}

	bad := Repeat{Freq: "weekly", ByDay: []string{"月"}}
	if _, err := bad.toItems(); err == nil {
		t.Error("読めない曜日がエラーにならない")
	}
}

// 編集を開いたときに渡した値は、そのまま返されたら元のタスクと同じに戻る。
// 戻らないと、他で変わっていないのに保存が衝突になるため。
func TestTaskEditRoundTrip(t *testing.T) {
	data := `- title: 朝会
  project:
    code: MTG
  tags: [会議]
  exclude: true
  time: ['09:00-09:15', 90m, 45m]
  repeat:
    freq: weekly
    interval: 2
    by_day: [we]
    start: 2026-11-04
    until: 2026-12-31
  favorite: true
- title: 月例
  project:
    code: ADMIN
    name: 事務
  time: [30m]
  repeat:
    freq: monthly
    by_day: [-1fr]
`
	list, err := items.Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range list {
		e := taskEditOf(it)
		got, err := e.toItem()
		if err != nil {
			t.Fatalf("%s: %v", it.Title, err)
		}
		want, _ := items.Marshal([]items.Item{it})
		back, _ := items.Marshal([]items.Item{got})
		if !bytes.Equal(back, want) {
			t.Errorf("%s:\n%s\nwant\n%s", it.Title, back, want)
		}
	}
}
