package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/settings"
)

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func TestSaveProjects(t *testing.T) {
	f := newFixture(t)
	base, err := f.app.AllProjects()
	if err != nil {
		t.Fatal(err)
	}
	// 期限を過ぎたものも一覧に入る。
	if len(base) != 2 || base[1].Code != "OLD-01" {
		t.Fatalf("AllProjects = %+v", base)
	}

	// 前後の空白を除いて書き、ほかの項目はそのまま残す。
	next := []settings.Project{{Code: " NEW-01 ", Name: " 新しい案件 ", Until: date(2027, 3, 31)}, base[0]}
	if err := f.app.SaveProjects(base, next); err != nil {
		t.Fatal(err)
	}
	st, _, err := f.store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	want := []settings.Project{{Code: "NEW-01", Name: "新しい案件", Until: date(2027, 3, 31)}, base[0]}
	if !reflect.DeepEqual(st.Projects, want) || len(st.Notes) != 2 || st.DefaultOutput != "日報" {
		t.Errorf("settings = %+v", st)
	}

	// 開いた後に他で一覧が変わっていたら書かない。
	if err := f.app.SaveProjects(base, base); !errors.Is(err, ErrConflict) {
		t.Errorf("他で変わった: err = %v, want ErrConflict", err)
	}

	for _, bad := range [][]settings.Project{
		{{Code: " "}},
		{{Code: "A"}, {Code: "A "}},
	} {
		if err := f.app.SaveProjects(want, bad); !IsInputError(err) {
			t.Errorf("%+v: err = %v, want input error", bad, err)
		}
	}
}

func TestReadCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.csv")
	// BOM を除き、行ごとに列の数が違っても読む。
	if err := os.WriteFile(path, []byte("\xef\xbb\xbfコード,名前\nA-1,案件A,2026-12-31\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"コード", "名前"}, {"A-1", "案件A", "2026-12-31"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %q", rows)
	}

	// Shift_JIS などは読まない。
	if err := os.WriteFile(path, []byte{0x83, 0x52, 0x81, 0x5b, 0x83, 0x68}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCSV(path); !IsInputError(err) {
		t.Errorf("UTF-8 でない: err = %v, want input error", err)
	}
}

