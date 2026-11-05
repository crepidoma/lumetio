package gui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/store"
)

// 画面の形で読んだ出力を、そのまま保存しても中身が変わらない。
func TestSaveTemplateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	csv := "kind: csv\nfile: \"'effort_'yyyyMMdd\"\nencoding: utf-8-bom\nnewline: lf\nheader: false\nrows: tasks\nskip_rows_without_counted: true\ncolumns:\n  - header: 名前\n    value: '{{.title}}'\n"
	text := "---\nfile: yyyyMMdd\nextension: txt\nencoding: utf-8-bom\nnewline: crlf\n---\n{{.date}}\n"
	write(t, dir, "templates/工数.csv.yaml", csv)
	write(t, dir, "templates/日報.tmpl", text)
	s := New(app.New(store.New(dir), t.TempDir(), nil), nil, Env{})

	list, err := s.Templates()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Entries) != 2 {
		t.Fatalf("Entries = %+v", list.Entries)
	}
	want := map[string]string{"工数.csv.yaml": csv, "日報.tmpl": text}
	for _, e := range list.Entries {
		d := TemplateDraft{Name: e.Name, Kind: e.Kind, Text: e.Text, CSV: e.CSV, Set: e.Set}
		file, err := s.SaveTemplate(e.FileName, e.Data, d)
		if err != nil {
			t.Fatalf("%s: %v", e.FileName, err)
		}
		got, _ := os.ReadFile(filepath.Join(dir, "templates", file))
		if !reflect.DeepEqual(string(got), want[e.FileName]) {
			t.Errorf("%s =\n%s\nwant\n%s", file, got, want[e.FileName])
		}
	}
}

// 一覧から選んだ列は、テンプレートにして書き、読み直すと同じ値として出る。
func TestSaveTemplateColumnField(t *testing.T) {
	dir := t.TempDir()
	s := New(app.New(store.New(dir), t.TempDir(), nil), nil, Env{})
	d := TemplateDraft{Name: "工数", Kind: "csv", CSV: &CSVSpec{
		File: "yyyyMMdd", Encoding: "utf-8", Newline: "crlf", Header: true, Rows: "projects",
		Columns: []CSVColumn{{Header: "コード", Field: "code"}, {Header: "時間", Field: "time", Format: "H'h'"}, {Header: "固定", Value: "x"}},
	}}
	if _, err := s.SaveTemplate("", "", d); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "templates", "工数.csv.yaml"))
	if want := `value: '{{duration .minutes "H''h''"}}'`; !strings.Contains(string(data), want) {
		t.Errorf("書いた中身:\n%s", data)
	}
	list, err := s.Templates()
	if err != nil {
		t.Fatal(err)
	}
	got := list.Entries[0].CSV.Columns
	if got[0].Field != "code" || got[1].Field != "time" || got[1].Format != "H'h'" || got[2].Field != "" || got[2].Value != "x" {
		t.Errorf("Columns = %+v", got)
	}

	// 行の単位で使えない値は、保存しない。
	d.CSV.Columns = []CSVColumn{{Header: "タスク", Field: "title"}}
	if _, err := s.SaveTemplate("", "", TemplateDraft{Name: "別", Kind: "csv", CSV: d.CSV}); !app.IsInputError(err) {
		t.Errorf("使えない値: err = %v", err)
	}
}
