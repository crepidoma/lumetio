// Package localconf は、PC ごとのローカル設定 (local.yaml) を読み書きする。
// 共有しない、その PC だけの設定を持つ。
package localconf

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/crepidoma/lumetio/internal/atomicfile"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// Local はローカル設定。
// 場所は、書かれていなければ初期値にした後の値。
type Local struct {
	DataDir   string
	OutputDir string
	// Window は前回閉じたときのウィンドウ。書かれていなければ nil。
	Window *Window
	// Panes はペインの幅。書かれていなければ nil。
	Panes *Panes
}

// Window は、閉じたときのウィンドウの大きさと位置。
// 最大化していたときは、最大化する前の大きさと位置を持つ。
type Window struct {
	// Width と Height は、書かれていなければ 0。
	Width, Height int
	// X と Y は、HasPosition が true のときだけ使う。
	X, Y        int
	HasPosition bool
	Maximized   bool
}

// Panes は左と右のペインの幅 (px) と、右ペインを手で畳んでいるか。幅は書かれていなければ 0。
type Panes struct {
	Left, Right int
	// RightCollapsed は、右ペインを閉じるボタンで畳んでいるか。次に開いたときも畳んだままにするため。
	RightCollapsed bool
}

type rawLocal struct {
	DataDir   string     `yaml:"data_dir"`
	OutputDir string     `yaml:"output_dir"`
	Window    *rawWindow `yaml:"window"`
	Panes     *rawPanes  `yaml:"panes"`
}

type rawWindow struct {
	Width     *int `yaml:"width"`
	Height    *int `yaml:"height"`
	X         *int `yaml:"x"`
	Y         *int `yaml:"y"`
	Maximized bool `yaml:"maximized"`
}

type rawPanes struct {
	Left           *int `yaml:"left"`
	Right          *int `yaml:"right"`
	RightCollapsed bool `yaml:"right_collapsed"`
}

// appDir は、ローカル設定と初期のデータフォルダを置くフォルダ。
// Windows は %APPDATA%\lumetio、macOS は ~/Library/Application Support/lumetio。
func appDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("設定を置くフォルダが分かりません: %w", err)
	}
	return filepath.Join(dir, "lumetio"), nil
}

// Path は local.yaml の場所を返す。
func Path() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "local.yaml"), nil
}

// Defaults は、何も書かれていないときのローカル設定。
func Defaults() (Local, error) {
	dir, err := appDir()
	if err != nil {
		return Local{}, err
	}
	docs, err := documentsDir()
	if err != nil {
		return Local{}, fmt.Errorf("ドキュメントのフォルダが分かりません: %w", err)
	}
	return Local{
		DataDir:   filepath.Join(dir, "data"),
		OutputDir: filepath.Join(docs, "lumetio"),
	}, nil
}

