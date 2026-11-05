package output

import (
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

const (
	textSrc = "---\nfile: d\nextension: txt\n---\nx"
	csvSrc  = "kind: csv\nfile: d\nrows: tasks\ncolumns:\n  - header: x\n    value: '{{.title}}'\n"
)

func names(outputs []Output) []string {
	var ns []string
	for _, o := range outputs {
		ns = append(ns, o.Name)
	}
	return ns
}

func TestLoad(t *testing.T) {
	lib, skipped := Load([]File{
		{"日報テキスト.tmpl", []byte(textSrc)},
		{"工数CSV.csv.yaml", []byte(csvSrc)},
		{"日報.set.yaml", []byte("kind: set\noutputs: [工数CSV, 日報テキスト]\n")},
		// 名前の形に合わないファイルは無視する。
		{"メモ.txt", []byte("x")},
		{"設定.yaml", []byte("x")},
		{".tmpl", []byte(textSrc)},
	})
	if len(skipped) > 0 {
		t.Fatalf("skipped = %v", skipped)
	}
	if got := strings.Join(names(lib.Outputs), ","); got != "日報テキスト,工数CSV" {
		t.Errorf("Outputs = %s", got)
	}
	outputs, err := lib.SetOutputs("日報")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(names(outputs), ","); got != "工数CSV,日報テキスト" {
		t.Errorf("SetOutputs = %s", got)
	}
}

// 読めないファイルは除き、ほかのファイルは使える。
func TestLoadSkipped(t *testing.T) {
	lib, skipped := Load([]File{
		{"壊れた.tmpl", []byte("本文だけ")},
		{"日報テキスト.tmpl", []byte(textSrc)},
		{"日報.set.yaml", []byte("kind: set\noutputs: [日報テキスト, 壊れた]\n")},
		{"ほか.set.yaml", []byte("kind: set\noutputs: [日報テキスト]\n")},
	})
	if len(skipped) != 1 || !strings.Contains(skipped[0].Error(), "壊れた.tmpl") {
		t.Errorf("skipped = %v", skipped)
	}
	if got := strings.Join(names(lib.Outputs), ","); got != "日報テキスト" {
		t.Errorf("Outputs = %s", got)
	}
	// 読めない出力を指す出力セットは、使うときにエラーにする。
	if _, err := lib.SetOutputs("日報"); err == nil {
		t.Error("SetOutputs(日報) succeeded, want error")
	}
	if _, err := lib.SetOutputs("ほか"); err != nil {
		t.Errorf("SetOutputs(ほか): %v", err)
	}
	if _, err := lib.SetOutputs("ない"); err == nil {
		t.Error("SetOutputs(ない) succeeded, want error")
	}
}

// 名前が重なったら、種類に関係なく、その名前のファイルはどれも使えない。
func TestLoadDuplicateNames(t *testing.T) {
	lib, skipped := Load([]File{
		{"日報.tmpl", []byte(textSrc)},
		{"日報.csv.yaml", []byte(csvSrc)},
		{"日報.set.yaml", []byte("kind: set\noutputs: [日報]\n")},
		{"工数.tmpl", []byte(textSrc)},
	})
	if len(skipped) != 3 {
		t.Errorf("skipped = %v", skipped)
	}
	if got := strings.Join(names(lib.Outputs), ","); got != "工数" {
		t.Errorf("Outputs = %s", got)
	}
	if len(lib.Sets) != 0 {
		t.Errorf("Sets = %v", lib.Sets)
	}
}

// macOS で作ったファイル名 (NFD) でも、NFC で書いた名前と同じものとして扱う。
func TestLoadNormalizesFileNames(t *testing.T) {
	nfd := norm.NFD.String("データ")
	lib, skipped := Load([]File{
		{nfd + ".tmpl", []byte(textSrc)},
		{"日報.set.yaml", []byte("kind: set\noutputs: [データ]\n")},
	})
	if len(skipped) > 0 {
		t.Fatalf("skipped = %v", skipped)
	}
	if _, err := lib.SetOutputs("日報"); err != nil {
		t.Errorf("SetOutputs: %v", err)
	}
	if _, ok := lib.Output(nfd); !ok {
		t.Error("Output(NFD) が見つからない")
	}

	// 表現が違うだけの名前は、重なりとして扱う。
	_, skipped = Load([]File{
		{nfd + ".tmpl", []byte(textSrc)},
		{"データ.csv.yaml", []byte(csvSrc)},
	})
	if len(skipped) != 2 {
		t.Errorf("skipped = %v", skipped)
	}
}
