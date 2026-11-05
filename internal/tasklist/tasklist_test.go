package tasklist

import (
	"slices"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/timeexpr"
)

func date(m time.Month, d int) time.Time {
	return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC)
}

func titles(rs []Recent) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Last.Title)
	}
	return out
}

func TestRecentTasks(t *testing.T) {
	days := []dayfile.Day{
		{Date: date(11, 5), Tasks: []dayfile.Task{{Title: "改修", Project: dayfile.Project{Code: "DEV-2"}}, {Title: "精算"}}},
		// 14 日の範囲の外 (11/5 を開いたときの範囲は 10/23〜11/5)。
		{Date: date(10, 22), Tasks: []dayfile.Task{{Title: "古い調査"}}},
		{Date: date(10, 23), Tasks: []dayfile.Task{{Title: "改修", Project: dayfile.Project{Code: "DEV-1"}}, {Title: "範囲の端"}}},
		{Date: date(11, 3), Tasks: []dayfile.Task{{Title: "終わった改修", Done: true}, {Title: "問い合わせ"}}},
		// 開いている日より後の記録は見ない。
		{Date: date(11, 6), Tasks: []dayfile.Task{{Title: "明日の会議"}}},
	}
	got := RecentTasks(days, nil, date(11, 5))

	want := []string{"改修", "精算", "問い合わせ", "範囲の端"}
	if !slices.Equal(titles(got), want) {
		t.Fatalf("RecentTasks = %v, want %v", titles(got), want)
	}
	// 最後の記録は、いちばん新しい日の記録。
	if got[0].Last.Project.Code != "DEV-2" || !got[0].LastDate.Equal(date(11, 5)) {
		t.Errorf("改修の最後の記録 = %+v (%s)", got[0].Last, got[0].LastDate)
	}
}

// ＋で記録するときの値は、items.yaml にあれば items.yaml の値、なければ最後の記録から写したもの。
// 一覧に出す値を、記録される値とそろえるため。
func TestRecentTasksBase(t *testing.T) {
	usual := []timeexpr.Entry{{Minutes: 30}}
	days := []dayfile.Day{
		{Date: date(11, 4), Tasks: []dayfile.Task{
			{Title: "調査", Project: dayfile.Project{Code: "OLD-1"}, Note: "メモ"},
			{Title: "雑談", Project: dayfile.Project{Code: "ETC-1"}, Exclude: true, Note: "メモ", Time: usual},
		}},
	}
	list := []items.Item{{Title: "調査", Project: dayfile.Project{Code: "DEV-1"}, Tags: []string{"設計"}, Time: usual}}
	got := RecentTasks(days, list, date(11, 5))
	if len(got) != 2 {
		t.Fatalf("RecentTasks = %v", titles(got))
	}
	if b := got[0].Base; b.Project.Code != "DEV-1" || !slices.Equal(b.Tags, []string{"設計"}) || len(b.Time) != 1 {
		t.Errorf("調査の Base = %+v", b)
	}
	// 時間とメモはその日ごとのものなので写さない。
	if b := got[1].Base; b.Project.Code != "ETC-1" || !b.Exclude || len(b.Time) != 0 || b.Note != "" {
		t.Errorf("雑談の Base = %+v", b)
	}
}

// 過去の日を開くと、その日の時点の直近のタスクになる。
func TestRecentTasksFromPastDay(t *testing.T) {
	days := []dayfile.Day{
		{Date: date(10, 23), Tasks: []dayfile.Task{{Title: "改修", Project: dayfile.Project{Code: "DEV-1"}}}},
		{Date: date(11, 5), Tasks: []dayfile.Task{{Title: "改修", Project: dayfile.Project{Code: "DEV-2"}}}},
	}
	got := RecentTasks(days, nil, date(10, 30))
	if len(got) != 1 || got[0].Last.Project.Code != "DEV-1" {
		t.Errorf("RecentTasks = %+v, want 10/23 の記録", got)
	}
}

// 完了の後にもう一度記録したら、直近のタスクに戻る。
func TestRecentTasksDoneThenRecordedAgain(t *testing.T) {
	days := []dayfile.Day{
		{Date: date(11, 2), Tasks: []dayfile.Task{{Title: "改修", Done: true}}},
		{Date: date(11, 4), Tasks: []dayfile.Task{{Title: "改修"}}},
	}
	if got := titles(RecentTasks(days, nil, date(11, 5))); !slices.Equal(got, []string{"改修"}) {
		t.Errorf("RecentTasks = %v", got)
	}
	// 11/3 を開くと、最後の記録は 11/2 の完了なので出さない。
	if got := RecentTasks(days, nil, date(11, 3)); len(got) != 0 {
		t.Errorf("RecentTasks(11/3) = %v, want なし", titles(got))
	}
}

