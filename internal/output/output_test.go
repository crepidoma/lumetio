package output

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/timeexpr"
)

// 2026-03-09 は月曜日。
var sampleDay = dayfile.Day{
	Date: time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
	Tasks: []dayfile.Task{
		{Title: "画面の実装", Project: dayfile.Project{Code: "DEV-001", Name: "新機能"}, Time: []timeexpr.Entry{{Minutes: 90}}},
		{Title: "週次ふりかえり", Project: dayfile.Project{Code: "MTG-01"}, Tags: []string{"ふりかえり"}, Note: "見積りが甘かった", Time: []timeexpr.Entry{{Minutes: 30}}},
		{Title: "調査", Project: dayfile.Project{Code: "DEV-001", Name: "新機能"}, Untimed: true},
		{Title: "雑談", Exclude: true, Time: []timeexpr.Entry{{Minutes: 15}}},
	},
	Notes: []dayfile.Note{
		{Name: "連絡", Kind: dayfile.NoteText, Text: ""},
		{Name: "在宅", Kind: dayfile.NoteBool, Bool: true},
	},
}

var sampleNotes = []settings.NoteDef{
	{Name: "連絡", Type: settings.NoteText},
	{Name: "在宅", Type: settings.NoteBool},
	{Name: "体調", Type: settings.NoteText},
}

func render(t *testing.T, o Output) Result {
	t.Helper()
	r, err := o.Render(sampleDay, sampleNotes)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return r
}

func TestRenderText(t *testing.T) {
	src := "---\n" +
		"file: \"'report_'yyyyMMdd\"\n" +
		"extension: txt\n" +
		"---\n" +
		"{{date \"yyyy/MM/dd(EEE)\"}} の日報\n" +
		"\n" +
		"■ やったこと\n" +
		"{{- range .projects}}\n" +
		"- {{.name}}{{if .counted}} ({{.time}}{{if .has_untimed}} +α{{end}}){{end}}\n" +
		"{{- range .tasks}}\n" +
		"  - {{.title}}{{if .counted}} ({{.time}}){{end}}\n" +
		"{{- end}}\n" +
		"{{- end}}\n" +
		"\n" +
		"■ 連絡\n" +
		"{{or .notes.連絡 \"なし\"}}\n" +
		"{{- with where .tasks \"tags\" \"ふりかえり\"}}\n" +
		"\n" +
		"■ ふりかえり\n" +
		"{{- range .}}\n" +
		"- {{.title}}: {{.note}}\n" +
		"{{- end}}\n" +
		"{{- end}}\n"
	o, err := ParseText("日報", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	r := render(t, o)
	want := "2026/03/09(月) の日報\n" +
		"\n" +
		"■ やったこと\n" +
		"- 新機能 (1:30 +α)\n" +
		"  - 画面の実装 (1:30)\n" +
		"  - 調査\n" +
		"- MTG-01 (0:30)\n" +
		"  - 週次ふりかえり (0:30)\n" +
		"- \n" +
		"  - 雑談\n" +
		"\n" +
		"■ 連絡\n" +
		"なし\n" +
		"\n" +
		"■ ふりかえり\n" +
		"- 週次ふりかえり: 見積りが甘かった\n"
	if r.Text != want {
		t.Errorf("Text =\n%q\nwant\n%q", r.Text, want)
	}
	if r.FileName != "report_20260309.txt" {
		t.Errorf("FileName = %q", r.FileName)
	}
}

func TestRenderValues(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"{{.date}}", "2026-03-09"},
		{"{{.total_minutes}} {{.total_time}}", "120 2:00"},
		{"{{duration .total_minutes \"H'h'\"}}", "2h"},
		{"{{.notes.在宅}} {{.notes.体調}}|", "true |"},
		{"{{index .notes \"連絡\"}}|", "|"},
		{"{{range .tasks}}{{.project.code}}/{{.project.name}},{{end}}", "DEV-001/新機能,MTG-01/MTG-01,DEV-001/新機能,/,"},
		{"{{range .tasks}}{{.untimed}}{{.counted}},{{end}}", "falsetrue,falsetrue,truefalse,falsefalse,"},
		{"{{len (where .tasks \"exclude\" true)}}", "1"},
		{"{{range where .tasks \"project.code\" \"DEV-001\"}}{{.title}},{{end}}", "画面の実装,調査,"},
		{"{{range where .tasks \"minutes\" 30}}{{.title}}{{end}}", "週次ふりかえり"},
		{"{{range .tasks}}{{if not (has .tags \"ふりかえり\")}}{{.title}},{{end}}{{end}}", "画面の実装,調査,雑談,"},
		{"{{range .tasks}}[{{join .tags \", \"}}]{{end}}", "[][ふりかえり][][]"},
		{"{{range .projects}}{{.code}}:{{.minutes}}:{{len .tasks}},{{end}}", "DEV-001:90:2,MTG-01:30:1,:0:1,"},
	}
	for _, tt := range tests {
		o, err := ParseText("t", []byte("---\nfile: d\nextension: txt\n---\n"+tt.body))
		if err != nil {
			t.Errorf("ParseText(%q): %v", tt.body, err)
			continue
		}
		r, err := o.Render(sampleDay, sampleNotes)
		if err != nil {
			t.Errorf("Render(%q): %v", tt.body, err)
			continue
		}
		if r.Text != tt.want {
			t.Errorf("Render(%q) = %q, want %q", tt.body, r.Text, tt.want)
		}
	}
}

