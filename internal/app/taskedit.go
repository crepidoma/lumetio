package app

import (
	"bytes"
	"fmt"
	"slices"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/timeexpr"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// TaskInput は、タスクの編集で保存する値。
type TaskInput struct {
	Title   string
	Project *dayfile.Project
	Tags    []string
	Exclude bool
	// Time は、いつもの時間の入力欄の文字列。
	Time     string
	Repeat   *items.Repeat
	Favorite bool
}

// Items は、items.yaml のタスクを、items.yaml の順に返す。
func (a *App) Items() ([]items.Item, error) {
	list, _, err := a.store.LoadItems()
	return list, err
}

// Item は、タスクの編集に出す items.yaml のタスクを返す。
// タスクの編集で直すのは items.yaml にあるタスクだけなので、ないタスクは入力の誤りにする (他で消された後に開いたときなど)。
func (a *App) Item(title string) (items.Item, error) {
	title = cleanName(title)
	list, _, err := a.store.LoadItems()
	if err != nil {
		return items.Item{}, err
	}
	i := slices.IndexFunc(list, func(it items.Item) bool { return it.Title == title })
	if i < 0 {
		return items.Item{}, inputErrorf("%s の設定はありません", title)
	}
	return list[i], nil
}

// SaveTask は、タスクの編集で直した値を items.yaml に書く。
// original は直す前のタスク名で、新しく作るときは空にする。
// base は編集を開いたときに読んだ値で、新しく作るときは nil にする。
// 開いているあいだに他でそのタスクが変わっていたら、他の変更を上書きしないよう、書かずに ErrConflict を返す。
// 設定 (よく使う、いつもの時間、繰り返し) がなくなれば、items.yaml から消す。
func (a *App) SaveTask(original string, base *items.Item, in TaskInput) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return err
	}
	list, v, err := a.store.LoadItems()
	if err != nil {
		return err
	}

	title := cleanName(in.Title)
	if title == "" {
		return inputErrorf("タスク名がありません")
	}
	original = cleanName(original)
	self := -1
	if original != "" {
		self = slices.IndexFunc(list, func(it items.Item) bool { return it.Title == original })
		if self < 0 {
			return fmt.Errorf("%s の設定が他で消されています: %w", original, ErrConflict)
		}
		if base == nil || !sameItem(list[self], *base) {
			return fmt.Errorf("%s の設定が他で変わっています: %w", original, ErrConflict)
		}
	}
	for i, it := range list {
		if i != self && it.Title == title {
			return inputErrorf("%s はもうあります", title)
		}
	}
	project, err := checkProject(in.Project, st.Projects)
	if err != nil {
		return err
	}
	usual, err := timeexpr.Parse(in.Time, timeexpr.Context{Detached: true})
	if err != nil {
		return err
	}
	if in.Repeat != nil {
		if err := checkRepeat(*in.Repeat); err != nil {
			return err
		}
	}
	it := items.Item{
		Title:    title,
		Project:  project,
		Tags:     cleanTags(in.Tags),
		Exclude:  in.Exclude,
		Time:     usual,
		Repeat:   in.Repeat,
		Favorite: in.Favorite,
	}

	switch {
	case !it.HasSettings() && self < 0:
		// 設定のないタスクは items.yaml に書かない。新しく作るのに設定がなければ、作るものがない。
		return inputErrorf("よく使う、いつもの時間、繰り返しのどれかを付けます")
	case !it.HasSettings():
		list = slices.Delete(list, self, self+1)
	case self >= 0:
		list[self] = it
	default:
		list = append(list, it)
	}

	// 画面で作れない組み合わせが紛れ込んでいたら、読めない items.yaml を書く前に止める。画面の不具合なので、入力の誤りにはしない。
	data, err := items.Marshal(list)
	if err != nil {
		return err
	}
	if _, err := items.Parse(data); err != nil {
		return fmt.Errorf("items.yaml に書けない値です: %w", err)
	}
	_, err = a.store.SaveItems(list, v)
	return err
}

// sameItem は、2 つのタスクの設定が、items.yaml に書いたときに同じになるかを返す。
// 空の並びと nil のような、書けば同じになる違いで衝突にしないため、書いた形で比べる。
func sameItem(a, b items.Item) bool {
	x, errX := items.Marshal([]items.Item{a})
	y, errY := items.Marshal([]items.Item{b})
	return errX == nil && errY == nil && bytes.Equal(x, y)
}

// checkRepeat は、画面で選び終えていない繰り返しを入力の誤りにする。
func checkRepeat(r items.Repeat) error {
	switch {
	case r.Freq == items.Weekly && len(r.ByDay) == 0:
		return inputErrorf("曜日を選びます")
	case r.Freq == items.Monthly && len(r.ByDay) == 0 && r.ByMonthDay == 0:
		return inputErrorf("日を選びます")
	case r.Interval == 2 && r.Start.IsZero():
		return inputErrorf("隔週では、いつからを入れます")
	case !r.Start.IsZero() && !r.Until.IsZero() && r.Until.Before(r.Start):
		return inputErrorf("いつまでが、いつからより前です")
	}
	return nil
}

// CompleteTask は、開いている日までで、そのタスクを最後に記録した日の記録に完了の印を付ける。
// その日より後に記録があれば、そちらが最後の記録になるので、完了としては扱われない。
func (a *App) CompleteTask(date time.Time, title string) (warnings []string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	title = cleanName(title)
	last, on, found, skipped, err := a.lastRecord(title, yamlfile.DateOnly(date).AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, inputErrorf("%s の記録がありません", title)
	}
	if last.Done {
		return skipped, nil
	}
	day, v, i, err := a.loadRecord(on, title)
	if err != nil {
		return nil, err
	}
	day.Tasks[i].Done = true
	if _, err := a.store.SaveDay(day, v); err != nil {
		return nil, err
	}
	return skipped, nil
}

// PreviewUsualTime は、いつもの時間の入力を解釈した結果を返す。打っているあいだに見せるため。
func (a *App) PreviewUsualTime(input string) ([]timeexpr.Entry, error) {
	return timeexpr.Parse(input, timeexpr.Context{Detached: true})
}
