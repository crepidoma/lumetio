package app

import (
	"fmt"
	"slices"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// ReorderRecords は、その日の記録を titles の順に並べ替える。
// titles は、その日の記録の名前をすべて、新しい順に並べたもの。
// 画面に出していた記録とファイルの記録が食い違うときは、他で変わったものとして書かない。
func (a *App) ReorderRecords(date time.Time, titles []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	day, v, err := a.store.LoadDay(date)
	if err != nil {
		return err
	}
	ordered, ok := reorder(day.Tasks, titles, func(t dayfile.Task) string { return t.Title })
	if !ok {
		return fmt.Errorf("%s の記録が変わっています: %w", yamlfile.DateOnly(date).Format(yamlfile.DateLayout), ErrConflict)
	}
	// 並びが変わらなければ書かない。記録のない日のファイルを作らないため。
	if sameOrder(ordered, day.Tasks, func(t dayfile.Task) string { return t.Title }) {
		return nil
	}
	day.Tasks = ordered
	_, err = a.store.SaveDay(day, v)
	return err
}

// ReorderFavorites は、よく使うタスクを titles の順に並べ替える。
// よく使うタスクどうしの順だけを入れ替え、ほかのタスクの位置は変えない。
// 予定は items.yaml の順に出るので、予定の並びまで変わらないようにするため。
func (a *App) ReorderFavorites(titles []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	list, v, err := a.store.LoadItems()
	if err != nil {
		return err
	}
	var slots []int
	var favorites []items.Item
	for i, it := range list {
		if it.Favorite {
			slots = append(slots, i)
			favorites = append(favorites, it)
		}
	}
	ordered, ok := reorder(favorites, titles, func(it items.Item) string { return it.Title })
	if !ok {
		return fmt.Errorf("よく使うタスクが変わっています: %w", ErrConflict)
	}
	// 並びが変わらなければ書かない。items.yaml がないときに作らないため。
	if sameOrder(ordered, favorites, func(it items.Item) string { return it.Title }) {
		return nil
	}
	for i, slot := range slots {
		list[slot] = ordered[i]
	}
	_, err = a.store.SaveItems(list, v)
	return err
}

// sameOrder は、a と b が同じ名前の順に並んでいるかを返す。
func sameOrder[T any](a, b []T, title func(T) string) bool {
	return slices.EqualFunc(a, b, func(x, y T) bool { return title(x) == title(y) })
}

// reorder は、list を titles の順に並べ替える。
// titles が list の名前をちょうど 1 回ずつ含んでいなければ、ok を false にする。
func reorder[T any](list []T, titles []string, title func(T) string) (ordered []T, ok bool) {
	if len(list) != len(titles) {
		return nil, false
	}
	for _, t := range titles {
		i := slices.IndexFunc(list, func(v T) bool { return title(v) == cleanName(t) })
		if i < 0 || slices.ContainsFunc(ordered, func(v T) bool { return title(v) == cleanName(t) }) {
			return nil, false
		}
		ordered = append(ordered, list[i])
	}
	return ordered, true
}