// Load は local.yaml を読む。
// ファイルがなければ初期値を返し、ファイルは作らない。
func Load(path string) (Local, error) {
	l, err := Defaults()
	if err != nil {
		return Local{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return Local{}, fmt.Errorf("%s を読めません: %w", path, err)
	}
	if err := parse(data, &l); err != nil {
		return Local{}, fmt.Errorf("%s: %w", path, err)
	}
	return l, nil
}

// parse は local.yaml の中身を l に重ねる。
// 書かれていない項目は l の値のままにする。
func parse(data []byte, l *Local) error {
	var raw rawLocal
	if err := yamlfile.Decode(data, &raw); err != nil {
		return fmt.Errorf("local.yaml を読めません: %w", err)
	}
	for _, f := range []struct {
		key   string
		value string
		dst   *string
	}{
		{"data_dir", raw.DataDir, &l.DataDir},
		{"output_dir", raw.OutputDir, &l.OutputDir},
	} {
		if f.value == "" {
			continue
		}
		// 相対パスは、どこを基準にするか分かりにくい。
		if !filepath.IsAbs(f.value) {
			return fmt.Errorf("%s の %q は、ドライブやルートから書きます", f.key, f.value)
		}
		*f.dst = filepath.Clean(f.value)
	}
	if w := raw.Window; w != nil {
		l.Window = &Window{Width: deref(w.Width), Height: deref(w.Height), Maximized: w.Maximized}
		// 位置は x と y がそろっているときだけ使う。片方だけでは置き場所が決まらないため。
		if w.X != nil && w.Y != nil {
			l.Window.X, l.Window.Y, l.Window.HasPosition = *w.X, *w.Y, true
		}
	}
	if p := raw.Panes; p != nil {
		l.Panes = &Panes{Left: deref(p.Left), Right: deref(p.Right), RightCollapsed: p.RightCollapsed}
	}
	return nil
}

func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// File は、アプリが local.yaml の window と panes を書くためのもの。
// 画面 (ペインの幅) と、ウィンドウを閉じる処理の両方から書くので、1 つずつ順に書く。
type File struct {
	path string
	mu   sync.Mutex
}

// NewFile は path の local.yaml を書く File を作る。
func NewFile(path string) *File {
	return &File{path: path}
}

// SaveWindow は、ウィンドウの大きさと位置を書く。ほかの項目は書かれていたまま残す。
func (f *File) SaveWindow(w Window) error {
	return f.update(func(raw *rawLocal) {
		raw.Window = &rawWindow{Width: &w.Width, Height: &w.Height, Maximized: w.Maximized}
		if w.HasPosition {
			raw.Window.X, raw.Window.Y = &w.X, &w.Y
		}
	})
}

// SavePanes は、ペインの幅と、右ペインを畳んでいるかを書く。ほかの項目は書かれていたまま残す。
func (f *File) SavePanes(p Panes) error {
	return f.update(func(raw *rawLocal) {
		raw.Panes = &rawPanes{Left: &p.Left, Right: &p.Right, RightCollapsed: p.RightCollapsed}
	})
}

// SaveDirs は、データフォルダと出力先を書く。ほかの項目は書かれていたまま残す。
// 初期値と同じ場所は書かない。初期値が変わったときに追いつけるようにするため。
func (f *File) SaveDirs(dataDir, outputDir string) error {
	def, err := Defaults()
	if err != nil {
		return err
	}
	for _, dir := range []string{dataDir, outputDir} {
		if !filepath.IsAbs(dir) {
			return fmt.Errorf("%q は、ドライブやルートから書きます", dir)
		}
	}
	keep := func(dir, def string) string {
		if SameDir(dir, def) {
			return ""
		}
		return filepath.Clean(dir)
	}
	return f.update(func(raw *rawLocal) {
		raw.DataDir = keep(dataDir, def.DataDir)
		raw.OutputDir = keep(outputDir, def.OutputDir)
	})
}

// SameDir は、a と b が同じフォルダを指すかを返す。
// 書き方が違っても同じフォルダのことがあるため、文字列ではなくフォルダとして比べる (Windows の短い名前、大文字と小文字など)。
func SameDir(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	if errA == nil && errB == nil {
		return os.SameFile(ia, ib)
	}
	// まだないフォルダ (初めて起動する前のデータフォルダなど) は、ある親までを正しい書き方にそろえて比べる。
	a, b = resolve(a), resolve(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// resolve は、p のうち、ある親までを正しい書き方 (長い名前、実際の大文字と小文字) にする。
func resolve(p string) string {
	rest := ""
	for dir := p; ; {
		if r, err := filepath.EvalSymlinks(dir); err == nil {
			return filepath.Join(r, rest)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return p
		}
		rest = filepath.Join(filepath.Base(dir), rest)
		dir = parent
	}
}

// update は local.yaml を読んで change で変え、一時ファイルを通して書く。
// 書かれていない場所 (data_dir など) は書き足さない。初期値を書き込むと、初期値が変わったときに追いつけないため。
func (f *File) update(change func(*rawLocal)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	var raw rawLocal
	data, err := os.ReadFile(f.path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s を読めません: %w", f.path, err)
	}
	if err := yamlfile.Decode(data, &raw); err != nil {
		return fmt.Errorf("%s を読めません: %w", f.path, err)
	}
	change(&raw)
	out, err := encode(raw)
	if err != nil {
		return err
	}
	return atomicfile.Write(f.path, out)
}

// encode は local.yaml を、キーを決まった順に並べて書く。
func encode(raw rawLocal) ([]byte, error) {
	root := yamlfile.Mapping()
	if raw.DataDir != "" {
		yamlfile.Add(root, "data_dir", yamlfile.Text(raw.DataDir))
	}
	if raw.OutputDir != "" {
		yamlfile.Add(root, "output_dir", yamlfile.Text(raw.OutputDir))
	}
	if w := raw.Window; w != nil {
		m := yamlfile.Mapping()
		for _, f := range []struct {
			key   string
			value *int
		}{{"width", w.Width}, {"height", w.Height}, {"x", w.X}, {"y", w.Y}} {
			if f.value != nil {
				yamlfile.Add(m, f.key, yamlfile.Int(*f.value))
			}
		}
		yamlfile.Add(m, "maximized", yamlfile.Bool(w.Maximized))
		yamlfile.Add(root, "window", m)
	}
	if p := raw.Panes; p != nil {
		m := yamlfile.Mapping()
		if p.Left != nil {
			yamlfile.Add(m, "left", yamlfile.Int(*p.Left))
		}
		if p.Right != nil {
			yamlfile.Add(m, "right", yamlfile.Int(*p.Right))
		}
		yamlfile.Add(m, "right_collapsed", yamlfile.Bool(p.RightCollapsed))
		yamlfile.Add(root, "panes", m)
	}
	data, err := yamlfile.Encode(root)
	if err != nil {
		return nil, fmt.Errorf("local.yaml を書けません: %w", err)
	}
	return data, nil
}
