package watch

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	tests := map[string]Change{
		"settings.yaml":              {Settings: true},
		"items.yaml":                 {Items: true},
		"templates/日報.tmpl":          {Templates: true},
		"templates":                  {Templates: true},
		"templates/.日報.tmpl.123.tmp": {},
		"days/2026/2026-11-04.yaml":  {Dates: []string{"2026-11-04"}},
		// 年のフォルダと日付が合わないもの、日次ファイルでないものは無視する。
		"days/2025/2026-11-04.yaml": {},
		"days/2026/メモ.txt":          {},
		"local.yaml":                {},
	}
	for path, want := range tests {
		if got := classify(strings.Split(path, "/")); !reflect.DeepEqual(got, want) {
			t.Errorf("classify(%q) = %+v, want %+v", path, got, want)
		}
	}
}

// 後からできたフォルダも監視し、続けて来た知らせを 1 つにまとめて届ける。
func TestWatch(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	got := make(chan Change, 10)
	w, err := Start(root, func(c Change) { got <- c }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if err := os.MkdirAll(filepath.Join(root, "days", "2026"), 0o755); err != nil {
		t.Fatal(err)
	}
	// フォルダができた知らせを受けて監視に足すまで、少し待つ。
	time.Sleep(200 * time.Millisecond)
	write := func(rel string) {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("days/2026/2026-11-04.yaml")
	write("items.yaml")

	select {
	case c := <-got:
		if !reflect.DeepEqual(c, Change{Dates: []string{"2026-11-04"}, Items: true}) {
			t.Errorf("Change = %+v", c)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("知らせが来ない")
	}
	select {
	case c := <-got:
		t.Errorf("知らせがまとまっていない: %+v", c)
	case <-time.After(2 * delay):
	}
}

// フォルダとファイルが一度にできても (同期で年のフォルダごと届いたときなど)、中の日次ファイルの変更を取りこぼさない。
func TestWatchFolderMovedIn(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	got := make(chan Change, 10)
	w, err := Start(root, func(c Change) { got <- c }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	outside := filepath.Join(t.TempDir(), "days")
	if err := os.MkdirAll(filepath.Join(outside, "2027"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "2027", "2027-01-04.yaml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(outside, filepath.Join(root, "days")); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-got:
		if !reflect.DeepEqual(c.Dates, []string{"2027-01-04"}) {
			t.Errorf("Change = %+v", c)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("知らせが来ない")
	}
}

// 知らせが途切れなくても、最初の知らせから maxDelay ほどで届ける。
func TestQueueMaxDelay(t *testing.T) {
	got := make(chan Change, 10)
	w := &Watcher{notify: func(c Change) { got <- c }, done: make(chan struct{})}
	start := time.Now()
	stop := time.After(maxDelay + time.Second)
	tick := time.NewTicker(delay / 3)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			w.queue("items.yaml", Change{Items: true})
		case <-got:
			if d := time.Since(start); d > maxDelay+delay {
				t.Errorf("届くまで %v かかった", d)
			}
			return
		case <-stop:
			t.Fatal("知らせが続くあいだ届かない")
		}
	}
}

// ignore が true を返したファイルの知らせは届けない。このアプリ自身が書いたファイルの知らせで、画面を読み直さないため。
func TestWatchIgnore(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	got := make(chan Change, 10)
	mine := filepath.Join(root, "items.yaml")
	w, err := Start(root, func(c Change) { got <- c }, func(path string, _ time.Time) bool { return path == mine })
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.WriteFile(mine, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "settings.yaml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-got:
		if !reflect.DeepEqual(c, Change{Settings: true}) {
			t.Errorf("Change = %+v", c)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("知らせが来ない")
	}
}

// templates/ ができたときは、中のファイルで知らせる。このアプリが最初の出力を書いてできたフォルダなら、知らせない。
func TestWatchIgnoreNewTemplatesFolder(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	got := make(chan Change, 10)
	dir := filepath.Join(root, "templates")
	mine := filepath.Join(dir, "a.tmpl")
	w, err := Start(root, func(c Change) { got <- c }, func(path string, _ time.Time) bool { return path == mine })
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mine, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-got:
		t.Errorf("自分の書き込みで知らせが来た: %+v", c)
	case <-time.After(time.Second):
	}
}
