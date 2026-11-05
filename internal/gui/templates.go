package gui

import (
	"fmt"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/output"
)

// TextSpec は、テキストの出力を画面で直すときの中身。
type TextSpec struct {
	File      string `json:"file"`
	Extension string `json:"extension"`
	// Encoding は utf-8 か utf-8-bom。Newline は lf か crlf。
	Encoding string `json:"encoding"`
	Newline  string `json:"newline"`
	Body     string `json:"body"`
}

// CSVColumn は CSV の列 1 つ。
type CSVColumn struct {
	Header string `json:"header"`
	// Field は一覧から選んだ値 (CSVFields の key)。空なら Value にテンプレートをそのまま書く。
	Field string `json:"field"`
	// Format は、日付と時間の書式。
	Format string `json:"format"`
	Value  string `json:"value"`
}

// CSVField は、CSV の列の値として一覧から選べるもの。
type CSVField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Format は、書式を選ぶものなら date か duration。選ばないなら空。
	Format        string `json:"format"`
	DefaultFormat string `json:"defaultFormat"`
}

// CSVFields は、行の単位 (projects か tasks) で、列の値として選べるものを返す。
func (s *Service) CSVFields(rows string) []CSVField {
	out := []CSVField{}
	for _, f := range output.ColumnFields(output.Rows(rows)) {
		out = append(out, CSVField{Key: f.Key, Label: f.Label, Format: string(f.Format), DefaultFormat: f.DefaultFormat})
	}
	return out
}

// CSVSpec は、CSV の出力を画面で直すときの中身。
type CSVSpec struct {
	File     string `json:"file"`
	Encoding string `json:"encoding"`
	Newline  string `json:"newline"`
	Header   bool   `json:"header"`
	// Rows は projects か tasks。
	Rows                   string      `json:"rows"`
	SkipRowsWithoutCounted bool        `json:"skipRowsWithoutCounted"`
	Columns                []CSVColumn `json:"columns"`
}

// TemplateEntry は、templates/ にある出力か出力セット 1 つ。
type TemplateEntry struct {
	FileName string `json:"fileName"`
	Name     string `json:"name"`
	// Kind は text、csv、set のどれか。
	Kind string `json:"kind"`
	// Data は読んだときの中身。保存と削除のときに、そのまま返す。
	Data string `json:"data"`
	// Error は読めない理由。読めれば空。
	Error string    `json:"error"`
	Text  *TextSpec `json:"text"`
	CSV   *CSVSpec  `json:"csv"`
	Set   []string  `json:"set"`
}

// TemplateList は、設定画面で直す出力と出力セット。
type TemplateList struct {
	Entries []TemplateEntry `json:"entries"`
	Default string          `json:"default"`
	// Notices は、ファイルを読めずに一覧に入れられなかった知らせ。
	Notices []string `json:"notices"`
}

// TemplateDraft は、画面で直した出力か出力セットの中身。種類に合うものだけを入れる。
type TemplateDraft struct {
	Name string    `json:"name"`
	Kind string    `json:"kind"`
	Text *TextSpec `json:"text"`
	CSV  *CSVSpec  `json:"csv"`
	Set  []string  `json:"set"`
}

// Templates は、出力と出力セットを、読めないものも含めて返す。
func (s *Service) Templates() (TemplateList, error) {
	if s.startErr != nil {
		return TemplateList{}, s.startErr
	}
	ts, err := s.app().Templates()
	if err != nil {
		return TemplateList{}, err
	}
	out := TemplateList{Entries: []TemplateEntry{}, Default: ts.Default, Notices: nonNil(ts.Skipped)}
	for _, e := range ts.Entries {
		te := TemplateEntry{FileName: e.FileName, Name: e.Name, Kind: string(e.Kind), Data: e.Data, Error: e.Err, Set: e.Set}
		if e.Text != nil {
			te.Text = &TextSpec{File: e.Text.File, Extension: e.Text.Extension, Encoding: string(e.Text.Encoding), Newline: string(e.Text.Newline), Body: e.Text.Body}
		}
		if e.CSV != nil {
			te.CSV = &CSVSpec{
				File: e.CSV.File, Encoding: string(e.CSV.Encoding), Newline: string(e.CSV.Newline),
				Header: e.CSV.Header, Rows: string(e.CSV.Rows), SkipRowsWithoutCounted: e.CSV.SkipRowsWithoutCounted, Columns: []CSVColumn{},
			}
			for _, c := range e.CSV.Columns {
				// 一覧の値と同じ形の列は、選んだ値として出す。テンプレートを書かなくて済むようにするため。
				field, format := output.DescribeColumn(e.CSV.Rows, c.Value)
				te.CSV.Columns = append(te.CSV.Columns, CSVColumn{Header: c.Header, Field: field, Format: format, Value: c.Value})
			}
		}
		out.Entries = append(out.Entries, te)
	}
	return out, nil
}

