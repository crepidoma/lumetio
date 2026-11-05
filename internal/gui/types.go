package gui

import (
	"fmt"
	"slices"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/durfmt"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/timeexpr"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// Project はプロジェクト。
type Project struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// TimeEntry は時間 1 つ分。
type TimeEntry struct {
	// Label は画面に出す文字。開始-終了は保存の形、長さは長さの表記で書く。
	Label string `json:"label"`
	// Input は、直す欄に入れておく文字 (entryInput)。
	Input string `json:"input"`
	// Value は保存形式 (HH:MM-HH:MM か Nm)。直すときに、どの時間を見ていたかを送るために使う。
	Value string `json:"value"`
}

// Record は、その日の記録 1 つ。
type Record struct {
	Title   string   `json:"title"`
	Project *Project `json:"project"`
	// OutOfPeriod は、プロジェクトがその日には期間の外 (開始前か期限切れ) であること。
	OutOfPeriod bool        `json:"outOfPeriod"`
	Tags        []string    `json:"tags"`
	Note        string      `json:"note"`
	Exclude     bool        `json:"exclude"`
	Done        bool        `json:"done"`
	Times       []TimeEntry `json:"times"`
	// Total は合計を長さの表記で書いたもの。時間不定なら空。
	Total string `json:"total"`
}

// Plan は、まだ記録にしていない予定。
type Plan struct {
	Title   string      `json:"title"`
	Project *Project    `json:"project"`
	Times   []TimeEntry `json:"times"`
}

// Note はメモ欄 1 つ。
type Note struct {
	Name string `json:"name"`
	// Type は値の形 (text か bool)。
	Type string `json:"type"`
	Text string `json:"text"`
	Bool bool   `json:"bool"`
	// Defined は、settings.yaml に定義がある欄か。定義のない欄 (後で消した欄など) は書き換えられない。
	Defined bool `json:"defined"`
}

// Day は、1 日を開いたときに見せるもの。
type Day struct {
	Date string `json:"date"`
	// Exists は、その日の日次ファイルがあるか。
	Exists  bool     `json:"exists"`
	Records []Record `json:"records"`
	Plans   []Plan   `json:"plans"`
	Notes   []Note   `json:"notes"`
	// Total は、計上する時間の合計を長さの表記で書いたもの。
	Total    string   `json:"total"`
	Warnings []string `json:"warnings"`
	// Notices は、読めずに計算から除いたファイルの知らせ。
	Notices []string `json:"notices"`
}

// NewRecord は、記録に足すタスク。nil の項目はタスクの設定や過去の記録から決める。
type NewRecord struct {
	Title   string    `json:"title"`
	Time    *string   `json:"time,omitempty"`
	Project *Project  `json:"project,omitempty"`
	Tags    *[]string `json:"tags,omitempty"`
	Note    *string   `json:"note,omitempty"`
	Exclude *bool     `json:"exclude,omitempty"`
}

// RecordPatch は、記録を直すときの指定。nil の項目は変えない。
type RecordPatch struct {
	Title *string `json:"title,omitempty"`
	// Time は時間の並びをまるごと置き換える。空文字なら時間不定にする。
	Time *string `json:"time,omitempty"`
	// Project のコードが空なら、プロジェクトを外す。
	Project *Project  `json:"project,omitempty"`
	Tags    *[]string `json:"tags,omitempty"`
	Note    *string   `json:"note,omitempty"`
	Exclude *bool     `json:"exclude,omitempty"`
	Done    *bool     `json:"done,omitempty"`
}

// durationFormat は、長さの表記を読む。
// settings.yaml を読むときに確かめているので、失敗したら省略時の表記にする。
func durationFormat(layout string) durfmt.Format {
	_, f, err := settings.ParseDurationFormat(layout)
	if err != nil {
		_, f, _ = settings.ParseDurationFormat("")
	}
	return f
}

// entryInput は、時間を直す欄に入れる文字にする。
// 長さの表記は読めない形にもできるので、必ず時間の書き方として読める形にする。
func entryInput(e timeexpr.Entry) string {
	if e.IsRange {
		return e.String()
	}
	return fmt.Sprintf("%d:%02d", e.Minutes/60, e.Minutes%60)
}

func projectOf(p dayfile.Project) *Project {
	if p.Code == "" {
		return nil
	}
	return &Project{Code: p.Code, Name: p.DisplayName()}
}

func timeEntries(es []timeexpr.Entry, f durfmt.Format) []TimeEntry {
	out := []TimeEntry{}
	for _, e := range es {
		// 開始-終了は保存の形のまま出し、長さは長さの表記で出す。
		label := e.String()
		if !e.IsRange {
			label = f.Format(e.Minutes)
		}
		out = append(out, TimeEntry{Label: label, Input: entryInput(e), Value: e.String()})
	}
	return out
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func newDay(v app.DayView) Day {
	f := durationFormat(v.DurationFormat)
	out := Day{
		Date:     v.Day.Date.Format(yamlfile.DateLayout),
		Exists:   v.Exists,
		Records:  []Record{},
		Plans:    []Plan{},
		Notes:    []Note{},
		Total:    f.Format(v.Summary.TotalMinutes),
		Warnings: nonNil(v.Warnings),
		Notices:  nonNil(v.Skipped),
	}
	for _, t := range v.Summary.Tasks {
		r := Record{
			Title:       t.Title,
			Project:     projectOf(t.Project),
			OutOfPeriod: t.Project.Code != "" && slices.Contains(v.OutOfPeriod, t.Project.Code),
			Tags:        nonNil(t.Tags),
			Note:        t.Note,
			Exclude:     t.Exclude,
			Done:        t.Done,
			Times:       timeEntries(t.Time, f),
		}
		if !t.Untimed {
			r.Total = f.Format(t.Minutes)
		}
		out.Records = append(out.Records, r)
	}
	for _, p := range v.Plans {
		out.Plans = append(out.Plans, Plan{Title: p.Title, Project: projectOf(p.Project), Times: timeEntries(p.Time, f)})
	}
	for _, n := range v.Notes {
		note := Note{Name: n.Name, Type: string(settings.NoteText), Text: n.Text, Bool: n.Bool}
		if n.Kind == dayfile.NoteBool {
			note.Type = string(settings.NoteBool)
		}
		note.Defined = slices.ContainsFunc(v.NoteDefs, func(d settings.NoteDef) bool { return d.Name == n.Name })
		out.Notes = append(out.Notes, note)
	}
	return out
}
