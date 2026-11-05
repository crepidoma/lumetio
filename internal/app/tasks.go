package app

import (
	"slices"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/tasklist"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// SetFavorite は、タスクをよく使うタスクに入れる、または外す。
// まだ items.yaml にないタスクを入れるときは、その日までで一番新しい記録からプロジェクトなどを写す。
// 外して設定が何も残らなければ、items.yaml から消える。
func (a *App) SetFavorite(date time.Time, title string, favorite bool) (warnings []string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	title = cleanName(title)
	if title == "" {
		return nil, inputErrorf("タスク名がありません")
	}
	list, v, err := a.store.LoadItems()
	if err != nil {
		return nil, err
	}
	if i := slices.IndexFunc(list, func(it items.Item) bool { return it.Title == title }); i >= 0 {
		list[i].Favorite = favorite
	} else {
		if !favorite {
			return nil, nil
		}
		// その日の記録も写す元に入れるため、次の日より前を探す。
		last, _, found, skipped, err := a.lastRecord(title, yamlfile.DateOnly(date).AddDate(0, 0, 1))
		if err != nil {
			return nil, err
		}
		warnings = skipped
		it := items.Item{Title: title}
		if found {
			it = tasklist.NewItem(last)
		}
		it.Favorite = true
		list = append(list, it)
	}
	if _, err := a.store.SaveItems(list, v); err != nil {
		return nil, err
	}
	return warnings, nil
}

// Candidate は、検索で探せるタスク 1 つ。
type Candidate struct {
	Title string
	// LastDate は最後に記録した日。記録したことがなければゼロ値。
	LastDate time.Time
	// Project は、最後の記録のプロジェクト。記録がなければ items.yaml の値。
	Project dayfile.Project
	// InItems は、items.yaml にあるタスクか。タスクの編集を開けるのは、items.yaml にあるタスクだけのため。
	InItems bool
}

// Candidates は、検索で探せるタスクを、最後に記録した日が新しい順に返す。
// 記録したことのない items.yaml のタスクは、その後に items.yaml の順で並べる。
// すべての日次ファイルを読むので、打つたびではなく、検索を始めるときに 1 回だけ呼ぶ。
func (a *App) Candidates() (list []Candidate, skipped []string, err error) {
	seen := map[string]bool{}
	all := func(time.Time) bool { return true }
	skipped, err = a.eachDayNewestFirst(all, func(d time.Time, day dayfile.Day) bool {
		for _, t := range day.Tasks {
			if seen[t.Title] {
				continue
			}
			seen[t.Title] = true
			list = append(list, Candidate{Title: t.Title, LastDate: d, Project: t.Project})
		}
		return false
	})
	if err != nil {
		return nil, nil, err
	}
	its, _, err := a.store.LoadItems()
	if err != nil {
		return nil, nil, err
	}
	inItems := map[string]bool{}
	for _, it := range its {
		inItems[it.Title] = true
		if !seen[it.Title] {
			seen[it.Title] = true
			list = append(list, Candidate{Title: it.Title, Project: it.Project})
		}
	}
	for i := range list {
		list[i].InItems = inItems[list[i].Title]
	}
	return list, skipped, nil
}
