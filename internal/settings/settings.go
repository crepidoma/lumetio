// Package settings は settings.yaml の中身を読み書きする。
package settings

import (
	"fmt"
	"strings"
	"time"

	"github.com/crepidoma/lumetio/internal/durfmt"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// DefaultDurationFormat は、長さの表記を決めていないときの書式。
const DefaultDurationFormat = "H:mm"

// Settings は複数の PC で共有する設定。
type Settings struct {
	Projects []Project
	// Notes の並びが、メモ欄を画面に出す順。
	Notes []NoteDef
	// Tags の並びが、タグの候補の順。
	Tags           []string
	DurationFormat string
	// DefaultOutput が空なら、既定の出力はない。
	DefaultOutput string
}

// ParseDurationFormat は長さの表記を読む。前後の空白を除き、空なら省略時の表記にする。
// 保存するとき、入力中に見本を出すとき、画面に出すときで、同じ決まりで読むため。
func ParseDurationFormat(layout string) (string, durfmt.Format, error) {
	layout = strings.TrimSpace(layout)
	if layout == "" {
		layout = DefaultDurationFormat
	}
	f, err := durfmt.Parse(layout)
	return layout, f, err
}

// Default は settings.yaml がないときの設定。
func Default() Settings {
	return Settings{DurationFormat: DefaultDurationFormat}
}

// Project はプロジェクトの一覧の 1 件。
type Project struct {
	Code string
	Name string
	// Start はこの日から使えるという開始日。
	// ゼロ値なら開始日はない。
	Start time.Time
	// Until はこの日まで使えるという期限。
	// ゼロ値なら期限はない。
	Until time.Time
}

// InPeriod は、date がプロジェクトの期間 (Start から Until まで) の中か。
// date は時刻を持たない日付で渡す。
func (p Project) InPeriod(date time.Time) bool {
	return (p.Start.IsZero() || !date.Before(p.Start)) && (p.Until.IsZero() || !date.After(p.Until))
}

// NoteType はメモ欄の値の型。
type NoteType string

const (
	NoteText NoteType = "text"
	NoteBool NoteType = "bool"
)

// NoteDef はメモ欄 1 つ分の定義。
type NoteDef struct {
	Name string
	Type NoteType
	// CarryOver が true なら、前回の値を引き継ぐ。
	// false なら毎回空から始める。
	CarryOver bool
}

type rawSettings struct {
	Projects       []rawProject `yaml:"projects"`
	Notes          []rawNoteDef `yaml:"notes"`
	Tags           []string     `yaml:"tags"`
	DurationFormat string       `yaml:"duration_format"`
	DefaultOutput  string       `yaml:"default_output"`
}

type rawProject struct {
	Code  string `yaml:"code"`
	Name  string `yaml:"name"`
	Start string `yaml:"start"`
	Until string `yaml:"until"`
}

type rawNoteDef struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	CarryOver bool   `yaml:"carry_over"`
}

