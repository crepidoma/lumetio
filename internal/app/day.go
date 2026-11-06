package app

import (
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/effort"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/memo"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/store"
	"github.com/crepidoma/lumetio/internal/tasklist"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// DayView は、1 日を開いたときに見せるもの。
type DayView struct {
	Day dayfile.Day
	// Exists は、その日の日次ファイルがあるか。
	Exists bool
	Plans  []items.Item
	// Notes は画面に出すメモ欄の値。ファイルがなければ、作るときに書く値。
	Notes    []dayfile.Note
	NoteDefs []settings.NoteDef
	Summary  effort.Summary
	// Warnings は、期間の外のプロジェクトなどの警告。
	Warnings []string
	// OutOfPeriod は、記録にある期間の外のプロジェクトのコード。画面で記録の行に印を付けるため。
	OutOfPeriod []string
	// DurationFormat は、画面で長さを出すときの書式。
	DurationFormat string
	// Skipped は、読めずに計算から除いたファイルの知らせ。
	Skipped []string
}

// Opened は、日を開いたときに画面に出すものすべて。
type Opened struct {
	Day DayView
	// Projects は、その日に使えるプロジェクト。
	Projects []settings.Project
	// Tags は、タグの一覧。タグを入力するときの候補。
	Tags  []string
	Tasks TaskLists
}

// Open は、その日を開いたときに画面に出すものを、まとめて返す。
// 設定と items.yaml は 1 回だけ読む。日を開くたびや書くたびに、同じファイルを何度も読まないため。
// 日を開いただけではファイルを作らない。
func (a *App) Open(date time.Time) (Opened, error) {
	s, err := a.loadState()
	if err != nil {
		return Opened{}, err
	}
	view, err := a.dayView(date, s)
	if err != nil {
		return Opened{}, err
	}
	return Opened{Day: view, Projects: projectsOn(date, s.settings.Projects), Tags: s.settings.Tags, Tasks: a.taskLists(date, s)}, nil
}

// Day は、その日を開いたときに見せるものを返す。
// 日を開いただけではファイルを作らない。
func (a *App) Day(date time.Time) (DayView, error) {
	s, err := a.loadState()
	if err != nil {
		return DayView{}, err
	}
	return a.dayView(date, s)
}

func (a *App) dayView(date time.Time, s state) (DayView, error) {
	day, v, err := a.store.LoadDay(date)
	if err != nil {
		return DayView{}, err
	}
	out := outOfPeriodProjects(day, s.settings.Projects)
	view := DayView{
		Day:            day,
		Exists:         v.Exists(),
		Plans:          tasklist.Plans(day, s.items),
		NoteDefs:       s.settings.Notes,
		Summary:        effort.Summarize(day),
		Warnings:       append(periodWarnings(out), missingTimeWarnings(day)...),
		DurationFormat: s.settings.DurationFormat,
	}
	for _, p := range out {
		view.OutOfPeriod = append(view.OutOfPeriod, p.Code)
	}
	if view.Exists {
		view.Notes = memo.ForDay(s.settings.Notes, day)
	} else {
		notes, skipped, err := a.newDayNotes(date, s.settings.Notes)
		if err != nil {
			return DayView{}, err
		}
		view.Notes = notes
		view.Skipped = skipped
	}
	return view, nil
}

// newDayNotes は、まだファイルのない日のメモ欄の値を作る。
func (a *App) newDayNotes(date time.Time, defs []settings.NoteDef) ([]dayfile.Note, []string, error) {
	prev, found, skipped, err := a.store.PreviousDay(date)
	if err != nil {
		return nil, nil, err
	}
	var previous *dayfile.Day
	if found {
		previous = &prev
	}
	return memo.ForNewDay(defs, previous), errorStrings(skipped), nil
}

// TaskLists は左ペインのタスクの一覧。
type TaskLists struct {
	Recent    []tasklist.Recent
	Favorites []items.Item
	// RepeatingOn はその日に予定が出る定期的なタスク、RepeatingOff はそれ以外。
	RepeatingOn  []items.Item
	RepeatingOff []items.Item
	// Recorded は、その日にもう記録があるタスクの名前。
	Recorded []string
	Skipped  []string
}

// Tasks は、その日を起点にしたタスクの一覧を返す。
func (a *App) Tasks(date time.Time) (TaskLists, error) {
	s, err := a.loadState()
	if err != nil {
		return TaskLists{}, err
	}
	return a.taskLists(date, s), nil
}

func (a *App) taskLists(date time.Time, s state) TaskLists {
	days, skipped := a.store.LoadDays(date.AddDate(0, 0, -(tasklist.RecentDays-1)), date)
	on, off := tasklist.Repeating(s.items, date)
	lists := TaskLists{
		Recent:       tasklist.RecentTasks(days, s.items, date),
		Favorites:    tasklist.Favorites(s.items),
		RepeatingOn:  on,
		RepeatingOff: off,
		Skipped:      errorStrings(skipped),
	}
	for _, day := range days {
		if day.Date.Equal(yamlfile.DateOnly(date)) {
			for _, t := range day.Tasks {
				lists.Recorded = append(lists.Recorded, t.Title)
			}
		}
	}
	return lists
}

// DurationFormat は、画面で長さを出すときの書式を返す。
func (a *App) DurationFormat() (string, error) {
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return "", err
	}
	return st.DurationFormat, nil
}

// Projects は、その日に使えるプロジェクトを返す。
// 期間は今日ではなく、その日と比べる。
func (a *App) Projects(date time.Time) ([]settings.Project, error) {
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return nil, err
	}
	return projectsOn(date, st.Projects), nil
}

// projectsOn は、その日に期間の中にあるプロジェクトを返す。
func projectsOn(date time.Time, projects []settings.Project) []settings.Project {
	var out []settings.Project
	for _, p := range projects {
		if p.InPeriod(yamlfile.DateOnly(date)) {
			out = append(out, p)
		}
	}
	return out
}

// loadDayForWrite は、書き換えるためにその日を読む。
// ファイルがなければ、作るときに書くメモ欄の値を入れた日を返す。
// メモ欄に触らなかった日も、引き継ぎが途切れないようにするため。
func (a *App) loadDayForWrite(date time.Time, defs []settings.NoteDef) (dayfile.Day, store.Version, error) {
	day, v, err := a.store.LoadDay(date)
	if err != nil {
		return dayfile.Day{}, store.Version{}, err
	}
	if !v.Exists() {
		notes, _, err := a.newDayNotes(date, defs)
		if err != nil {
			return dayfile.Day{}, store.Version{}, err
		}
		day.Notes = notes
	}
	return day, v, nil
}
