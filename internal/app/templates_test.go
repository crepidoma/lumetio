package app

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/crepidoma/lumetio/internal/output"
)

const (
	textTmpl = "---\nfile: \"'report_'yyyyMMdd\"\nextension: txt\n---\n{{range .tasks}}{{.title}}\n{{end}}"
	csvTmpl  = "kind: csv\nfile: \"'effort_'yyyyMMdd\"\nrows: projects\ncolumns:\n  - header: コード\n    value: '{{.code}}'\n"
	setTmpl  = "kind: set\noutputs: [日報テキスト, 工数]\n"
)

func (f *fixture) templates() Templates {
	f.t.Helper()
	ts, err := f.app.Templates()
	if err != nil {
		f.t.Fatal(err)
	}
	return ts
}

func (f *fixture) entry(fileName string) TemplateEntry {
	f.t.Helper()
	e, ok := findEntry(f.templates().Entries, fileName)
	if !ok {
		f.t.Fatalf("%s がありません", fileName)
	}
	return e
}

func (f *fixture) templateExists(fileName string) bool {
	_, err := os.Stat(filepath.Join(f.root, "templates", fileName))
	return err == nil
}

func newTemplateFixture(t *testing.T) *fixture {
	f := newFixture(t)
	f.write("templates/日報テキスト.tmpl", textTmpl)
	f.write("templates/工数.csv.yaml", csvTmpl)
	f.write("templates/日報.set.yaml", setTmpl)
	return f
}

// 読めないものも一覧に入れ、読めたものは画面で直す形にする。
func TestTemplates(t *testing.T) {
	f := newTemplateFixture(t)
	f.write("templates/壊れた.set.yaml", "kind: set\n")
	f.write("templates/メモ.txt", "名前の形に合わない")

	ts := f.templates()
	var names []string
	for _, e := range ts.Entries {
		names = append(names, e.FileName)
	}
	if !slices.Equal(names, []string{"壊れた.set.yaml", "工数.csv.yaml", "日報.set.yaml", "日報テキスト.tmpl"}) || ts.Default != "日報" {
		t.Fatalf("Templates = %v, %q", names, ts.Default)
	}
	if e := f.entry("壊れた.set.yaml"); e.Err == "" || e.Set != nil {
		t.Errorf("壊れた = %+v", e)
	}
	if e := f.entry("日報テキスト.tmpl"); e.Text == nil || e.Text.Extension != "txt" || e.Data != textTmpl {
		t.Errorf("日報テキスト = %+v", e)
	}
	if e := f.entry("日報.set.yaml"); !slices.Equal(e.Set, []string{"日報テキスト", "工数"}) {
		t.Errorf("日報 = %+v", e)
	}
}

// 出力の名前を変えると、ファイル名と、その出力を入れている出力セットも変わる。
func TestSaveTemplateRenameOutput(t *testing.T) {
	f := newTemplateFixture(t)
	e := f.entry("日報テキスト.tmpl")
	spec := *e.Text
	spec.Extension = "md"
	file, err := f.app.SaveTemplate(e.FileName, e.Data, TemplateDraft{Name: " 日報 本文 ", Kind: output.KindText, Text: &spec})
	if err != nil {
		t.Fatal(err)
	}
	if file != "日報 本文.tmpl" || f.templateExists("日報テキスト.tmpl") {
		t.Errorf("file = %q", file)
	}
	if got := f.entry("日報 本文.tmpl"); got.Text == nil || got.Text.Extension != "md" {
		t.Errorf("書いた後 = %+v", got)
	}
	if got := f.entry("日報.set.yaml").Set; !slices.Equal(got, []string{"日報 本文", "工数"}) {
		t.Errorf("出力セット = %v", got)
	}
}

// 出力セットの名前を変えると、既定の出力も追いかける。
func TestSaveTemplateRenameSet(t *testing.T) {
	f := newTemplateFixture(t)
	e := f.entry("日報.set.yaml")
	if _, err := f.app.SaveTemplate(e.FileName, e.Data, TemplateDraft{Name: "毎日", Kind: output.KindSet, Set: []string{"工数"}}); err != nil {
		t.Fatal(err)
	}
	if ts := f.templates(); ts.Default != "毎日" || !slices.Equal(f.entry("毎日.set.yaml").Set, []string{"工数"}) {
		t.Errorf("Templates = %+v", ts)
	}
}

