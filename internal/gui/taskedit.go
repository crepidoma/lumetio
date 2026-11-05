package gui

import (
	"fmt"
	"strings"
	"time"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/timeexpr"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// Repeat はタスクの繰り返し。値の書き方は items.yaml と同じにする。
type Repeat struct {
	// Freq は daily、weekly、monthly。
	Freq     string `json:"freq"`
	Interval int    `json:"interval"`
	// ByDay は mo、2mo、-1fr のような曜日。
	ByDay      []string `json:"byDay"`
	ByMonthDay int      `json:"byMonthDay"`
	// Start と Until は YYYY-MM-DD。なければ空。
	Start string `json:"start"`
	Until string `json:"until"`
}

// TaskEdit は、タスクの編集で開いたときの値と、保存する値。どちらも同じ形なので 1 つの型にする。
type TaskEdit struct {
	Title   string   `json:"title"`
	Project *Project `json:"project"`
	Tags    []string `json:"tags"`
	Exclude bool     `json:"exclude"`
	// Time は、いつもの時間を入力欄に入れる形で書いたもの。
	Time     string  `json:"time"`
	Repeat   *Repeat `json:"repeat"`
	Favorite bool    `json:"favorite"`
}

// ManagedTask は、タスクの管理に並べる、items.yaml のタスク 1 つ。
type ManagedTask struct {
	Title    string      `json:"title"`
	Project  *Project    `json:"project"`
	Favorite bool        `json:"favorite"`
	Time     []TimeEntry `json:"time"`
	Repeat   *Repeat     `json:"repeat"`
}

func repeatOf(r *items.Repeat) *Repeat {
	if r == nil {
		return nil
	}
	out := &Repeat{Freq: string(r.Freq), Interval: r.Interval, ByDay: []string{}, ByMonthDay: r.ByMonthDay}
	for _, d := range r.ByDay {
		out.ByDay = append(out.ByDay, d.String())
	}
	if !r.Start.IsZero() {
		out.Start = r.Start.Format(yamlfile.DateLayout)
	}
	if !r.Until.IsZero() {
		out.Until = r.Until.Format(yamlfile.DateLayout)
	}
	return out
}

// toItems は、画面で選んだ繰り返しを items の形にする。
// 画面は選択肢からしか作らないので、読めない値は画面の不具合として内部の誤りにする。
func (r *Repeat) toItems() (*items.Repeat, error) {
	if r == nil {
		return nil, nil
	}
	out := &items.Repeat{Freq: items.Freq(r.Freq), Interval: r.Interval, ByMonthDay: r.ByMonthDay}
	if out.Interval == 0 {
		out.Interval = 1
	}
	for _, s := range r.ByDay {
		d, ok := items.ParseDaySpec(s)
		if !ok {
			return nil, fmt.Errorf("曜日 %q を読めません", s)
		}
		out.ByDay = append(out.ByDay, d)
	}
	var err error
	if r.Start != "" {
		if out.Start, err = time.Parse(yamlfile.DateLayout, r.Start); err != nil {
			return nil, err
		}
	}
	if r.Until != "" {
		if out.Until, err = time.Parse(yamlfile.DateLayout, r.Until); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// toItem は、TaskEdit で渡した値を items の形に戻す。
// 画面は受け取った値をそのまま返すので、戻せなければ画面の不具合として内部の誤りにする。
func (e *TaskEdit) toItem() (items.Item, error) {
	r, err := e.Repeat.toItems()
	if err != nil {
		return items.Item{}, err
	}
	usual, err := timeexpr.Parse(e.Time, timeexpr.Context{Detached: true})
	if err != nil {
		return items.Item{}, fmt.Errorf("いつもの時間 %q を戻せません: %w", e.Time, err)
	}
	it := items.Item{Title: e.Title, Tags: e.Tags, Exclude: e.Exclude, Time: usual, Repeat: r, Favorite: e.Favorite}
	if e.Project != nil {
		it.Project = *e.Project.toDayfile()
	}
	return it, nil
}

// editableProject は、直す欄に入れるプロジェクト。
// 名前がないときにコードを名前として埋めると、保存したときに名前として書いてしまうので、そのまま渡す。
func editableProject(p dayfile.Project) *Project {
	if p.Code == "" {
		return nil
	}
	return &Project{Code: p.Code, Name: p.Name}
}

// usualInput は、いつもの時間を入力欄に入れる形にする。
func usualInput(es []timeexpr.Entry) string {
	var parts []string
	for _, e := range es {
		parts = append(parts, entryInput(e))
	}
	return strings.Join(parts, " ")
}

// TaskEdit は、タスクの編集に出す items.yaml のタスクの値を返す。
func (s *Service) TaskEdit(title string) (TaskEdit, error) {
	if s.startErr != nil {
		return TaskEdit{}, s.startErr
	}
	it, err := s.app().Item(title)
	if err != nil {
		return TaskEdit{}, err
	}
	return taskEditOf(it), nil
}

func taskEditOf(it items.Item) TaskEdit {
	return TaskEdit{
		Title:    it.Title,
		Project:  editableProject(it.Project),
		Tags:     nonNil(it.Tags),
		Exclude:  it.Exclude,
		Time:     usualInput(it.Time),
		Repeat:   repeatOf(it.Repeat),
		Favorite: it.Favorite,
	}
}

// SaveTask は、タスクの編集で直した値を書く。original は直す前のタスク名で、新しく作るときは空にする。
// base は編集を開いたときに TaskEdit で受け取った値で、新しく作るときは nil にする。開いているあいだに他で変わっていないかを確かめるため。
// 予定や左ペインの一覧も変わるので、画面は書いた後にその日を読み直す。
func (s *Service) SaveTask(original string, base *TaskEdit, in TaskEdit) error {
	if s.startErr != nil {
		return s.startErr
	}
	r, err := in.Repeat.toItems()
	if err != nil {
		return err
	}
	var baseItem *items.Item
	if base != nil {
		it, err := base.toItem()
		if err != nil {
			return err
		}
		baseItem = &it
	}
	return s.app().SaveTask(original, baseItem, app.TaskInput{
		Title:    in.Title,
		Project:  in.Project.toDayfile(),
		Tags:     in.Tags,
		Exclude:  in.Exclude,
		Time:     in.Time,
		Repeat:   r,
		Favorite: in.Favorite,
	})
}

// CompleteTask は、開いている日までで、そのタスクを最後に記録した日の記録に完了の印を付ける。
// 読めずに除いたファイルの知らせを返す。
func (s *Service) CompleteTask(date, title string) ([]string, error) {
	d, err := s.date(date)
	if err != nil {
		return nil, err
	}
	skipped, err := s.app().CompleteTask(d, title)
	if err != nil {
		return nil, err
	}
	return nonNil(skipped), nil
}

// ManagedTasks は、タスクの管理に並べる items.yaml のタスクを返す。
func (s *Service) ManagedTasks() ([]ManagedTask, error) {
	if s.startErr != nil {
		return nil, s.startErr
	}
	f, err := s.durFormat()
	if err != nil {
		return nil, err
	}
	list, err := s.app().Items()
	if err != nil {
		return nil, err
	}
	out := []ManagedTask{}
	for _, it := range list {
		out = append(out, ManagedTask{
			Title:    it.Title,
			Project:  projectOf(it.Project),
			Favorite: it.Favorite,
			Time:     timeEntries(it.Time, f),
			Repeat:   repeatOf(it.Repeat),
		})
	}
	return out, nil
}

// PreviewUsualTime は、入力中のいつもの時間を解釈した結果を返す。
func (s *Service) PreviewUsualTime(input string) ([]TimeEntry, error) {
	if s.startErr != nil {
		return nil, s.startErr
	}
	f, err := s.durFormat()
	if err != nil {
		return nil, err
	}
	es, err := s.app().PreviewUsualTime(input)
	if err != nil {
		return nil, err
	}
	return timeEntries(es, f), nil
}
