package gui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/localconf"
	"github.com/crepidoma/lumetio/internal/store"
	"github.com/crepidoma/lumetio/internal/watch"
)

// データフォルダを変えたら、新しいフォルダを監視し、前のフォルダの変更は知らせない。
func TestWatchFollowsDataDir(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	got := make(chan watch.Change, 10)
	s := New(app.New(store.New(first), out, nil), nil, Env{
		Local:     localconf.NewFile(filepath.Join(t.TempDir(), "local.yaml")),
		Locations: Locations{DataDir: first, OutputDir: out},
		Notify:    func(c watch.Change) { got <- c },
	})
	if err := s.Watch(); err != nil {
		t.Fatal(err)
	}
	defer s.StopWatch()

	wait := func(want bool) {
		t.Helper()
		select {
		case c := <-got:
			if !want {
				t.Errorf("知らせが来た: %+v", c)
			} else if !c.Settings {
				t.Errorf("Change = %+v", c)
			}
		case <-time.After(2 * time.Second):
			if want {
				t.Error("知らせが来ない")
			}
		}
	}
	write(t, first, "settings.yaml", "duration_format: H:mm\n")
	wait(true)

	if err := s.SaveLocations(Locations{DataDir: second, OutputDir: out}); err != nil {
		t.Fatal(err)
	}
	write(t, first, "settings.yaml", "duration_format: m'm'\n")
	wait(false)
	write(t, second, "settings.yaml", "duration_format: H:mm\n")
	wait(true)
}

// このアプリ自身が書いたファイルの知らせは届けない。書いた結果はもう画面に返しているので。
func TestWatchIgnoresOwnWrites(t *testing.T) {
	dir := t.TempDir()
	got := make(chan watch.Change, 10)
	s := New(app.New(store.New(dir), t.TempDir(), nil), nil, Env{
		Locations: Locations{DataDir: dir},
		Notify:    func(c watch.Change) { got <- c },
	})
	if err := s.Watch(); err != nil {
		t.Fatal(err)
	}
	defer s.StopWatch()

	if err := s.SaveDurationFormat("H:mm", "m'm'"); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-got:
		t.Errorf("自分の書き込みで知らせが来た: %+v", c)
	case <-time.After(time.Second):
	}

	// 他で書き換えたら知らせる。
	write(t, dir, "settings.yaml", "duration_format: H'h'\n")
	select {
	case c := <-got:
		if !c.Settings {
			t.Errorf("Change = %+v", c)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("知らせが来ない")
	}
}

// 他で書き換えた知らせを待っているあいだに自分で書いても、他の書き換えの知らせは届ける。
// 自分の書き込みは他の書き換えの上に書くので、画面には他の書き換えも読み直させるため。
func TestWatchKeepsOthersBeforeOwnWrite(t *testing.T) {
	dir := t.TempDir()
	got := make(chan watch.Change, 10)
	s := New(app.New(store.New(dir), t.TempDir(), nil), nil, Env{
		Locations: Locations{DataDir: dir},
		Notify:    func(c watch.Change) { got <- c },
	})
	if err := s.Watch(); err != nil {
		t.Fatal(err)
	}
	defer s.StopWatch()

	write(t, dir, "settings.yaml", "default_output: 日報\nduration_format: H:mm\n")
	// 知らせが監視に届くのを待つ。まとめて待つ時間より短くし、届ける前に自分で書く。
	time.Sleep(100 * time.Millisecond)
	if err := s.SaveDurationFormat("H:mm", "m'm'"); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-got:
		if !c.Settings {
			t.Errorf("Change = %+v", c)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("他の書き換えの知らせが来ない")
	}
}
