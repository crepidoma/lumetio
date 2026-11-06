// Package gui は、画面から呼ぶ操作を Wails のサービスとして公開する。
// 決まりはすべて app にあり、ここは日付の受け渡しと結果の形を変えるだけにする。
package gui

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/durfmt"
	"github.com/crepidoma/lumetio/internal/localconf"
	"github.com/crepidoma/lumetio/internal/watch"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// Service は画面から呼ぶ操作。
// 書き込む操作は、書いた後のその日、プロジェクト、左ペインの一覧 (Opened) を返す。画面が読み直しのために、もう一度呼ばなくて済むようにするため。
type Service struct {
	// current は今のデータフォルダへの操作。設定画面で場所を変えると入れ替わるので、app で取り出して使う。
	current atomic.Pointer[app.App]
	// startErr は、起動時にローカル設定を読めなかったときの誤り。どの操作もこれを返す。
	startErr error
	env      Env

	mu sync.Mutex
	// candidates は検索で探せるタスク。すべての日次ファイルを読んで作るので、検索を始めるときに 1 回だけ作り、打つたびに使い回す。
	candidates []app.Candidate
	// panes は今のペインの幅。保存するたびに読み直さなくて済むよう、覚えておく。
	panes localconf.Panes
	// locations は今のデータフォルダと出力先。
	locations Locations
	// watcher は今のデータフォルダの監視。始めていなければ nil。
	watcher *watch.Watcher
}

// Env は、サービスが画面の外とやり取りするためのもの。テストで差し替えられるよう、外から渡す。
type Env struct {
	// Open は、ファイルを OS の標準のアプリで開く。
	Open func(path string) error
	// Local はペインの幅を書く local.yaml。nil なら書かない (ローカル設定を読めなかったときなど)。
	Local *localconf.File
	// Panes は、起動したときに local.yaml に書かれていたペインの幅。なければ nil。
	Panes *localconf.Panes
	// Locations は、起動したときのデータフォルダと出力先。
	Locations Locations
	// DefaultLocations は、データフォルダと出力先の初期値。分からなければ空。
	DefaultLocations Locations
	// ChooseFolder は、OS のフォルダを選ぶダイアログを出し、選んだフォルダを返す。やめたら空を返す。
	ChooseFolder func(title, current string) (string, error)
	// ChooseCSV は、OS のファイルを選ぶダイアログで CSV を選ばせる。やめたら空を返す。
	ChooseCSV func() (string, error)
	// Notify は、データフォルダのファイルが変わったことを画面に知らせる。nil なら監視しない。
	Notify func(watch.Change)
}

// New は、a を操作するサービスを作る。
// startErr が nil でなければ、どの操作もそれを返す。
func New(a *app.App, startErr error, env Env) *Service {
	s := &Service{startErr: startErr, env: env, locations: env.Locations}
	s.current.Store(a)
	if env.Panes != nil {
		s.panes = *env.Panes
	}
	return s
}

func (s *Service) app() *app.App {
	return s.current.Load()
}

// errorCause は、画面でエラーの種類を見分けるために、エラーと一緒に渡す値。
type errorCause struct {
	// Kind は input (入力の誤り)、conflict (他でファイルが変わった)、internal (読み書きの失敗など)。
	Kind string `json:"kind"`
}

// MarshalError は、エラーの種類を画面に渡す。
// 入力の誤りは入力欄に、衝突と内部の失敗は画面の端の知らせに出し分けるため。
func MarshalError(err error) []byte {
	kind := "internal"
	switch {
	case app.IsInputError(err):
		kind = "input"
	case errors.Is(err, app.ErrConflict):
		kind = "conflict"
	}
	data, _ := json.Marshal(errorCause{Kind: kind})
	return data
}

func (s *Service) date(str string) (time.Time, error) {
	if s.startErr != nil {
		return time.Time{}, s.startErr
	}
	d, err := time.Parse(yamlfile.DateLayout, str)
	if err != nil {
		// 画面は日付を YYYY-MM-DD で渡すので、ここに来るのは画面の不具合。
		return time.Time{}, err
	}
	return d, nil
}

