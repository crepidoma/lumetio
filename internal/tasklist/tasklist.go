// Package tasklist は、左ペインのタスクの一覧と、その日の予定を組み立てる。
// どれも開いている日を起点にする。過去の日を後から埋めるときに、当時のタスクと値を使うため。
package tasklist

import (
	"slices"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// RecentDays は、直近のタスクとして見る日数。開いている日を含む。
const RecentDays = 14

// Recent は直近のタスクの 1 件。
type Recent struct {
	// Last は、そのタスクを最後に記録した日の記録。
	Last     dayfile.Task
	LastDate time.Time
	// Base は、＋で記録するときの値。items.yaml にあれば items.yaml の値、なければ Last から写したもの。
	// 一覧に出す値を、記録される値とそろえるために使う。
	Base dayfile.Task
}

// RecentTasks は、開いている日までの RecentDays 日に記録したタスクを、最後に記録した日が新しい順に返す。
// days には、その期間の日次ファイルを、list には items.yaml のタスクを渡す。期間の外の日は無視する。
// 最後の記録が完了 (done) のタスクは返さない。
func RecentTasks(days []dayfile.Day, list []items.Item, opened time.Time) []Recent {
	to := yamlfile.DateOnly(opened)
	from := to.AddDate(0, 0, -(RecentDays - 1))

	// 新しい日から見ていき、タスクごとに最初に見つけた記録を最後の記録とする。
	sorted := slices.Clone(days)
	slices.SortStableFunc(sorted, func(a, b dayfile.Day) int { return yamlfile.DateOnly(b.Date).Compare(yamlfile.DateOnly(a.Date)) })

	seen := map[string]bool{}
	var out []Recent
	for _, day := range sorted {
		d := yamlfile.DateOnly(day.Date)
		if d.Before(from) || d.After(to) {
			continue
		}
		var found []Recent
		// 同じ日に同じタスクが 2 回あれば、後ろの記録をその日の最後の記録とする。
		for i := len(day.Tasks) - 1; i >= 0; i-- {
			t := day.Tasks[i]
			if seen[t.Title] {
				continue
			}
			seen[t.Title] = true
			if t.Done {
				continue
			}
			base := FromItem(NewItem(t))
			if it, ok := Find(list, t.Title); ok {
				base = FromItem(it)
			}
			found = append(found, Recent{Last: t, LastDate: d, Base: base})
		}
		// 同じ日のタスクは、日次ファイルに書かれた順に並べる。
		slices.Reverse(found)
		out = append(out, found...)
	}
	return out
}

// FromItem は、よく使うタスクや定期的なタスクを＋や実施済で記録するときの記録を作る。
// items.yaml の値だけを使い、過去の記録で穴を埋めない。
func FromItem(it items.Item) dayfile.Task {
	return dayfile.Task{
		Title:   it.Title,
		Project: it.Project,
		Tags:    slices.Clone(it.Tags),
		Exclude: it.Exclude,
		Time:    slices.Clone(it.Time),
	}
}

// NewItem は、タスクを初めて items.yaml に書くときの設定を作る。
// プロジェクト、タグ、計上しないは、開いている日までで一番新しい記録から写す。
// 時間とメモはその日ごとのものなので写さない。items.yaml にない直近のタスクを＋で記録するときも、これを使う。
func NewItem(last dayfile.Task) items.Item {
	return items.Item{
		Title:   last.Title,
		Project: last.Project,
		Tags:    slices.Clone(last.Tags),
		Exclude: last.Exclude,
	}
}

// Favorites は、よく使うタスクを items.yaml の順に返す。
func Favorites(list []items.Item) []items.Item {
	var out []items.Item
	for _, it := range list {
		if it.Favorite {
			out = append(out, it)
		}
	}
	return out
}

// Repeating は、左ペインに出す定期的なタスクを items.yaml の順に返す。
// on は開いている日に予定が出るもの、off はそれ以外 (別の曜日など) で、折りたたんだ領域に出す。
// start より前のもの、until を過ぎたものは、タスクの管理画面にだけ出すので返さない。
func Repeating(list []items.Item, opened time.Time) (on, off []items.Item) {
	d := yamlfile.DateOnly(opened)
	for _, it := range list {
		r := it.Repeat
		if r == nil {
			continue
		}
		if !r.Start.IsZero() && d.Before(yamlfile.DateOnly(r.Start)) {
			continue
		}
		if !r.Until.IsZero() && d.After(yamlfile.DateOnly(r.Until)) {
			continue
		}
		if r.Occurs(d) {
			on = append(on, it)
		} else {
			off = append(off, it)
		}
	}
	return on, off
}

// Plans は、その日に予定として出す定期的なタスクを items.yaml の順に返す。
// その日にもう同じタスクの記録があるか、スキップしたものは出さない。
// 実施済にすると記録ができるので、予定はそこで消える。
func Plans(day dayfile.Day, list []items.Item) []items.Item {
	var out []items.Item
	for _, it := range list {
		if it.Repeat == nil || !it.Repeat.Occurs(day.Date) {
			continue
		}
		if slices.Contains(day.Skipped, it.Title) {
			continue
		}
		if slices.ContainsFunc(day.Tasks, func(t dayfile.Task) bool { return t.Title == it.Title }) {
			continue
		}
		out = append(out, it)
	}
	return out
}

// Find は、items.yaml の一覧から、名前でタスクを探す。
func Find(list []items.Item, title string) (items.Item, bool) {
	for _, it := range list {
		if it.Title == title {
			return it, true
		}
	}
	return items.Item{}, false
}