func TestSaveTemplateNew(t *testing.T) {
	f := newTemplateFixture(t)
	spec := output.CSVSpec{File: "yyyyMMdd", Header: true, Rows: output.RowsTasks, Columns: []output.CSVColumn{{Header: "名前", Value: "{{.title}}"}}}
	file, err := f.app.SaveTemplate("", "", TemplateDraft{Name: "作業", Kind: output.KindCSV, CSV: &spec})
	if err != nil {
		t.Fatal(err)
	}
	if file != "作業.csv.yaml" || f.entry(file).CSV == nil {
		t.Errorf("file = %q", file)
	}
}

// 誤りのある中身、重なる名前、ない出力を入れた出力セットは保存しない。
func TestSaveTemplateErrors(t *testing.T) {
	f := newTemplateFixture(t)
	text := func(body string) *output.TextSpec {
		return &output.TextSpec{File: "yyyyMMdd", Extension: "txt", Body: body}
	}
	tests := map[string]TemplateDraft{
		"テンプレートの誤り": {Name: "新しい", Kind: output.KindText, Text: text("{{.title")},
		"拡張子がない":    {Name: "新しい", Kind: output.KindText, Text: &output.TextSpec{File: "yyyyMMdd"}},
		"名前が重なる":    {Name: "工数", Kind: output.KindText, Text: text("")},
		"名前に使えない文字": {Name: "a/b", Kind: output.KindText, Text: text("")},
		"名前が空":      {Name: " ", Kind: output.KindText, Text: text("")},
		"ない出力":      {Name: "新しい", Kind: output.KindSet, Set: []string{"ない"}},
		"空の出力セット":   {Name: "新しい", Kind: output.KindSet, Set: []string{}},
	}
	for name, d := range tests {
		if _, err := f.app.SaveTemplate("", "", d); !IsInputError(err) {
			t.Errorf("%s: err = %v, want input error", name, err)
		}
	}
	if f.templateExists("新しい.tmpl") || f.templateExists("新しい.set.yaml") {
		t.Error("誤りがあったのに書いた")
	}
}

// 開いた後に他でファイルが変わっていたら、保存も削除もしない。
func TestTemplateConflict(t *testing.T) {
	f := newTemplateFixture(t)
	e := f.entry("日報テキスト.tmpl")
	f.write("templates/日報テキスト.tmpl", textTmpl+"追記\n")
	if _, err := f.app.SaveTemplate(e.FileName, e.Data, TemplateDraft{Name: e.Name, Kind: e.Kind, Text: e.Text}); !errors.Is(err, ErrConflict) {
		t.Errorf("保存: err = %v", err)
	}
	if err := f.app.DeleteTemplate(e.FileName, e.Data); !errors.Is(err, ErrConflict) {
		t.Errorf("削除: err = %v", err)
	}
	if !f.templateExists("日報テキスト.tmpl") {
		t.Error("消えた")
	}
}

// 出力セットに入っている出力は消さない。既定の出力を消したら、既定はなくなる。
func TestDeleteTemplate(t *testing.T) {
	f := newTemplateFixture(t)
	e := f.entry("工数.csv.yaml")
	if err := f.app.DeleteTemplate(e.FileName, e.Data); !IsInputError(err) {
		t.Errorf("出力セットに入っている: err = %v", err)
	}
	set := f.entry("日報.set.yaml")
	if err := f.app.DeleteTemplate(set.FileName, set.Data); err != nil {
		t.Fatal(err)
	}
	if ts := f.templates(); ts.Default != "" || f.templateExists("日報.set.yaml") {
		t.Errorf("Templates = %+v", ts)
	}
	if err := f.app.DeleteTemplate(e.FileName, e.Data); err != nil || f.templateExists("工数.csv.yaml") {
		t.Errorf("出力セットを消した後: err = %v", err)
	}
}

