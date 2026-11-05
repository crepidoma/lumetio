// Package app は、画面と MCP の両方から使う操作をまとめる。
// ファイルを読み、決まりに沿って変え、他で変わっていなければ保存する。
// どちらから操作しても、ファイルを作るタイミングやメモ欄の引き継ぎが同じになるようにするため。
package app

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/store"
	"github.com/crepidoma/lumetio/internal/timeexpr"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// App は 1 つのデータフォルダへの操作。
type App struct {
	// mu は、書き込む操作を 1 つずつ順に行うためのもの。
	// MCP は要求を並行に処理するので、同じプロセスの中で読んでから書くまでの間に、別の書き込みが割り込まないようにする。
	mu    sync.Mutex
	store *store.Store
	// outputDir は出力先のフォルダ。
	outputDir string
	// now は今の時刻を返す。テストで時刻を決めるために差し替える。
	now func() time.Time
}

// New は、st のデータフォルダを操作し、outputDir に出力を書く App を作る。
// now が nil なら time.Now を使う。
func New(st *store.Store, outputDir string, now func() time.Time) *App {
	if now == nil {
		now = time.Now
	}
	return &App{store: st, outputDir: outputDir, now: now}
}

// InputError は、操作の指定の誤り。
// 内部の失敗 (I/O エラーなど) と区別して、画面や AI にそのまま返す。
type InputError struct {
	Message string
}

func (e *InputError) Error() string {
	return e.Message
}

func inputErrorf(format string, args ...any) error {
	return &InputError{Message: fmt.Sprintf(format, args...)}
}

// ErrConflict は、読んだ後に他でファイルが変わったため、書かなかったことを表す。
var ErrConflict = store.ErrConflict

// WrittenHere は、path が、このアプリが最後に書いたときのままかを返す。
// 監視の知らせが、自分の書き込みか、他の PC や MCP の書き込みかを見分けるため。
func (a *App) WrittenHere(path string, since time.Time) bool {
	return a.store.WrittenHere(path, since)
}

// Today は今日の日付を返す。
func (a *App) Today() time.Time {
	return yamlfile.DateOnly(a.now())
}

// isToday は date が今日か。
func (a *App) isToday(date time.Time) bool {
	return yamlfile.DateOnly(date).Equal(a.Today())
}

// minuteOfDay は今の時刻を 0:00 からの分で返す。
func (a *App) minuteOfDay() int {
	n := a.now()
	return n.Hour()*60 + n.Minute()
}

// state は、1 つの操作で使うファイルの中身。
type state struct {
	settings settings.Settings
	items    []items.Item
}

func (a *App) loadState() (state, error) {
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return state{}, err
	}
	list, _, err := a.store.LoadItems()
	if err != nil {
		return state{}, err
	}
	return state{settings: st, items: list}, nil
}

// outOfPeriodWarnings は、その日の記録にある期間の外のプロジェクトを知らせる文を返す。
// 期間は今日ではなく、その日と比べる。
// 一覧にないプロジェクトは期間がないので知らせない。
func outOfPeriodWarnings(day dayfile.Day, projects []settings.Project) []string {
	return periodWarnings(outOfPeriodProjects(day, projects))
}

// periodWarnings は、期間の外のプロジェクトを知らせる文にする。
func periodWarnings(out []settings.Project) []string {
	var warnings []string
	for _, p := range out {
		warnings = append(warnings, fmt.Sprintf("プロジェクト %s の期間 (%s〜%s) の外です", p.Code, yamlfile.FormatDate(p.Start), yamlfile.FormatDate(p.Until)))
	}
	return warnings
}

// outOfPeriodProjects は、その日の記録にある期間の外のプロジェクトを、記録に出てくる順に返す。
func outOfPeriodProjects(day dayfile.Day, projects []settings.Project) []settings.Project {
	var out []settings.Project
	seen := map[string]bool{}
	for _, t := range day.Tasks {
		code := t.Project.Code
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		for _, p := range projects {
			if p.Code == code && !p.InPeriod(day.Date) {
				out = append(out, p)
			}
		}
	}
	return out
}

func errorStrings(errs []error) []string {
	var ss []string
	for _, err := range errs {
		ss = append(ss, err.Error())
	}
	return ss
}

// IsInputError は、err が入力の誤り (時間の書き方の誤りを含む) か。
func IsInputError(err error) bool {
	var ie *InputError
	var te *timeexpr.InputError
	return errors.As(err, &ie) || errors.As(err, &te)
}
