package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/settings"
)

var day1029 = time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)

func TestDayPath(t *testing.T) {
	s := New("root")
	want := filepath.Join("root", "days", "2026", "2026-10-29.yaml")
	// 時刻やタイムゾーンが付いていても、日付だけで決まる。
	date := time.Date(2026, 10, 29, 23, 30, 0, 0, time.FixedZone("JST", 9*60*60))
	if got := s.DayPath(date); got != want {
		t.Errorf("DayPath = %s, want %s", got, want)
	}
}

func TestLoadMissingDayCreatesNothing(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	day, _, err := s.LoadDay(day1029)
	if err != nil {
		t.Fatal(err)
	}
	if !day.Date.Equal(day1029) || len(day.Tasks) != 0 {
		t.Errorf("day = %+v", day)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("開いただけでファイルができた: %v", entries)
	}
}

func TestSaveAndLoad(t *testing.T) {
	s := New(t.TempDir())
	day, v, err := s.LoadDay(day1029)
	if err != nil {
		t.Fatal(err)
	}
	day.Tasks = append(day.Tasks, dayfile.Task{Title: "資料の作成"})
	v, err = s.SaveDay(day, v)
	if err != nil {
		t.Fatal(err)
	}

	loaded, v2, err := s.LoadDay(day1029)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Tasks) != 1 || loaded.Tasks[0].Title != "資料の作成" {
		t.Errorf("loaded = %+v", loaded)
	}
	if !v.same(v2) {
		t.Errorf("保存後と読み込み後の Version が違う: %+v, %+v", v, v2)
	}

	// 続けて保存できる。
	loaded.Tasks = append(loaded.Tasks, dayfile.Task{Title: "レビュー"})
	if _, err := s.SaveDay(loaded, v2); err != nil {
		t.Fatal(err)
	}
	assertNoTempFiles(t, filepath.Dir(s.DayPath(day1029)))
}