func TestSaveDefaultOutput(t *testing.T) {
	f := newTemplateFixture(t)
	if err := f.app.SaveDefaultOutput("日報", ""); err != nil {
		t.Fatal(err)
	}
	if err := f.app.SaveDefaultOutput("日報", "日報"); !errors.Is(err, ErrConflict) {
		t.Errorf("他で変わった: err = %v", err)
	}
	if ts := f.templates(); ts.Default != "" {
		t.Errorf("Default = %q", ts.Default)
	}
}

// 保存する前の中身で、その日の結果を作る。誤りも結果として返す。
func TestPreviewTemplate(t *testing.T) {
	f := newTemplateFixture(t)
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 片付け\n")
	spec := output.TextSpec{File: "'report_'yyyyMMdd", Extension: "txt", Body: "{{range .tasks}}- {{.title}}\n{{end}}"}
	p, err := f.app.PreviewTemplate(today, TemplateDraft{Name: "下書き", Kind: output.KindText, Text: &spec})
	if err != nil || p.Err != "" || p.Text != "- 片付け\n" || p.FileName != "report_20261105.txt" {
		t.Errorf("Preview = %+v, %v", p, err)
	}
	spec.Body = "{{.ない}}"
	if p, err := f.app.PreviewTemplate(today, TemplateDraft{Name: "下書き", Kind: output.KindText, Text: &spec}); err != nil || p.Err == "" {
		t.Errorf("誤り: %+v, %v", p, err)
	}
}

// 既定の出力にした出力の名前を変えると既定も追いかけ、消すと既定はなくなる。
func TestDefaultOutputFollowsOutput(t *testing.T) {
	f := newTemplateFixture(t)
	if err := f.app.SaveDefaultOutput("日報", "日報テキスト"); err != nil {
		t.Fatal(err)
	}
	e := f.entry("日報テキスト.tmpl")
	if _, err := f.app.SaveTemplate(e.FileName, e.Data, TemplateDraft{Name: "本文", Kind: output.KindText, Text: e.Text}); err != nil {
		t.Fatal(err)
	}
	if got := f.templates().Default; got != "本文" {
		t.Errorf("名前を変えた後 = %q", got)
	}
	// 出力セットに入っているので、先に出力セットを消す。
	set := f.entry("日報.set.yaml")
	if err := f.app.DeleteTemplate(set.FileName, set.Data); err != nil {
		t.Fatal(err)
	}
	e = f.entry("本文.tmpl")
	if err := f.app.DeleteTemplate(e.FileName, e.Data); err != nil {
		t.Fatal(err)
	}
	if got := f.templates().Default; got != "" {
		t.Errorf("消した後 = %q", got)
	}
}

// 大文字と小文字だけが違う名前は重なりとして保存しない。Windows や macOS ではほかの出力を上書きしてしまうため。
func TestSaveTemplateCaseInsensitiveName(t *testing.T) {
	f := newTemplateFixture(t)
	f.write("templates/Report.tmpl", textTmpl)
	spec := output.TextSpec{File: "yyyyMMdd", Extension: "txt"}
	if _, err := f.app.SaveTemplate("", "", TemplateDraft{Name: "report", Kind: output.KindText, Text: &spec}); !IsInputError(err) {
		t.Errorf("err = %v, want input error", err)
	}
	if got := f.entry("Report.tmpl").Data; got != textTmpl {
		t.Errorf("Report.tmpl が変わった:\n%s", got)
	}
}

