// Package watch は、データフォルダのファイルが変わったことを、OS の知らせ (fsnotify) で受け取る。
// 他の PC が OneDrive などを通して書き換えたものを、画面に読み直させるため。
package watch

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Change は、まとめて届ける変更。
type Change struct {
	// Dates は、変わった日次ファイルの日付 (yyyy-MM-dd)。
	Dates     []string `json:"dates"`
	Items     bool     `json:"items"`
	Settings  bool     `json:"settings"`
	Templates bool     `json:"templates"`
}

func (c Change) empty() bool {
	return len(c.Dates) == 0 && !c.Items && !c.Settings && !c.Templates
}

// delay は、知らせをまとめて待つ時間。
// 1 回の保存でも、一時ファイルを書いて置き換えるので、知らせがいくつも続けて来るため。
const delay = 300 * time.Millisecond

// maxDelay は、知らせが途切れなくても届けるまでの時間。
// OneDrive がたくさんのファイルを同期しているあいだも、画面が読み直されないままにならないようにするため。
const maxDelay = 2 * time.Second

var dayFile = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\.yaml$`)

// pendingChange は、まだ届けていない 1 つのファイルの変更。
type pendingChange struct {
	c Change
	// since は、このファイルの最初の知らせが来た時刻。
	since time.Time
}

// Watcher はデータフォルダの監視。
type Watcher struct {
	root   string
	notify func(Change)
	// ignore は、since から来た path の知らせを捨てるか。このアプリ自身が書いたファイルの知らせで、画面を読み直さないため。
	ignore func(path string, since time.Time) bool
	fs     *fsnotify.Watcher

	mu sync.Mutex
	// pending は、まだ届けていない変わったファイルと、その種類と最初の知らせが来た時刻。
	// 自分の書き込みかは、知らせを受けたときでなく届けるときに見分ける。書いた直後の知らせは、書いた後の状態を覚えるより先に来るため。
	pending map[string]pendingChange
	// since は、まだ届けていない最初の知らせが来た時刻。
	since time.Time
	timer *time.Timer
	done  chan struct{}
}

// Start は root の監視を始める。root がなければ作る。
// fsnotify はフォルダの中まではたどらないので、days/ と年のフォルダ、templates/ もそれぞれ監視する。
// ignore が true を返したファイルの知らせは届けない。nil ならすべて届ける。
func Start(root string, notify func(Change), ignore func(path string, since time.Time) bool) (*Watcher, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{root: filepath.Clean(root), notify: notify, ignore: ignore, fs: fw, done: make(chan struct{})}
	if err := fw.Add(w.root); err != nil {
		fw.Close()
		return nil, err
	}
	w.addTree(filepath.Join(w.root, "days"))
	w.add(filepath.Join(w.root, "templates"))
	go w.loop()
	return w, nil
}

// Close は監視をやめる。まとめて待っている知らせは届けない。
func (w *Watcher) Close() error {
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.mu.Unlock()
	close(w.done)
	return w.fs.Close()
}

// add はフォルダを監視に足す。まだないフォルダは、できたときに足すので、何もしない。
func (w *Watcher) add(dir string) {
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		_ = w.fs.Add(dir)
	}
}

// addTree は days/ と、その中の年のフォルダを監視に足す。
func (w *Watcher) addTree(dir string) {
	w.add(dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			w.add(filepath.Join(dir, e.Name()))
		}
	}
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			w.handle(ev)
		case _, ok := <-w.fs.Errors:
			// 知らせを取りこぼしても、保存するときに他の変更は確かめるので、止めずに続ける。
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) handle(ev fsnotify.Event) {
	rel, err := filepath.Rel(w.root, ev.Name)
	if err != nil {
		return
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	// 後からできた days/、年のフォルダ、templates/ も監視に足す。
	if ev.Has(fsnotify.Create) {
		switch {
		case len(parts) == 1 && parts[0] == "days":
			w.addTree(ev.Name)
			w.queueExisting(ev.Name)
		case len(parts) == 1 && parts[0] == "templates":
			w.add(ev.Name)
			w.queueExisting(ev.Name)
		case len(parts) == 2 && parts[0] == "days":
			w.add(ev.Name)
			w.queueExisting(ev.Name)
		}
	}
	// templates/ のフォルダそのものは、なくなったときだけ知らせる。できたときと中が変わったとき (Windows では
	// フォルダの書き換えとして来る) は、中のファイルで知らせる。このアプリが書いたファイルなら、知らせずに済ませるため。
	if len(parts) == 1 && parts[0] == "templates" && !ev.Has(fsnotify.Remove) && !ev.Has(fsnotify.Rename) {
		return
	}
	if c := classify(parts); !c.empty() {
		w.queue(ev.Name, c)
	}
}

// queueExisting は、後から監視に足したフォルダに、足す前にできていたファイルを変わったものとする。
// 同期でフォルダとファイルがほぼ同時にできると、監視に足す前のファイルの知らせを取りこぼすため。
func (w *Watcher) queueExisting(dir string) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if rel, err := filepath.Rel(w.root, path); err == nil {
			if c := classify(strings.Split(filepath.ToSlash(rel), "/")); !c.empty() {
				w.queue(path, c)
			}
		}
		return nil
	})
}

// classify は、データフォルダからの相対パスを、変わったものの種類にする。
// 一時ファイルや知らないファイルは、何も変わっていないものとする。
func classify(parts []string) Change {
	switch {
	case len(parts) == 1 && parts[0] == "settings.yaml":
		return Change{Settings: true}
	case len(parts) == 1 && parts[0] == "items.yaml":
		return Change{Items: true}
	case len(parts) >= 1 && parts[0] == "templates":
		// templates/ のフォルダごと消えたりできたりしたときも、一覧が変わる。
		if len(parts) == 2 && strings.HasPrefix(parts[1], ".") {
			return Change{}
		}
		return Change{Templates: true}
	case len(parts) == 3 && parts[0] == "days":
		if m := dayFile.FindStringSubmatch(parts[2]); m != nil && strings.HasPrefix(m[1], parts[1]+"-") {
			return Change{Dates: []string{m[1]}}
		}
	}
	return Change{}
}

// queue は変更を貯め、しばらく次の知らせが来なければまとめて届ける。
// 知らせが続いても、最初の知らせから maxDelay たったら届ける。
func (w *Watcher) queue(path string, c Change) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	if len(w.pending) == 0 {
		w.since = now
		w.pending = map[string]pendingChange{}
	}
	p, ok := w.pending[path]
	if !ok {
		p.since = now
	}
	p.c = c
	w.pending[path] = p
	if w.timer != nil {
		w.timer.Stop()
	}
	wait := min(delay, max(0, maxDelay-now.Sub(w.since)))
	w.timer = time.AfterFunc(wait, w.flush)
}

func (w *Watcher) flush() {
	w.mu.Lock()
	pending := w.pending
	w.pending = nil
	w.mu.Unlock()
	select {
	case <-w.done:
		return
	default:
	}
	var c Change
	for path, p := range pending {
		if w.ignore != nil && w.ignore(path, p.since) {
			continue
		}
		pc := p.c
		for _, d := range pc.Dates {
			if !slices.Contains(c.Dates, d) {
				c.Dates = append(c.Dates, d)
			}
		}
		c.Items = c.Items || pc.Items
		c.Settings = c.Settings || pc.Settings
		c.Templates = c.Templates || pc.Templates
	}
	slices.Sort(c.Dates)
	if !c.empty() {
		w.notify(c)
	}
}
