// Package dayfile は日次ファイルの中身を読み書きする。
package dayfile

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/crepidoma/lumetio/internal/timeexpr"
	"github.com/crepidoma/lumetio/internal/yamlfile"
	"go.yaml.in/yaml/v3"
)

// Day はその日の記録。
type Day struct {
	Date    time.Time
	Tasks   []Task
	Skipped []string
	// Notes はファイルに書かれた順を保つ。
	// 過去の日には今の定義にない欄もある。
	Notes []Note
}

// Task は 1 つのタスクの記録。
// プロジェクトやタグは記録した時点の値を写して持つ。
type Task struct {
	Title string
	// Project の Code が空なら、プロジェクトがない。
	Project Project
	Tags    []string
	Note    string
	Exclude bool
	// Done は、そのタスクがこの日で完了したこと。
	// タスクが最後に出てくる記録で true なら、直近のタスクに出さない。
	Done bool
	// Time が空で Untimed でもなければ、時間未入力 (時間の入れ忘れ)。
	Time []timeexpr.Entry
	// Untimed は時間不定。時間を決めず、出勤した時間の残りを割り振る記録。Time とは一緒に持たない。
	Untimed bool
}

// Project はプロジェクト。
// 工数システムへの取り込みはコードで行うので、コードを必須にし、名前は任意にする。
type Project struct {
	Code string
	Name string
}

// DisplayName はプロジェクトの名前。
// 名前がなければコードを使う。
func (p Project) DisplayName() string {
	if p.Name == "" {
		return p.Code
	}
	return p.Name
}

// Note はメモ欄 1 つ分の値。
type Note struct {
	Name string
	// Kind は、日次ファイルに書かれた値の形から決める。
	// 欄の定義を後で消しても、過去の日を正しく読めるようにするため。
	Kind NoteKind
	// Text は Kind が NoteText のときの値。
	Text string
	// Bool は Kind が NoteBool のときの値。
	Bool bool
}

// Empty は、値が空 (空文字か false) か。
// 空の欄は書いていない欄と同じ意味なので、日次ファイルには書かない。
func (n Note) Empty() bool {
	if n.Kind == NoteBool {
		return !n.Bool
	}
	return n.Text == ""
}

// NoteKind はメモ欄の値の形。
type NoteKind int

const (
	NoteText NoteKind = iota
	NoteBool
)

type rawDay struct {
	Date    string    `yaml:"date"`
	Tasks   []rawTask `yaml:"tasks"`
	Skipped []string  `yaml:"skipped"`
	Notes   yaml.Node `yaml:"notes"`
}

type rawTask struct {
	Title   string       `yaml:"title"`
	Project *ProjectYAML `yaml:"project"`
	Tags    []string     `yaml:"tags"`
	Note    string       `yaml:"note"`
	Exclude bool         `yaml:"exclude"`
	Done    bool         `yaml:"done"`
	Time    []string     `yaml:"time"`
	Untimed bool         `yaml:"untimed"`
}

// ProjectYAML はプロジェクトの YAML での形。
// 日次ファイルと items.yaml で同じ形を使う。
type ProjectYAML struct {
	Code string `yaml:"code"`
	Name string `yaml:"name"`
}

// Project は YAML から読んだプロジェクトを返す。
// p が nil なら、プロジェクトがない。
func (p *ProjectYAML) Project() (Project, error) {
	if p == nil {
		return Project{}, nil
	}
	if strings.TrimSpace(p.Code) == "" {
		return Project{}, fmt.Errorf("project に code がありません")
	}
	return Project{Code: p.Code, Name: p.Name}, nil
}

// ProjectNode はプロジェクトを YAML のノードにする。
// プロジェクトがなければ nil を返す。
func ProjectNode(p Project) *yaml.Node {
	if p.Code == "" {
		return nil
	}
	n := yamlfile.Mapping()
	yamlfile.Add(n, "code", yamlfile.Text(p.Code))
	if p.Name != "" {
		yamlfile.Add(n, "name", yamlfile.Text(p.Name))
	}
	return n
}

// ParseTimes は保存された形の時間の並びを読む。
func ParseTimes(ss []string) ([]timeexpr.Entry, error) {
	var es []timeexpr.Entry
	for _, s := range ss {
		e, err := timeexpr.ParseStored(s)
		if err != nil {
			return nil, err
		}
		es = append(es, e)
	}
	return es, nil
}

// TimesNode は時間の並びを、保存する形の YAML のノードにする。
func TimesNode(es []timeexpr.Entry) *yaml.Node {
	ss := make([]string, len(es))
	for i, e := range es {
		ss[i] = e.String()
	}
	return yamlfile.FlowList(ss)
}