func TestImportProjects(t *testing.T) {
	rows := [][]string{
		{"名前", "コード", "期限"},
		{"案件A", "A-1", "12/31/2026"},
		{"案件B", " B-1 ", ""},
		{"", "", ""},
		{"案件A2", "A-1", "01/31/2027"},
	}
	opt := ProjectImport{SkipHeader: true, CodeColumn: 1, NameColumn: 0, StartColumn: -1, UntilColumn: 2, DateFormat: "MM/dd/yyyy"}
	current := []settings.Project{{Code: "B-1", Name: "前の名前", Until: date(2026, 1, 1)}, {Code: "C-1"}}

	// 足すときは、同じコードを CSV の値で置き換える。CSV の中で重なるコードは後の行を使う。
	got, err := ImportProjects(rows, opt, current)
	if err != nil {
		t.Fatal(err)
	}
	want := []settings.Project{
		{Code: "B-1", Name: "案件B"},
		{Code: "C-1"},
		{Code: "A-1", Name: "案件A2", Until: date(2027, 1, 31)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("足す: %+v", got)
	}

	opt.Replace = true
	got, err = ImportProjects(rows, opt, current)
	if err != nil {
		t.Fatal(err)
	}
	want = []settings.Project{
		{Code: "A-1", Name: "案件A2", Until: date(2027, 1, 31)},
		{Code: "B-1", Name: "案件B"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("置き換える: %+v", got)
	}

	// 名前と期限の列は使わなくてもよい。見出しを飛ばさなければ 1 行目も取り込む。
	got, err = ImportProjects([][]string{{"X"}, {"Y"}}, ProjectImport{CodeColumn: 0, NameColumn: -1, StartColumn: -1, UntilColumn: -1}, nil)
	if err != nil || !reflect.DeepEqual(got, []settings.Project{{Code: "X"}, {Code: "Y"}}) {
		t.Errorf("コードだけ: %+v, %v", got, err)
	}
}

// 読めない行が 1 つでもあれば、何も取り込まない。
func TestImportProjectsErrors(t *testing.T) {
	ok := ProjectImport{CodeColumn: 0, NameColumn: -1, StartColumn: -1, UntilColumn: 1, DateFormat: "yyyy-MM-dd"}
	tests := []struct {
		name string
		rows [][]string
		opt  ProjectImport
		want string
	}{
		{"コードが空", [][]string{{"A", ""}, {" ", "2026-01-01"}}, ok, "2 行目のコードが空です"},
		{"書式と違う", [][]string{{"A", "2026/12/31"}}, ok, `1 行目の期限 "2026/12/31" を yyyy-MM-dd で読めません`},
		{"ない日", [][]string{{"A", "2026-02-30"}}, ok, `1 行目の期限 "2026-02-30" を yyyy-MM-dd で読めません`},
		{"取り込む行がない", [][]string{{"コード", "期限"}}, ProjectImport{SkipHeader: true, CodeColumn: 0, NameColumn: -1, StartColumn: -1, UntilColumn: -1}, "取り込む行がありません"},
		{"コードの列がない", [][]string{{"A"}}, ProjectImport{CodeColumn: -1, NameColumn: -1, StartColumn: -1, UntilColumn: -1}, "コードの列を選んでください"},
	}
	for _, tt := range tests {
		_, err := ImportProjects(tt.rows, tt.opt, nil)
		if !IsInputError(err) || err.Error() != tt.want {
			t.Errorf("%s: err = %v, want %q", tt.name, err, tt.want)
		}
	}
	for _, layout := range []string{"yyyy-MM-dd(EEE)", "yyyy-MM", "yyyy-MM-dd'"} {
		opt := ok
		opt.DateFormat = layout
		if _, err := ImportProjects([][]string{{"A", ""}}, opt, nil); !IsInputError(err) {
			t.Errorf("%q: err = %v, want input error", layout, err)
		}
	}
}

// 開始日も期限と同じ書式で読む。開始日が期限より後の行があれば、何も取り込まない。
func TestImportProjectsStart(t *testing.T) {
	opt := ProjectImport{CodeColumn: 0, NameColumn: -1, StartColumn: 1, UntilColumn: 2, DateFormat: "yyyy/M/d"}
	got, err := ImportProjects([][]string{{"A", "2026/4/1", "2027/3/31"}, {"B", "2026/10/1", ""}}, opt, nil)
	want := []settings.Project{
		{Code: "A", Start: date(2026, 4, 1), Until: date(2027, 3, 31)},
		{Code: "B", Start: date(2026, 10, 1)},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, %v", got, err)
	}

	for rows, msg := range map[string]string{
		"A,2026/4/2,2026/4/1": "1 行目の開始日が期限より後です",
		"A,2026/04/01,":       `1 行目の開始日 "2026/04/01" を yyyy/M/d で読めません`,
	} {
		_, err := ImportProjects([][]string{strings.Split(rows, ",")}, opt, nil)
		if !IsInputError(err) || err.Error() != msg {
			t.Errorf("%s: err = %v, want %q", rows, err, msg)
		}
	}
}

func TestSaveProjectsStartAfterUntil(t *testing.T) {
	f := newFixture(t)
	base, err := f.app.AllProjects()
	if err != nil {
		t.Fatal(err)
	}
	bad := []settings.Project{{Code: "A", Start: date(2026, 5, 1), Until: date(2026, 4, 30)}}
	if err := f.app.SaveProjects(base, bad); !IsInputError(err) {
		t.Errorf("err = %v, want input error", err)
	}
}

// 保存できない行は、行の番号でなく中身で指す。画面では期限切れの行を隠すので、番号が画面の並びと合わないため。
func TestSaveProjectsErrorNamesRow(t *testing.T) {
	f := newFixture(t)
	base, err := f.app.AllProjects()
	if err != nil {
		t.Fatal(err)
	}
	old := settings.Project{Code: "OLD", Until: date(2026, 1, 31)}
	cases := map[string][]settings.Project{
		`名前 "案件B" のコードが空です`:               {old, {Name: "案件B"}},
		"コードが空の行があります":                    {old, {}},
		`コード "B" の開始日が期限より後です`:            {old, {Code: "B", Start: date(2026, 5, 1), Until: date(2026, 4, 30)}},
		`コード "OLD" が、期限切れのプロジェクトと重なっています`: {old, {Code: "OLD"}},
	}
	for want, rows := range cases {
		if err := f.app.SaveProjects(base, rows); err == nil || err.Error() != want {
			t.Errorf("err = %v, want %s", err, want)
		}
	}
}

// NFC にそろえると同じになるコードは、重なりとして保存しない。
// そろえずに書くと、settings.yaml を読み直したときに重なりで読めなくなるため。
func TestSaveProjectsNFC(t *testing.T) {
	f := newFixture(t)
	base, err := f.app.AllProjects()
	if err != nil {
		t.Fatal(err)
	}
	// 「ガ」を、macOS で出ることのある「カ」と濁点の 2 文字で書く。
	decomposed, composed := "ガ-1", "ガ-1"
	if err := f.app.SaveProjects(base, []settings.Project{{Code: decomposed}, {Code: composed}}); !IsInputError(err) {
		t.Errorf("err = %v, want input error", err)
	}
	if err := f.app.SaveProjects(base, []settings.Project{{Code: decomposed}}); err != nil {
		t.Fatal(err)
	}
	if ps, _ := f.app.AllProjects(); len(ps) != 1 || ps[0].Code != composed {
		t.Errorf("Projects = %+v", ps)
	}
}

func TestReadCSVNFC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.csv")
	if err := os.WriteFile(path, []byte("ガ-1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadCSV(path)
	if err != nil || rows[0][0] != "ガ-1" {
		t.Errorf("rows = %q, %v", rows, err)
	}
}