// Today は今日の日付 (YYYY-MM-DD) を返す。
func (s *Service) Today() (string, error) {
	if s.startErr != nil {
		return "", s.startErr
	}
	return s.app().Today().Format(yamlfile.DateLayout), nil
}

// Opened は、日を開いたときに画面に出すものすべて。
// 書き込む操作も、書いた後のこれを返す。画面が日、プロジェクト、左ペインの一覧を、別々に読み直さなくて済むようにするため。
type Opened struct {
	Day Day `json:"day"`
	// Projects は、その日に使えるプロジェクト。
	Projects []Project `json:"projects"`
	// Tags は、タグの一覧。タグを入力するときの候補。
	Tags  []string  `json:"tags"`
	Tasks TaskLists `json:"tasks"`
}

// Open は、その日を開いたときに画面に出すものを返す。
func (s *Service) Open(date string) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	return s.open(d)
}

func (s *Service) open(d time.Time) (Opened, error) {
	o, err := s.app().Open(d)
	if err != nil {
		return Opened{}, err
	}
	out := Opened{Day: newDay(o.Day), Projects: []Project{}, Tags: append([]string{}, o.Tags...), Tasks: newTaskLists(o.Tasks)}
	for _, p := range o.Projects {
		out.Projects = append(out.Projects, Project{Code: p.Code, Name: p.Name})
	}
	return out, nil
}

// SetFavorite は、タスクをよく使うタスクに入れる、または外し、書いた後のその日を返す。
func (s *Service) SetFavorite(date, title string, favorite bool) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	if _, err := s.app().SetFavorite(d, title, favorite); err != nil {
		return Opened{}, err
	}
	return s.open(d)
}

// SearchResult は検索の結果 1 つ。
type SearchResult struct {
	Title   string   `json:"title"`
	Project *Project `json:"project"`
	// LastDate は最後に記録した日 (YYYY-MM-DD)。記録したことがなければ空。
	LastDate string `json:"lastDate"`
	// InItems は、items.yaml にあるタスクか。タスクの編集を開けるのは、items.yaml にあるタスクだけ。
	InItems bool `json:"inItems"`
}

// StartSearch は、検索で探せるタスクの一覧を作り直す。検索欄に入ったときに呼ぶ。
// 読めなかった日次ファイルの知らせを返す。
func (s *Service) StartSearch() ([]string, error) {
	if s.startErr != nil {
		return nil, s.startErr
	}
	list, skipped, err := s.app().Candidates()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.candidates = list
	s.mu.Unlock()
	return nonNil(skipped), nil
}

// Search は、StartSearch で作った一覧から、query を含むタスクを返す。
func (s *Service) Search(query string) []SearchResult {
	s.mu.Lock()
	found := search(s.candidates, query)
	s.mu.Unlock()
	out := []SearchResult{}
	for _, c := range found {
		r := SearchResult{Title: c.Title, Project: projectOf(c.Project), InItems: c.InItems}
		if !c.LastDate.IsZero() {
			r.LastDate = c.LastDate.Format(yamlfile.DateLayout)
		}
		out = append(out, r)
	}
	return out
}

// PreviewTime は、入力中の時間の書き方を解釈した結果を返す。
// index が -1 なら title の記録に足すとき、0 以上なら index 番目の時間を直すとき。
func (s *Service) PreviewTime(date, title string, index int, input string) ([]TimeEntry, error) {
	d, err := s.date(date)
	if err != nil {
		return nil, err
	}
	f, err := s.durFormat()
	if err != nil {
		return nil, err
	}
	es, err := s.app().PreviewTime(d, title, index, input)
	if err != nil {
		return nil, err
	}
	return timeEntries(es, f), nil
}

