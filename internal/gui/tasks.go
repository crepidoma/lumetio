package gui

import (
	"slices"
	"strings"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/items"
	"golang.org/x/text/unicode/norm"
)

// Task は左ペインや検索の結果に出すタスク 1 つ。
type Task struct {
	Title   string   `json:"title"`
	Project *Project `json:"project"`
	// Recorded は、開いている日にもう記録があること。
	Recorded bool `json:"recorded"`
	Favorite bool `json:"favorite"`
}

// TaskLists は左ペインのタスクの一覧。
type TaskLists struct {
	Recent    []Task `json:"recent"`
	Favorites []Task `json:"favorites"`
	// RepeatingOn は開いている日に予定が出る定期的なタスク、RepeatingOff はそれ以外。
	RepeatingOn  []Task `json:"repeatingOn"`
	RepeatingOff []Task `json:"repeatingOff"`
	// Notices は、読めずに除いたファイルの知らせ。
	Notices []string `json:"notices"`
}

func newTaskLists(l app.TaskLists) TaskLists {
	favorite := map[string]bool{}
	for _, it := range l.Favorites {
		favorite[it.Title] = true
	}
	task := func(title string, p *Project) Task {
		return Task{Title: title, Project: p, Recorded: slices.Contains(l.Recorded, title), Favorite: favorite[title]}
	}
	fromItems := func(list []items.Item) []Task {
		out := []Task{}
		for _, it := range list {
			out = append(out, task(it.Title, projectOf(it.Project)))
		}
		return out
	}
	out := TaskLists{
		Recent:       []Task{},
		Favorites:    fromItems(l.Favorites),
		RepeatingOn:  fromItems(l.RepeatingOn),
		RepeatingOff: fromItems(l.RepeatingOff),
		Notices:      nonNil(l.Skipped),
	}
	// 同じタスクが 2 か所に並ばないよう、直近のタスクからは、よく使うタスクとその日に予定の出るタスクを除く。
	shown := map[string]bool{}
	for _, t := range slices.Concat(out.Favorites, out.RepeatingOn) {
		shown[t.Title] = true
	}
	for _, r := range l.Recent {
		if !shown[r.Last.Title] {
			out.Recent = append(out.Recent, task(r.Last.Title, projectOf(r.Base.Project)))
		}
	}
	return out
}

// maxResults は、検索の結果に出す数。打ち始めは多く当たるので、見て選べる数に絞る。
const maxResults = 30

// fold は、検索で比べるために文字をそろえる。
// 全角と半角は NFKC で、大文字と小文字は小文字にそろえる。ひらがなとカタカナはそろえない。
func fold(s string) string {
	return strings.ToLower(norm.NFKC.String(s))
}

// search は、candidates から query を含むタスクを、並びを保ったまま返す。
func search(candidates []app.Candidate, query string) []app.Candidate {
	q := fold(strings.TrimSpace(query))
	var out []app.Candidate
	for _, c := range candidates {
		if strings.Contains(fold(c.Title), q) {
			out = append(out, c)
			if len(out) == maxResults {
				break
			}
		}
	}
	return out
}