// Parse は settings.yaml の中身を読む。
// 書かれていない項目は省略時の値にする。
func Parse(data []byte) (Settings, error) {
	var raw rawSettings
	if err := yamlfile.Decode(data, &raw); err != nil {
		return Settings{}, fmt.Errorf("settings.yaml を読めません: %w", err)
	}

	s := Default()
	if raw.DurationFormat != "" {
		if _, err := durfmt.Parse(raw.DurationFormat); err != nil {
			return Settings{}, fmt.Errorf("duration_format: %w", err)
		}
		s.DurationFormat = raw.DurationFormat
	}
	s.DefaultOutput = raw.DefaultOutput

	codes := map[string]bool{}
	for i, rp := range raw.Projects {
		if strings.TrimSpace(rp.Code) == "" {
			return Settings{}, fmt.Errorf("%d 件目のプロジェクトに code がありません", i+1)
		}
		// 同じコードが 2 つあると、どちらの名前や期間を使うか決められない。
		if codes[rp.Code] {
			return Settings{}, fmt.Errorf("プロジェクトのコード %q が重なっています", rp.Code)
		}
		codes[rp.Code] = true

		p := Project{Code: rp.Code, Name: rp.Name}
		if rp.Start != "" {
			start, err := time.Parse(yamlfile.DateLayout, rp.Start)
			if err != nil {
				return Settings{}, fmt.Errorf("プロジェクト %q の開始日 %q を読めません", rp.Code, rp.Start)
			}
			p.Start = start
		}
		if rp.Until != "" {
			until, err := time.Parse(yamlfile.DateLayout, rp.Until)
			if err != nil {
				return Settings{}, fmt.Errorf("プロジェクト %q の期限 %q を読めません", rp.Code, rp.Until)
			}
			p.Until = until
		}
		// 開始日が期限より後だと、使える日が 1 日もない。
		if !p.Start.IsZero() && !p.Until.IsZero() && p.Start.After(p.Until) {
			return Settings{}, fmt.Errorf("プロジェクト %q の開始日 %s が期限 %s より後です", rp.Code, rp.Start, rp.Until)
		}
		s.Projects = append(s.Projects, p)
	}

	names := map[string]bool{}
	for i, rn := range raw.Notes {
		if strings.TrimSpace(rn.Name) == "" {
			return Settings{}, fmt.Errorf("%d 件目のメモ欄に name がありません", i+1)
		}
		// 日次ファイルはメモ欄を名前で持つので、名前が重なると値を取り違える。
		if names[rn.Name] {
			return Settings{}, fmt.Errorf("メモ欄の名前 %q が重なっています", rn.Name)
		}
		names[rn.Name] = true

		n := NoteDef{Name: rn.Name, Type: NoteText, CarryOver: rn.CarryOver}
		switch NoteType(rn.Type) {
		case "", NoteText:
		case NoteBool:
			n.Type = NoteBool
		default:
			return Settings{}, fmt.Errorf("メモ欄 %q の type %q は使えません。text か bool を書きます", rn.Name, rn.Type)
		}
		s.Notes = append(s.Notes, n)
	}

	tags := map[string]bool{}
	for i, t := range raw.Tags {
		if strings.TrimSpace(t) == "" {
			return Settings{}, fmt.Errorf("%d 件目のタグが空です", i+1)
		}
		// 同じタグが 2 つあると、候補に同じものが並ぶ。
		if tags[t] {
			return Settings{}, fmt.Errorf("タグ %q が重なっています", t)
		}
		tags[t] = true
		s.Tags = append(s.Tags, t)
	}
	return s, nil
}

// Marshal は settings.yaml の中身を書く。
// 省略時の値と同じ項目は書かない。
func Marshal(s Settings) ([]byte, error) {
	root := yamlfile.Mapping()

	if len(s.Projects) > 0 {
		projects := yamlfile.Sequence()
		for _, p := range s.Projects {
			n := yamlfile.Mapping()
			yamlfile.Add(n, "code", yamlfile.Text(p.Code))
			if p.Name != "" {
				yamlfile.Add(n, "name", yamlfile.Text(p.Name))
			}
			if !p.Start.IsZero() {
				yamlfile.Add(n, "start", yamlfile.Date(p.Start))
			}
			if !p.Until.IsZero() {
				yamlfile.Add(n, "until", yamlfile.Date(p.Until))
			}
			projects.Content = append(projects.Content, n)
		}
		yamlfile.Add(root, "projects", projects)
	}
	if len(s.Notes) > 0 {
		notes := yamlfile.Sequence()
		for _, d := range s.Notes {
			n := yamlfile.Mapping()
			yamlfile.Add(n, "name", yamlfile.Text(d.Name))
			if d.Type != "" && d.Type != NoteText {
				yamlfile.Add(n, "type", yamlfile.Text(string(d.Type)))
			}
			if d.CarryOver {
				yamlfile.Add(n, "carry_over", yamlfile.Bool(true))
			}
			notes.Content = append(notes.Content, n)
		}
		yamlfile.Add(root, "notes", notes)
	}
	if len(s.Tags) > 0 {
		yamlfile.Add(root, "tags", yamlfile.FlowList(s.Tags))
	}
	if s.DurationFormat != "" && s.DurationFormat != DefaultDurationFormat {
		yamlfile.Add(root, "duration_format", yamlfile.Text(s.DurationFormat))
	}
	if s.DefaultOutput != "" {
		yamlfile.Add(root, "default_output", yamlfile.Text(s.DefaultOutput))
	}

	// 何も設定していないときに "{}" と書かないよう、空のファイルにする。
	if len(root.Content) == 0 {
		return nil, nil
	}
	data, err := yamlfile.Encode(root)
	if err != nil {
		return nil, fmt.Errorf("settings.yaml を書けません: %w", err)
	}
	return data, nil
}