// 書き間違えた値の名前や、関数への誤った値は、黙って空にせずエラーにする。
func TestRenderErrors(t *testing.T) {
	for _, body := range []string{
		"{{.titel}}",
		"{{range .tasks}}{{.titel}}{{end}}",
		"{{.notes.ないメモ}}",
		"{{where .tasks \"titel\" \"x\"}}",
		"{{where .tasks \"title.code\" \"x\"}}",
		"{{date \"HH\"}}",
		"{{duration 30 \"x\"}}",
	} {
		o, err := ParseText("t", []byte("---\nfile: d\nextension: txt\n---\n"+body))
		if err != nil {
			t.Errorf("ParseText(%q): %v", body, err)
			continue
		}
		if _, err := o.Render(sampleDay, sampleNotes); err == nil {
			t.Errorf("Render(%q) succeeded, want error", body)
		}
	}
}

func TestParseTextFront(t *testing.T) {
	o, err := ParseText("t", []byte("---\nfile: \"'a'\"\nextension: md\nencoding: utf-8-bom\nnewline: crlf\n---\nx"))
	if err != nil {
		t.Fatal(err)
	}
	if o.Extension != "md" || o.Encoding != UTF8BOM || o.Newline != CRLF || o.IsCSV() {
		t.Errorf("ParseText = %+v", o)
	}

	o, err = ParseText("t", []byte("---\nfile: d\nextension: txt\n---\nx"))
	if err != nil {
		t.Fatal(err)
	}
	if o.Encoding != UTF8 || o.Newline != LF {
		t.Errorf("省略時の値: %+v", o)
	}
}