// durFormat は、settings.yaml の長さの表記を読む。
func (s *Service) durFormat() (durfmt.Format, error) {
	layout, err := s.app().DurationFormat()
	if err != nil {
		return durfmt.Format{}, err
	}
	return durationFormat(layout), nil
}

// AddRecords は、その日の記録にタスクを足す。
// 予定の実施済と、記録の行の時間の入力欄からの追加にも使う。
func (s *Service) AddRecords(date string, records []NewRecord) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	var recs []app.NewRecord
	for _, r := range records {
		recs = append(recs, app.NewRecord{Title: r.Title, Time: r.Time, Project: r.Project.toDayfile(), Tags: r.Tags, Note: r.Note, Exclude: r.Exclude})
	}
	if _, err := s.app().AddRecords(d, recs); err != nil {
		return Opened{}, err
	}
	return s.open(d)
}

// UpdateRecord は、その日の記録を直す。
func (s *Service) UpdateRecord(date, title string, patch RecordPatch) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	if _, err := s.app().UpdateRecord(d, title, app.RecordPatch{
		Title:   patch.Title,
		Time:    patch.Time,
		Untimed: patch.Untimed,
		Project: patch.Project.toDayfile(),
		Tags:    patch.Tags,
		Note:    patch.Note,
		Exclude: patch.Exclude,
		Done:    patch.Done,
	}); err != nil {
		return Opened{}, err
	}
	return s.open(d)
}

// ReplaceTime は、記録の index 番目の時間を input で置き換える。空なら消す。
// current は、画面で見ていたその時間の保存形式 (TimeEntry.Value)。
func (s *Service) ReplaceTime(date, title string, index int, current, input string) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	if _, err := s.app().ReplaceTime(d, title, index, current, input); err != nil {
		return Opened{}, err
	}
	return s.open(d)
}

// RemoveRecord は、その日の記録を消す。
func (s *Service) RemoveRecord(date, title string) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	if err := s.app().RemoveRecord(d, title); err != nil {
		return Opened{}, err
	}
	return s.open(d)
}

// SkipPlans は、その日の予定をスキップする。
func (s *Service) SkipPlans(date string, titles []string) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	if err := s.app().SkipPlans(d, titles); err != nil {
		return Opened{}, err
	}
	return s.open(d)
}

// SetText は、テキストのメモ欄に書く。
// 画面は欄ごとに書くので、値の型ごとに分けて、型の決まった引数で受け取る。
// current は、打ち始めたときに画面で見ていた値。ファイルの値と違えば、書かずに ErrConflict を返す。
func (s *Service) SetText(date, name, current, value string) (Opened, error) {
	return s.setNote(date, app.NoteValue{Name: name, Value: value, Current: &current})
}

// SetBool は、真偽値のメモ欄に書く。
func (s *Service) SetBool(date, name string, value bool) (Opened, error) {
	return s.setNote(date, app.NoteValue{Name: name, Value: value})
}

func (s *Service) setNote(date string, nv app.NoteValue) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	if err := s.app().SetNotes(d, []app.NoteValue{nv}); err != nil {
		return Opened{}, err
	}
	return s.open(d)
}

func (p *Project) toDayfile() *dayfile.Project {
	if p == nil {
		return nil
	}
	return &dayfile.Project{Code: p.Code, Name: p.Name}
}

// ReorderRecords は、その日の記録を titles の順に並べ替え、並べ替えた後のその日を返す。
func (s *Service) ReorderRecords(date string, titles []string) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	if err := s.app().ReorderRecords(d, titles); err != nil {
		return Opened{}, err
	}
	return s.open(d)
}

// ReorderFavorites は、よく使うタスクを titles の順に並べ替え、書いた後のその日を返す。
func (s *Service) ReorderFavorites(date string, titles []string) (Opened, error) {
	d, err := s.date(date)
	if err != nil {
		return Opened{}, err
	}
	if err := s.app().ReorderFavorites(titles); err != nil {
		return Opened{}, err
	}
	return s.open(d)
}
