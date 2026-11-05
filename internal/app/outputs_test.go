package app

import (
	"os"
	"slices"
	"testing"

	"github.com/crepidoma/lumetio/internal/output"
)

func TestChoices(t *testing.T) {
	f := newFixture(t)
	f.write("templates/日報テキスト.tmpl", "---\nfile: \"'report_'yyyyMMdd\"\nextension: txt\n---\n")
	f.write("templates/工数.csv.yaml", "kind: csv\nfile: \"'effort_'yyyyMMdd\"\nrows: projects\ncolumns:\n  - header: コード\n    value: \"{{.code}}\"\n")
	f.write("templates/日報.set.yaml", "kind: set\noutputs: [日報テキスト, 工数]\n")
	f.write("templates/壊れた.set.yaml", "kind: set\n")

	c, err := f.app.Choices()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Sets, []string{"日報"}) || !slices.Equal(c.Outputs, []string{"工数", "日報テキスト"}) || c.Default != "日報" || len(c.Skipped) != 1 {
		t.Errorf("Choices = %+v", c)
	}
}

// 作れない出力があっても、ほかの出力は作る。
func TestPreview(t *testing.T) {
	f := newFixture(t)
	f.write("templates/日報テキスト.tmpl", "---\nfile: \"'report_'yyyyMMdd\"\nextension: txt\n---\n{{range .tasks}}{{.title}}\n{{end}}")
	f.write("templates/誤り.tmpl", "---\nfile: \"'bad_'yyyyMMdd\"\nextension: txt\n---\n{{.ない}}")
	f.write("templates/日報.set.yaml", "kind: set\noutputs: [誤り, 日報テキスト]\n")
	f.write("days/2026/2026-11-05.yaml", "date: 2026-11-05\ntasks:\n  - title: 片付け\n    project:\n      code: OLD-01\n")

	r, err := f.app.Preview(today, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Outputs) != 2 || r.Outputs[0].Err == "" || r.Outputs[1].Err != "" || r.Outputs[1].Text != "片付け\n" || r.Outputs[1].FileName != "report_20261105.txt" {
		t.Errorf("Outputs = %+v", r.Outputs)
	}
	if _, err := f.app.Preview(today, "ない"); !IsInputError(err) {
		t.Errorf("ない名前: err = %v", err)
	}
}

// 既定の出力は、出力セットでなく出力 1 つでもよい。
func TestChoicesDefaultOutput(t *testing.T) {
	f := newFixture(t)
	f.write("settings.yaml", "default_output: 日報テキスト\n")
	f.write("templates/日報テキスト.tmpl", "---\nfile: \"'report_'yyyyMMdd\"\nextension: txt\n---\n")
	c, err := f.app.Choices()
	if err != nil {
		t.Fatal(err)
	}
	if c.Default != "日報テキスト" {
		t.Errorf("Default = %q", c.Default)
	}
}

// 出力セットの中で、大文字と小文字だけが違うファイル名になる出力があれば、何も書かずに入力の誤りにする。
// Windows や macOS では同じファイルになり、後の出力で先の出力を消してしまうため。
func TestExportDuplicateFileName(t *testing.T) {
	f := newFixture(t)
	f.write("templates/a.tmpl", "---\nfile: \"'Daily'\"\nextension: txt\n---\nA\n")
	f.write("templates/b.tmpl", "---\nfile: \"'daily'\"\nextension: txt\n---\nB\n")
	f.write("templates/両方.set.yaml", "kind: set\noutputs: [a, b]\n")
	if _, _, err := f.app.Export(today, "両方", output.ConflictAsk); !IsInputError(err) {
		t.Errorf("err = %v, want input error", err)
	}
	if entries, _ := os.ReadDir(f.out); len(entries) != 0 {
		t.Errorf("書いた: %v", entries)
	}
}