func TestSaveConflictWhenChangedElsewhere(t *testing.T) {
	s := New(t.TempDir())
	day, v, _ := s.LoadDay(day1029)
	day.Tasks = []dayfile.Task{{Title: "a"}}
	v, err := s.SaveDay(day, v)
	if err != nil {
		t.Fatal(err)
	}

	// 他の PC が書き換えた。
	path := s.DayPath(day1029)
	other := []byte("date: 2026-10-29\ntasks:\n  - title: 他の PC\n")
	if err := os.WriteFile(path, other, 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}

	day.Tasks = []dayfile.Task{{Title: "こちら"}}
	if _, err := s.SaveDay(day, v); !errors.Is(err, ErrConflict) {
		t.Fatalf("SaveDay error = %v, want ErrConflict", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(other) {
		t.Errorf("他の PC の変更が上書きされた:\n%s", got)
	}
}

func TestSaveConflictWhenCreatedElsewhere(t *testing.T) {
	s := New(t.TempDir())
	day, v, _ := s.LoadDay(day1029)

	path := s.DayPath(day1029)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("date: 2026-10-29\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	day.Tasks = []dayfile.Task{{Title: "a"}}
	if _, err := s.SaveDay(day, v); !errors.Is(err, ErrConflict) {
		t.Fatalf("SaveDay error = %v, want ErrConflict", err)
	}
}

func TestLoadDateMismatch(t *testing.T) {
	s := New(t.TempDir())
	path := s.DayPath(day1029)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("date: 2026-10-28\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.LoadDay(day1029); err == nil {
		t.Error("LoadDay succeeded, want error")
	}
}

func TestLoadMissingSettingsCreatesNothing(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	st, _, err := s.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if st.DurationFormat != settings.DefaultDurationFormat {
		t.Errorf("settings = %+v, want 省略時の値", st)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("読んだだけでファイルができた: %v", entries)
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	s := New(t.TempDir())
	st, v, err := s.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	st.Projects = []settings.Project{{Code: "DEV-001", Name: "開発"}}
	if _, err := s.SaveSettings(st, v); err != nil {
		t.Fatal(err)
	}

	loaded, _, err := s.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Projects) != 1 || loaded.Projects[0].Code != "DEV-001" {
		t.Errorf("loaded = %+v", loaded)
	}
	assertNoTempFiles(t, filepath.Dir(s.SettingsPath()))
}

// 設定を開いた後に、他の PC が settings.yaml を作っていたら上書きしない。
func TestSaveSettingsConflict(t *testing.T) {
	s := New(t.TempDir())
	st, v, err := s.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	other := []byte("projects:\n  - code: OTHER\n")
	if err := os.WriteFile(s.SettingsPath(), other, 0o644); err != nil {
		t.Fatal(err)
	}

	st.Projects = []settings.Project{{Code: "MINE"}}
	if _, err := s.SaveSettings(st, v); !errors.Is(err, ErrConflict) {
		t.Fatalf("SaveSettings error = %v, want ErrConflict", err)
	}
	got, err := os.ReadFile(s.SettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(other) {
		t.Errorf("他の PC の変更が上書きされた:\n%s", got)
	}
}

func TestLoadBrokenSettings(t *testing.T) {
	s := New(t.TempDir())
	if err := os.WriteFile(s.SettingsPath(), []byte("theme: dark\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.LoadSettings(); err == nil {
		t.Error("LoadSettings succeeded, want error")
	}
}

func TestLoadMissingItemsCreatesNothing(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	list, _, err := s.LoadItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("items = %+v", list)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("読んだだけでファイルができた: %v", entries)
	}
}

func TestSaveAndLoadItems(t *testing.T) {
	s := New(t.TempDir())
	list, v, err := s.LoadItems()
	if err != nil {
		t.Fatal(err)
	}
	list = append(list, items.Item{Title: "問い合わせ対応", Favorite: true})
	if _, err := s.SaveItems(list, v); err != nil {
		t.Fatal(err)
	}

	loaded, _, err := s.LoadItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Title != "問い合わせ対応" || !loaded[0].Favorite {
		t.Errorf("loaded = %+v", loaded)
	}
	assertNoTempFiles(t, filepath.Dir(s.ItemsPath()))
}

func TestSaveItemsConflict(t *testing.T) {
	s := New(t.TempDir())
	list, v, err := s.LoadItems()
	if err != nil {
		t.Fatal(err)
	}
	other := []byte("- title: 他の PC\n  favorite: true\n")
	if err := os.WriteFile(s.ItemsPath(), other, 0o644); err != nil {
		t.Fatal(err)
	}

	list = append(list, items.Item{Title: "こちら", Favorite: true})
	if _, err := s.SaveItems(list, v); !errors.Is(err, ErrConflict) {
		t.Fatalf("SaveItems error = %v, want ErrConflict", err)
	}
	got, err := os.ReadFile(s.ItemsPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(other) {
		t.Errorf("他の PC の変更が上書きされた:\n%s", got)
	}
}

func TestLoadBrokenItems(t *testing.T) {
	s := New(t.TempDir())
	if err := os.WriteFile(s.ItemsPath(), []byte("- title: a\n  done: 2026-10-20\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.LoadItems(); err == nil {
		t.Error("LoadItems succeeded, want error")
	}
}

// writeDay はテスト用に日次ファイルを置く。
func writeDay(t *testing.T, s *Store, date time.Time, content string) {
	t.Helper()
	path := s.DayPath(date)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func dayOf(m time.Month, d int) time.Time {
	return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC)
}

func TestLoadDays(t *testing.T) {
	s := New(t.TempDir())
	writeDay(t, s, dayOf(10, 31), "date: 2026-10-31\n")
	writeDay(t, s, dayOf(11, 2), "date: 2026-11-02\ntasks:\n  - title: a\n")
	writeDay(t, s, dayOf(11, 3), "date: 2026-11-03\nbroken: true\n")
	writeDay(t, s, dayOf(11, 5), "date: 2026-11-05\n")

	days, skipped := s.LoadDays(dayOf(11, 1), dayOf(11, 4))
	if len(days) != 1 || !days[0].Date.Equal(dayOf(11, 2)) {
		t.Errorf("days = %+v, want 11/2 だけ", days)
	}
	// 壊れた 11/3 は除き、理由を返す。
	if len(skipped) != 1 {
		t.Errorf("skipped = %v", skipped)
	}
}

func TestPreviousDay(t *testing.T) {
	s := New(t.TempDir())
	// 年をまたいで探す。
	writeDay(t, s, time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC), "date: 2025-12-30\nnotes:\n  在宅: true\n")
	writeDay(t, s, dayOf(1, 5), "date: 2026-01-05\nbroken: true\n")
	writeDay(t, s, dayOf(1, 8), "date: 2026-01-08\n")
	// 日次ファイルの形でない名前は無視する。
	writeDay(t, s, dayOf(1, 6), "date: 2026-01-06\n")
	if err := os.Rename(s.DayPath(dayOf(1, 6)), filepath.Join(filepath.Dir(s.DayPath(dayOf(1, 6))), "メモ.yaml")); err != nil {
		t.Fatal(err)
	}

	// 1/7 の前は 1/5 だが、読めないので飛ばして 2025/12/30 を返す。
	day, found, skipped, err := s.PreviousDay(dayOf(1, 7))
	if err != nil {
		t.Fatal(err)
	}
	if !found || !day.Date.Equal(time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("PreviousDay = %v (%v), want 2025-12-30", day.Date, found)
	}
	if len(skipped) != 1 {
		t.Errorf("skipped = %v", skipped)
	}

	// その日自身は「前」に入らない。
	day, found, _, err = s.PreviousDay(dayOf(1, 9))
	if err != nil || !found || !day.Date.Equal(dayOf(1, 8)) {
		t.Errorf("PreviousDay(1/9) = %v, %v, %v", day.Date, found, err)
	}
	if _, found, _, err := s.PreviousDay(dayOf(1, 8)); err != nil || !found {
		t.Errorf("PreviousDay(1/8) found = %v, err = %v", found, err)
	}
}

func TestPreviousDayEmptyFolder(t *testing.T) {
	s := New(t.TempDir())
	_, found, skipped, err := s.PreviousDay(dayOf(11, 5))
	if err != nil || found || len(skipped) != 0 {
		t.Errorf("PreviousDay = %v, %v, %v", found, skipped, err)
	}
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("一時ファイルが残っている: %s", e.Name())
		}
	}
}

func TestLoadTemplatesMissingFolder(t *testing.T) {
	lib, skipped, err := New(t.TempDir()).LoadTemplates()
	if err != nil || len(skipped) > 0 || len(lib.Outputs) > 0 || len(lib.Sets) > 0 {
		t.Errorf("LoadTemplates = %+v, %v, %v", lib, skipped, err)
	}
}

func TestLoadTemplates(t *testing.T) {
	s := New(t.TempDir())
	dir := s.TemplatesPath()
	if err := os.MkdirAll(filepath.Join(dir, "古い"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"日報テキスト.tmpl": "---\nfile: d\nextension: txt\n---\nx",
		"日報.set.yaml": "kind: set\noutputs: [日報テキスト]\n",
		"壊れた.tmpl":    "本文だけ",
		// フォルダの中は見ない。
		filepath.Join("古い", "古い日報.tmpl"): "---\nfile: d\nextension: txt\n---\nx",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	lib, skipped, err := s.LoadTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 1 {
		t.Errorf("skipped = %v", skipped)
	}
	if len(lib.Outputs) != 1 || lib.Outputs[0].Name != "日報テキスト" {
		t.Errorf("Outputs = %+v", lib.Outputs)
	}
	if _, err := lib.SetOutputs("日報"); err != nil {
		t.Errorf("SetOutputs: %v", err)
	}
}

// 自分で書いたファイルは、書いたときのままなら自分のものと分かる。他の PC が書き換えたら、もう自分のものではない。
func TestWrittenHere(t *testing.T) {
	s := New(t.TempDir())
	path := s.DayPath(day1029)
	before := time.Now().Add(-time.Second)
	if s.WrittenHere(path, time.Now()) {
		t.Error("まだ書いていないのに自分のもの")
	}
	day, v, _ := s.LoadDay(day1029)
	day.Tasks = []dayfile.Task{{Title: "a"}}
	if _, err := s.SaveDay(day, v); err != nil {
		t.Fatal(err)
	}
	if !s.WrittenHere(path, time.Now()) {
		t.Error("書いた直後なのに自分のものでない")
	}
	// 書き始める前に知らせが来ていたら、その前に他で書き換えたものがある。
	if s.WrittenHere(path, before) {
		t.Error("書く前に来た知らせまで自分のもの")
	}

	// 続けて書いたら、前の書き込みの知らせも自分のもの。
	between := time.Now()
	time.Sleep(10 * time.Millisecond)
	day, v, _ = s.LoadDay(day1029)
	day.Tasks = append(day.Tasks, dayfile.Task{Title: "b"})
	if _, err := s.SaveDay(day, v); err != nil {
		t.Fatal(err)
	}
	if !s.WrittenHere(path, between) {
		t.Error("続けて書いたら、前の書き込みの知らせが自分のものでない")
	}

	if err := os.WriteFile(path, []byte("date: 2026-10-29\ntasks:\n  - title: 他の PC\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
	if s.WrittenHere(path, time.Now()) {
		t.Error("他の PC が書き換えたのに自分のもの")
	}

	// 消したことも覚える。
	if err := s.WriteTemplate("a.tmpl", []byte("x")); err != nil {
		t.Fatal(err)
	}
	tmpl := filepath.Join(s.TemplatesPath(), "a.tmpl")
	if err := s.RemoveTemplate("a.tmpl"); err != nil {
		t.Fatal(err)
	}
	if !s.WrittenHere(tmpl, time.Now()) {
		t.Error("消した後なのに自分のものでない")
	}
}