// 記録を作った後に items.yaml の値を変えても、記録は変わらない。
func TestFromItemCopies(t *testing.T) {
	it := items.Item{Title: "朝会", Tags: []string{"会議"}, Time: []timeexpr.Entry{{Minutes: 15}}}
	task := FromItem(it)
	it.Tags[0] = "変更"
	it.Time[0].Minutes = 99
	if task.Tags[0] != "会議" || task.Time[0].Minutes != 15 {
		t.Errorf("FromItem の結果が items.yaml の値と共有されている: %+v", task)
	}
}

func TestNewItem(t *testing.T) {
	last := dayfile.Task{Title: "改修", Project: dayfile.Project{Code: "DEV-1"}, Tags: []string{"a"}, Exclude: true, Note: "メモ", Time: []timeexpr.Entry{{Minutes: 30}}}
	it := NewItem(last)
	if it.Title != "改修" || it.Project.Code != "DEV-1" || !slices.Equal(it.Tags, []string{"a"}) || !it.Exclude || len(it.Time) != 0 || it.Repeat != nil || it.Favorite {
		t.Errorf("NewItem = %+v", it)
	}
}

func TestFavoritesAndRepeating(t *testing.T) {
	thursday := &items.Repeat{Freq: items.Weekly, Interval: 1, ByDay: []items.DaySpec{{Weekday: time.Thursday}}}
	monday := &items.Repeat{Freq: items.Weekly, Interval: 1, ByDay: []items.DaySpec{{Weekday: time.Monday}}}
	ended := &items.Repeat{Freq: items.Daily, Interval: 1, Until: date(10, 31)}
	later := &items.Repeat{Freq: items.Daily, Interval: 1, Start: date(12, 1)}
	list := []items.Item{
		{Title: "問い合わせ", Favorite: true},
		{Title: "木曜の会議", Repeat: thursday},
		{Title: "月曜の会議", Repeat: monday},
		{Title: "終わった定期", Repeat: ended},
		{Title: "来月から", Repeat: later, Favorite: true},
	}

	var favs []string
	for _, it := range Favorites(list) {
		favs = append(favs, it.Title)
	}
	if !slices.Equal(favs, []string{"問い合わせ", "来月から"}) {
		t.Errorf("Favorites = %v", favs)
	}

	// 2026-11-05 は木曜。始まる前と終わったものは、どちらにも入らない。
	on, off := Repeating(list, date(11, 5))
	var onTitles, offTitles []string
	for _, it := range on {
		onTitles = append(onTitles, it.Title)
	}
	for _, it := range off {
		offTitles = append(offTitles, it.Title)
	}
	if !slices.Equal(onTitles, []string{"木曜の会議"}) || !slices.Equal(offTitles, []string{"月曜の会議"}) {
		t.Errorf("Repeating = %v, %v", onTitles, offTitles)
	}
}

func TestPlans(t *testing.T) {
	thursday := &items.Repeat{Freq: items.Weekly, Interval: 1, ByDay: []items.DaySpec{{Weekday: time.Thursday}}}
	list := []items.Item{
		{Title: "朝会", Repeat: &items.Repeat{Freq: items.Daily, Interval: 1}},
		{Title: "週次", Repeat: thursday},
		{Title: "中止", Repeat: thursday},
		{Title: "記録済み", Repeat: thursday},
		{Title: "よく使う", Favorite: true},
	}
	// 2026-11-05 は木曜。
	day := dayfile.Day{
		Date:    date(11, 5),
		Tasks:   []dayfile.Task{{Title: "記録済み"}},
		Skipped: []string{"中止"},
	}
	var got []string
	for _, it := range Plans(day, list) {
		got = append(got, it.Title)
	}
	if !slices.Equal(got, []string{"朝会", "週次"}) {
		t.Errorf("Plans = %v", got)
	}

	// 金曜には週次は出ない。
	got = nil
	for _, it := range Plans(dayfile.Day{Date: date(11, 6)}, list) {
		got = append(got, it.Title)
	}
	if !slices.Equal(got, []string{"朝会"}) {
		t.Errorf("Plans(11/6) = %v", got)
	}
}