// Parse は日次ファイルの中身を読む。
func Parse(data []byte) (Day, error) {
	var raw rawDay
	if err := yamlfile.Decode(data, &raw); err != nil {
		return Day{}, fmt.Errorf("日次ファイルを読めません: %w", err)
	}

	date, err := time.Parse(yamlfile.DateLayout, raw.Date)
	if err != nil {
		return Day{}, fmt.Errorf("日付 %q を読めません", raw.Date)
	}
	day := Day{Date: date, Skipped: raw.Skipped}

	for i, rt := range raw.Tasks {
		if strings.TrimSpace(rt.Title) == "" {
			return Day{}, fmt.Errorf("%d 件目のタスクに title がありません", i+1)
		}
		// 記録はタスク名で指すので、重なるとどちらの記録か決まらない。
		if slices.ContainsFunc(day.Tasks, func(t Task) bool { return t.Title == rt.Title }) {
			return Day{}, fmt.Errorf("タスク %q が 2 つあります", rt.Title)
		}
		t := Task{
			Title:   rt.Title,
			Tags:    rt.Tags,
			Note:    rt.Note,
			Exclude: rt.Exclude,
			Done:    rt.Done,
			Untimed: rt.Untimed,
		}
		if t.Project, err = rt.Project.Project(); err != nil {
			return Day{}, fmt.Errorf("タスク %q: %w", rt.Title, err)
		}
		if t.Time, err = ParseTimes(rt.Time); err != nil {
			return Day{}, fmt.Errorf("タスク %q: %w", rt.Title, err)
		}
		// 時間があるのか時間不定なのか決まらないので、両方は持たない。
		if t.Untimed && len(t.Time) > 0 {
			return Day{}, fmt.Errorf("タスク %q: time と untimed は一緒に書けません", rt.Title)
		}
		day.Tasks = append(day.Tasks, t)
	}

	notes, err := parseNotes(&raw.Notes)
	if err != nil {
		return Day{}, err
	}
	day.Notes = notes
	return day, nil
}

func parseNotes(n *yaml.Node) ([]Note, error) {
	if n.Kind == 0 {
		return nil, nil
	}
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("notes は欄の名前と値の組で書きます")
	}
	var notes []Note
	seen := map[string]bool{}
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if k.Kind != yaml.ScalarNode || v.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("notes の %d 番目の欄を読めません", i/2+1)
		}
		// ノードとして読むと YAML のライブラリはキーの重なりを確かめないので、ここで確かめる。
		// メモ欄は名前で値を引くので、重なっているとどちらの値を使うか決められない。
		if seen[k.Value] {
			return nil, fmt.Errorf("notes の欄 %q が重なっています", k.Value)
		}
		seen[k.Value] = true
		// 引用符で囲んだ "true" は文字列として読む。
		if v.ShortTag() == "!!bool" {
			// True や TRUE と書かれていても読めるように、YAML の解釈に任せる。
			var b bool
			if err := v.Decode(&b); err != nil {
				return nil, fmt.Errorf("notes の %q の値を読めません", k.Value)
			}
			notes = appendNote(notes, Note{Name: k.Value, Kind: NoteBool, Bool: b})
			continue
		}
		notes = appendNote(notes, Note{Name: k.Value, Kind: NoteText, Text: v.Value})
	}
	return notes, nil
}

// appendNote は、空でない欄だけを足す。手で書いた空の値は、書いていないものとして読む。
func appendNote(notes []Note, n Note) []Note {
	if n.Empty() {
		return notes
	}
	return append(notes, n)
}

// Marshal は日次ファイルの中身を書く。
// 空の項目は書かない。
func Marshal(day Day) ([]byte, error) {
	root := yamlfile.Mapping()
	yamlfile.Add(root, "date", yamlfile.Date(day.Date))

	if len(day.Tasks) > 0 {
		tasks := yamlfile.Sequence()
		for _, t := range day.Tasks {
			// 時間と時間不定の両方を持つ記録は、読むときにエラーになる。書いてから開けなくならないよう、書く前に止める。
			if t.Untimed && len(t.Time) > 0 {
				return nil, fmt.Errorf("タスク %q は時間と時間不定の両方を持っています", t.Title)
			}
			tasks.Content = append(tasks.Content, taskNode(t))
		}
		yamlfile.Add(root, "tasks", tasks)
	}
	if len(day.Skipped) > 0 {
		yamlfile.Add(root, "skipped", yamlfile.FlowList(day.Skipped))
	}
	notes := yamlfile.Mapping()
	for _, n := range day.Notes {
		if n.Empty() {
			continue
		}
		if n.Kind == NoteBool {
			yamlfile.Add(notes, n.Name, yamlfile.Bool(n.Bool))
			continue
		}
		yamlfile.Add(notes, n.Name, yamlfile.Text(n.Text))
	}
	if len(notes.Content) > 0 {
		yamlfile.Add(root, "notes", notes)
	}

	data, err := yamlfile.Encode(root)
	if err != nil {
		return nil, fmt.Errorf("日次ファイルを書けません: %w", err)
	}
	return data, nil
}

func taskNode(t Task) *yaml.Node {
	n := yamlfile.Mapping()
	yamlfile.Add(n, "title", yamlfile.Text(t.Title))
	if p := ProjectNode(t.Project); p != nil {
		yamlfile.Add(n, "project", p)
	}
	if len(t.Tags) > 0 {
		yamlfile.Add(n, "tags", yamlfile.FlowList(t.Tags))
	}
	if t.Note != "" {
		yamlfile.Add(n, "note", yamlfile.Text(t.Note))
	}
	if t.Exclude {
		yamlfile.Add(n, "exclude", yamlfile.Bool(true))
	}
	if t.Done {
		yamlfile.Add(n, "done", yamlfile.Bool(true))
	}
	if len(t.Time) > 0 {
		yamlfile.Add(n, "time", TimesNode(t.Time))
	}
	if t.Untimed {
		yamlfile.Add(n, "untimed", yamlfile.Bool(true))
	}
	return n
}
