package gui

import (
	"reflect"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/store"
)

// 画面の行で読んだ一覧を、そのまま書いて読み直せる。開始日と期限は yyyy-MM-dd でやり取りする。
func TestSaveProjectRows(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "settings.yaml", "projects:\n  - code: A-1\n    name: 案件A\n    start: 2026-04-01\n    until: 2026-12-31\n  - code: B-1\n")
	s := New(app.New(store.New(dir), t.TempDir(), fixedNow), nil, Env{})

	base, err := s.ProjectRows()
	if err != nil {
		t.Fatal(err)
	}
	want := []ProjectRow{{Code: "A-1", Name: "案件A", Start: "2026-04-01", Until: "2026-12-31"}, {Code: "B-1"}}
	if !reflect.DeepEqual(base, want) {
		t.Fatalf("ProjectRows = %+v", base)
	}
	next := append(base, ProjectRow{Code: "C-1", Name: "案件C", Until: "2027-01-31"})
	if err := s.SaveProjects(base, next); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ProjectRows(); !reflect.DeepEqual(got, next) {
		t.Errorf("書いた後 = %+v", got)
	}
	for _, bad := range []ProjectRow{{Code: "A-1", Until: "2026/12/31"}, {Code: "A-1", Start: "2026/04/01"}} {
		if err := s.SaveProjects(next, []ProjectRow{bad}); !app.IsInputError(err) {
			t.Errorf("%+v: err = %v, want input error", bad, err)
		}
	}
}

func TestImportProjectRows(t *testing.T) {
	s := New(app.New(store.New(t.TempDir()), t.TempDir(), fixedNow), nil, Env{})
	rows := [][]string{{"code", "name", "start", "until"}, {"B-1", "案件B", "2026/4/1", "2027/1/31"}}
	opt := ProjectImport{SkipHeader: true, CodeColumn: 0, NameColumn: 1, StartColumn: 2, UntilColumn: 3, DateFormat: "yyyy/M/d"}
	got, err := s.ImportProjects(rows, opt, []ProjectRow{{Code: "A-1"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []ProjectRow{{Code: "A-1"}, {Code: "B-1", Name: "案件B", Start: "2026-04-01", Until: "2027-01-31"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

// 取り込みで変わらなかった行は、期限切れかを編集中のままにする。CSV で期限が変わった行は、今日で決め直す。
func TestImportProjectRowsKeepsExpired(t *testing.T) {
	s := New(app.New(store.New(t.TempDir()), t.TempDir(), fixedNow), nil, Env{})
	rows := [][]string{{"B-1", "2026/1/31"}}
	opt := ProjectImport{CodeColumn: 0, NameColumn: -1, StartColumn: -1, UntilColumn: 1, DateFormat: "yyyy/M/d"}
	// A-1 は、直している途中で期限を過去にした行。
	current := []ProjectRow{{Code: "A-1", Until: "2026-01-31"}, {Code: "B-1"}}
	got, err := s.ImportProjects(rows, opt, current)
	if err != nil {
		t.Fatal(err)
	}
	want := []ProjectRow{{Code: "A-1", Until: "2026-01-31"}, {Code: "B-1", Until: "2026-01-31", Expired: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

// fixedNow は、テストでの今日。期限切れかどうかが、テストを動かした日で変わらないようにするため。
func fixedNow() time.Time {
	return time.Date(2026, 11, 5, 9, 0, 0, 0, time.Local)
}

// 期限が今日より前のプロジェクトに、期限切れの印を付ける。期限の日はまだ使える。
func TestProjectRowsExpired(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "settings.yaml", "projects:\n  - code: OLD\n    until: 2026-11-04\n  - code: LAST\n    until: 2026-11-05\n  - code: OPEN\n")
	s := New(app.New(store.New(dir), t.TempDir(), fixedNow), nil, Env{})
	rows, err := s.ProjectRows()
	if err != nil {
		t.Fatal(err)
	}
	var expired []string
	for _, r := range rows {
		if r.Expired {
			expired = append(expired, r.Code)
		}
	}
	if !reflect.DeepEqual(expired, []string{"OLD"}) {
		t.Errorf("expired = %v", expired)
	}
}