// toApp は、画面の中身を保存する形にする。一覧から選んだ列の値は、ここでテンプレートにする。
func (d TemplateDraft) toApp() (app.TemplateDraft, error) {
	out := app.TemplateDraft{Name: d.Name, Kind: output.Kind(d.Kind), Set: d.Set}
	if d.Text != nil {
		out.Text = &output.TextSpec{File: d.Text.File, Extension: d.Text.Extension, Encoding: output.Encoding(d.Text.Encoding), Newline: output.Newline(d.Text.Newline), Body: d.Text.Body}
	}
	if d.CSV != nil {
		rows := output.Rows(d.CSV.Rows)
		out.CSV = &output.CSVSpec{
			File: d.CSV.File, Encoding: output.Encoding(d.CSV.Encoding), Newline: output.Newline(d.CSV.Newline),
			Header: d.CSV.Header, Rows: rows, SkipRowsWithoutCounted: d.CSV.SkipRowsWithoutCounted,
		}
		for i, c := range d.CSV.Columns {
			value := c.Value
			if c.Field != "" {
				v, err := output.ColumnValue(rows, c.Field, c.Format)
				if err != nil {
					return app.TemplateDraft{}, &app.InputError{Message: fmt.Sprintf("%d 列目: %v", i+1, err)}
				}
				value = v
			}
			out.CSV.Columns = append(out.CSV.Columns, output.CSVColumn{Header: c.Header, Value: value})
		}
	}
	return out, nil
}

// SaveTemplate は、出力か出力セットを書き、書いたファイル名を返す。
// baseFile と baseData は直し始めたときのファイル名と中身。新しく作るときは baseFile を空にする。
func (s *Service) SaveTemplate(baseFile, baseData string, d TemplateDraft) (string, error) {
	if s.startErr != nil {
		return "", s.startErr
	}
	draft, err := d.toApp()
	if err != nil {
		return "", err
	}
	return s.app().SaveTemplate(baseFile, baseData, draft)
}

// DeleteTemplate は、出力か出力セットのファイルを消す。baseData は読んだときの中身。
func (s *Service) DeleteTemplate(fileName, baseData string) error {
	if s.startErr != nil {
		return s.startErr
	}
	return s.app().DeleteTemplate(fileName, baseData)
}

// SaveDefaultOutput は既定の出力を書く。base は読んだときの既定。
func (s *Service) SaveDefaultOutput(base, name string) error {
	if s.startErr != nil {
		return s.startErr
	}
	return s.app().SaveDefaultOutput(base, name)
}

// PreviewTemplate は、保存する前の出力の中身で、その日の結果を作る。誤りも結果に入れる。
func (s *Service) PreviewTemplate(date string, d TemplateDraft) (PreviewOutput, error) {
	day, err := s.date(date)
	if err != nil {
		return PreviewOutput{}, err
	}
	draft, err := d.toApp()
	// 直している途中の誤りは、結果の場所に出す。
	if app.IsInputError(err) {
		return PreviewOutput{Name: d.Name, Error: err.Error()}, nil
	}
	if err != nil {
		return PreviewOutput{}, err
	}
	p, err := s.app().PreviewTemplate(day, draft)
	if err != nil {
		return PreviewOutput{}, err
	}
	return PreviewOutput{Name: p.Name, FileName: p.FileName, Text: p.Text, Error: p.Err}, nil
}