// Windows で編集した CRLF のファイルや、BOM の付いたファイルも読める。
func TestParseTextCRLFAndBOM(t *testing.T) {
	src := "\xef\xbb\xbf---\r\nfile: d\r\nextension: txt\r\n---\r\na\r\nb\r\n"
	o, err := ParseText("t", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if r := render(t, o); r.Text != "a\nb\n" {
		t.Errorf("Text = %q", r.Text)
	}
}

func TestParseTextEmptyBody(t *testing.T) {
	for _, src := range []string{
		"---\nfile: d\nextension: txt\n---",
		"---\nfile: d\nextension: txt\n---\n",
	} {
		o, err := ParseText("t", []byte(src))
		if err != nil {
			t.Errorf("ParseText(%q): %v", src, err)
			continue
		}
		if r := render(t, o); r.Text != "" {
			t.Errorf("ParseText(%q): Text = %q", src, r.Text)
		}
	}
}

func TestParseTextErrors(t *testing.T) {
	for _, src := range []string{
		"本文だけ",
		"---\nfile: d\nextension: txt\n",
		"---\nextension: txt\n---\n",
		"---\nfile: d\n---\n",
		"---\nfile: HH\nextension: txt\n---\n",
		"---\nfile: \"yyyy/MM/dd\"\nextension: txt\n---\n",
		"---\nfile: d\nextension: .txt\n---\n",
		"---\nfile: d\nextension: txt\nencoding: shift_jis\n---\n",
		"---\nfile: d\nextension: txt\nnewline: cr\n---\n",
		"---\nfile: d\nextension: txt\nkind: text\n---\n",
		"---\nfile: d\nextension: txt\n---\n{{if}}",
		"---\nfile: d\nextension: txt\n---\n{{unknown}}",
	} {
		if _, err := ParseText("t", []byte(src)); err == nil {
			t.Errorf("ParseText(%q) succeeded, want error", src)
		}
	}
}

// テンプレートの誤りの行番号は、設定の行を含めたファイルの行で出す。
func TestParseTextErrorLine(t *testing.T) {
	_, err := ParseText("日報", []byte("---\nfile: d\nextension: txt\n---\n1 行目\n{{end}}\n"))
	if err == nil {
		t.Fatal("ParseText succeeded, want error")
	}
	if !strings.Contains(err.Error(), "日報:6:") {
		t.Errorf("err = %v, want line 6", err)
	}
}

func TestRenderCSV(t *testing.T) {
	src := `kind: csv
file: "'effort_'yyyyMMdd"
rows: projects
skip_rows_without_counted: true
columns:
  - header: date
    value: '{{date "yyyy/MM/dd"}}'
  - header: name
    value: '{{.name}}'
  - header: hours
    value: '{{if .has_untimed}}-{{else}}{{duration .minutes "H"}}{{end}}'
`
	o, err := ParseCSV("工数", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !o.IsCSV() || o.Newline != CRLF || o.Encoding != UTF8 {
		t.Errorf("省略時の値: %+v", o)
	}
	r := render(t, o)
	want := "date,name,hours\n" +
		"2026/03/09,新機能,-\n" +
		"2026/03/09,MTG-01,0.5\n"
	if r.Text != want {
		t.Errorf("Text =\n%q\nwant\n%q", r.Text, want)
	}
	if got := string(r.Bytes()); got != strings.ReplaceAll(want, "\n", "\r\n") {
		t.Errorf("Bytes = %q", got)
	}
	if r.FileName != "effort_20260309.csv" {
		t.Errorf("FileName = %q", r.FileName)
	}
}

// クォートとエスケープはアプリがする。
func TestRenderCSVQuote(t *testing.T) {
	src := `kind: csv
file: d
header: false
rows: tasks
columns:
  - header: x
    value: '{{.title}}'
  - header: y
    value: '{{.note}}'
`
	o, err := ParseCSV("t", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	day := dayfile.Day{
		Date:  sampleDay.Date,
		Tasks: []dayfile.Task{{Title: `a,"b"`, Note: "1 行目\n2 行目"}},
	}
	r, err := o.Render(day, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `"a,""b""","1 行目` + "\n" + `2 行目"` + "\n"
	if r.Text != want {
		t.Errorf("Text = %q, want %q", r.Text, want)
	}
}

func TestParseCSVErrors(t *testing.T) {
	base := "file: d\nrows: tasks\ncolumns:\n  - header: x\n    value: '{{.title}}'\n"
	for _, src := range []string{
		base,
		"kind: set\n" + base,
		"kind: csv\nfile: d\ncolumns:\n  - header: x\n    value: x\n",
		"kind: csv\nfile: d\nrows: days\ncolumns:\n  - header: x\n    value: x\n",
		"kind: csv\nfile: d\nrows: tasks\n",
		"kind: csv\nfile: d\nrows: tasks\ncolumns:\n  - header: x\n",
		"kind: csv\nfile: d\nrows: tasks\ncolumns:\n  - header: x\n    value: '{{if}}'\n",
		"kind: csv\nunknown: 1\n" + base,
		// 拡張子は csv に決まっているので書かない。
		"kind: csv\nextension: tsv\n" + base,
	} {
		if _, err := ParseCSV("t", []byte(src)); err == nil {
			t.Errorf("ParseCSV(%q) succeeded, want error", src)
		}
	}
}

func TestParseSet(t *testing.T) {
	s, err := ParseSet("日報", []byte("kind: set\noutputs: [日報テキスト, 工数CSV]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "日報" || !slices.Equal(s.Outputs, []string{"日報テキスト", "工数CSV"}) {
		t.Errorf("ParseSet = %+v", s)
	}

	for _, src := range []string{
		"outputs: [a]\n",
		"kind: csv\noutputs: [a]\n",
		"kind: set\n",
		"kind: set\noutputs: []\n",
		"kind: set\noutputs: [a, '']\n",
		"kind: set\noutputs: [a, a]\n",
	} {
		if _, err := ParseSet("t", []byte(src)); err == nil {
			t.Errorf("ParseSet(%q) succeeded, want error", src)
		}
	}
}

func TestResultBytes(t *testing.T) {
	r := Result{Text: "a\nb\n", encoding: UTF8BOM, newline: CRLF}
	if got := string(r.Bytes()); got != "\xef\xbb\xbfa\r\nb\r\n" {
		t.Errorf("Bytes = %q", got)
	}
	r = Result{Text: "a\nb\n", encoding: UTF8, newline: LF}
	if got := string(r.Bytes()); got != "a\nb\n" {
		t.Errorf("Bytes = %q", got)
	}
}
