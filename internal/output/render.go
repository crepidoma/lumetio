package output

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"text/template"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/effort"
	"github.com/crepidoma/lumetio/internal/memo"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// clock は分を、テンプレートの time に使う H:mm で書く。
// 画面の表記 (duration_format) を変えても、出力が変わらないようにするため固定する。
func clock(minutes int) string {
	return fmt.Sprintf("%d:%02d", minutes/60, minutes%60)
}

// Result は作った出力。
type Result struct {
	FileName string
	// Text は改行を LF にした中身。画面のプレビューとコピーに使う。
	Text     string
	encoding Encoding
	newline  Newline
}

// Bytes はファイルに書く中身を、出力の文字コードと改行で返す。
func (r Result) Bytes() []byte {
	s := r.Text
	if r.newline == CRLF {
		s = strings.ReplaceAll(s, "\n", "\r\n")
	}
	if r.encoding == UTF8BOM {
		s = bom + s
	}
	return []byte(s)
}

// Render はその日の記録から出力を作る。
// noteDefs は settings.yaml のメモ欄の定義。
// 予定は day に含まれないので、出力にも入らない。
func (o Output) Render(day dayfile.Day, noteDefs []settings.NoteDef) (Result, error) {
	data := templateData(day, noteDefs)
	fn := funcs(dayFuncs{date: day.Date})

	var text string
	var err error
	if o.csv != nil {
		text, err = o.renderCSV(data, fn)
	} else {
		text, err = execute(o.text, fn, data)
	}
	if err != nil {
		return Result{}, err
	}
	return Result{
		FileName: o.File.Format(day.Date) + "." + o.Extension,
		Text:     text,
		encoding: o.Encoding,
		newline:  o.Newline,
	}, nil
}

func (o Output) renderCSV(data map[string]any, fn template.FuncMap) (string, error) {
	var b strings.Builder
	w := csv.NewWriter(&b)
	if o.csv.header {
		headers := make([]string, len(o.csv.columns))
		for i, c := range o.csv.columns {
			headers[i] = c.header
		}
		if err := w.Write(headers); err != nil {
			return "", err
		}
	}
	for _, row := range data[string(o.csv.rows)].([]map[string]any) {
		if o.csv.skipRowsWithoutCounted && !row["counted"].(bool) {
			continue
		}
		record := make([]string, len(o.csv.columns))
		for i, c := range o.csv.columns {
			v, err := execute(c.value, fn, row)
			if err != nil {
				return "", err
			}
			record[i] = v
		}
		if err := w.Write(record); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return b.String(), nil
}

// execute は、関数をその日のものに差し替えてテンプレートを動かす。
// 読み込んだテンプレートは複数の出力で共有されるので、写しに差し替える。
func execute(t *template.Template, fn template.FuncMap, data any) (string, error) {
	c, err := t.Clone()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := c.Funcs(fn).Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// templateData はテンプレートに渡す値を作る。
// 名前は YAML のキーと同じく snake_case にするため、構造体ではなく map で渡す。
func templateData(day dayfile.Day, noteDefs []settings.NoteDef) map[string]any {
	s := effort.Summarize(day)

	tasks := []map[string]any{}
	for _, t := range s.Tasks {
		tasks = append(tasks, taskData(t))
	}
	projects := []map[string]any{}
	for _, p := range s.Projects {
		pt := []map[string]any{}
		for _, t := range p.Tasks {
			pt = append(pt, taskData(t))
		}
		projects = append(projects, map[string]any{
			"code":        p.Code,
			"name":        p.Name,
			"minutes":     p.Minutes,
			"time":        clock(p.Minutes),
			"counted":     p.Counted,
			"has_untimed": p.HasUntimed,
			"tasks":       pt,
		})
	}
	notes := map[string]any{}
	for _, n := range memo.ForDay(noteDefs, day) {
		if n.Kind == dayfile.NoteBool {
			notes[n.Name] = n.Bool
		} else {
			notes[n.Name] = n.Text
		}
	}
	return map[string]any{
		"date":          day.Date.Format(yamlfile.DateLayout),
		"tasks":         tasks,
		"projects":      projects,
		"total_minutes": s.TotalMinutes,
		"total_time":    clock(s.TotalMinutes),
		"notes":         notes,
	}
}

func taskData(t effort.Task) map[string]any {
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{
		"title": t.Title,
		"project": map[string]any{
			"code": t.Project.Code,
			"name": t.Project.DisplayName(),
		},
		"tags":    tags,
		"note":    t.Note,
		"exclude": t.Exclude,
		"minutes": t.Minutes,
		"time":    clock(t.Minutes),
		"untimed": t.Untimed,
		"counted": t.Counted,
	}
}