// 名前を変える途中で既定の出力を書けなかったら、新しいファイルを消して元に戻す。
// 出力セットの書き換えの途中で失敗したら、既定の出力と、書き換え終えた出力セットも元の名前に戻す。
func TestSaveTemplateRenameRollbackFollowers(t *testing.T) {
	f := newTemplateFixture(t)
	f.write("settings.yaml", "default_output: 日報テキスト\n")
	f.write("templates/週報.set.yaml", setTmpl)
	// 後から書き換える出力セットを読み取り専用にして、書き換えを失敗させる。
	locked := filepath.Join(f.root, "templates", "週報.set.yaml")
	if err := os.Chmod(locked, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o644) })

	e := f.entry("日報テキスト.tmpl")
	if _, err := f.app.SaveTemplate(e.FileName, e.Data, TemplateDraft{Name: "本文", Kind: output.KindText, Text: e.Text}); err == nil {
		t.Skip("この OS では読み取り専用のファイルも置き換えられるので、失敗させられない")
	}
	if !f.templateExists("日報テキスト.tmpl") || f.templateExists("本文.tmpl") {
		t.Errorf("出力が元に戻っていない")
	}
	if got, _ := os.ReadFile(filepath.Join(f.root, "templates", "日報.set.yaml")); string(got) != setTmpl {
		t.Errorf("日報.set.yaml = %q", got)
	}
	if got := f.templates().Default; got != "日報テキスト" {
		t.Errorf("既定の出力 = %q", got)
	}
}

// 古いファイルを消せなかったら、既定の出力と出力セットも元の名前に戻し、新しいファイルを消す。
func TestSaveTemplateRenameRollbackRemove(t *testing.T) {
	f := newTemplateFixture(t)
	f.write("settings.yaml", "default_output: 日報テキスト\n")
	// 古いファイルを開いたままにして、消すのを失敗させる (Windows では開いているファイルを消せない)。
	locked, err := os.Open(filepath.Join(f.root, "templates", "日報テキスト.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = locked.Close() })

	e := f.entry("日報テキスト.tmpl")
	if _, err := f.app.SaveTemplate(e.FileName, e.Data, TemplateDraft{Name: "本文", Kind: output.KindText, Text: e.Text}); err == nil {
		t.Skip("この OS では開いているファイルも消せるので、失敗させられない")
	}
	if !f.templateExists("日報テキスト.tmpl") || f.templateExists("本文.tmpl") {
		t.Errorf("出力が元に戻っていない")
	}
	if got, _ := os.ReadFile(filepath.Join(f.root, "templates", "日報.set.yaml")); string(got) != setTmpl {
		t.Errorf("日報.set.yaml = %q", got)
	}
	if got := f.templates().Default; got != "日報テキスト" {
		t.Errorf("既定の出力 = %q", got)
	}
}

// 大文字と小文字だけを変えて失敗したら、元の名前のファイルに戻す。出力の名前はファイル名から読むため。
func TestSaveTemplateCaseRenameRollback(t *testing.T) {
	f := newTemplateFixture(t)
	f.write("templates/Daily.tmpl", textTmpl)
	f.write("templates/週報.set.yaml", "kind: set\noutputs: [Daily]\n")
	locked := filepath.Join(f.root, "templates", "週報.set.yaml")
	if err := os.Chmod(locked, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o644) })

	e := f.entry("Daily.tmpl")
	if _, err := f.app.SaveTemplate(e.FileName, e.Data, TemplateDraft{Name: "daily", Kind: output.KindText, Text: e.Text}); err == nil {
		t.Skip("この OS では読み取り専用のファイルも置き換えられるので、失敗させられない")
	}
	var names []string
	for _, x := range f.templates().Entries {
		names = append(names, x.Name)
	}
	if !slices.Contains(names, "Daily") || slices.Contains(names, "daily") {
		t.Errorf("名前が戻っていない: %v", names)
	}
}

func TestSaveTemplateRenameRollback(t *testing.T) {
	f := newTemplateFixture(t)
	e := f.entry("日報テキスト.tmpl")
	// settings.yaml を読めなくして、既定の出力の書き換えを失敗させる。
	f.write("settings.yaml", "default_output: [\n")
	if _, err := f.app.SaveTemplate(e.FileName, e.Data, TemplateDraft{Name: "本文", Kind: output.KindText, Text: e.Text}); err == nil {
		t.Fatal("失敗しなかった")
	}
	if !f.templateExists("日報テキスト.tmpl") || f.templateExists("本文.tmpl") {
		t.Errorf("元に戻っていない: 日報テキスト=%v 本文=%v", f.templateExists("日報テキスト.tmpl"), f.templateExists("本文.tmpl"))
	}
}
