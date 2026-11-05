// Package store はデータフォルダのファイルを読み書きする。
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/crepidoma/lumetio/internal/atomicfile"
	"github.com/crepidoma/lumetio/internal/dayfile"
	"github.com/crepidoma/lumetio/internal/items"
	"github.com/crepidoma/lumetio/internal/output"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// ErrConflict は、読み込んだ後に他の PC などでファイルが変わっていたことを表す。
// 呼び出し側は上書きせずに読み直し、そのことを知らせる。
var ErrConflict = errors.New("ファイルが他で変更されています")

// Store は 1 つのデータフォルダを表す。
type Store struct {
	root string

	mu sync.Mutex
	// written は、このアプリが最後に書いた (消した) ファイルの、書き始めた時刻と書いた後の状態。
	// 監視の知らせが、自分の書き込みか、他の PC などの書き込みかを見分けるため。
	written map[string]write
}

type write struct {
	at time.Time
	v  Version
}

// New は root をデータフォルダとする Store を作る。
func New(root string) *Store {
	return &Store{root: root, written: map[string]write{}}
}

// WrittenHere は、since から path に来た変更が、このアプリの書き込みだけかを返す。
// since はそのファイルの最初の知らせが来た時刻。自分で書いたファイルの知らせで、画面を読み直さないため。
// 書き始めるより前に知らせが来ていたら、その前に他で書き換えたものがあるので、自分の書き込みだけとはしない。
func (s *Store) WrittenHere(path string, since time.Time) bool {
	v, err := stat(path)
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.written[filepath.Clean(path)]
	return ok && w.v.same(v) && !since.Before(w.at)
}

// remember は、at に書き始めて (消し始めて) 書いた後のファイルの状態 v を覚える。before は書く前の状態。
// 書く前が前に自分で書いたままなら、書き始めた時刻は前の書き込みのものを残す。
// 続けて書くと、前の書き込みの知らせが次の書き始めより前に来るので、その知らせも自分のものと分かるようにするため。
func (s *Store) remember(path string, at time.Time, before, v Version) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := filepath.Clean(path)
	if prev, ok := s.written[key]; ok && prev.v.same(before) {
		at = prev.at
	}
	s.written[key] = write{at: at, v: v}
}

// save は、ファイルが base から変わっていなければ data を書き、書いた後の状態を覚える。
func (s *Store) save(path string, data []byte, base Version) (Version, error) {
	at := time.Now()
	v, err := save(path, data, base)
	if err == nil {
		s.remember(path, at, base, v)
	}
	return v, err
}

// Version は、読み込んだ時点のファイルの状態。
// 保存の直前に比べて、他で変わっていないか確かめる。
type Version struct {
	exists  bool
	modTime time.Time
	size    int64
}

// Exists は、読んだ時点でファイルがあったか。
func (v Version) Exists() bool {
	return v.exists
}

func (v Version) same(o Version) bool {
	return v.exists == o.exists && v.modTime.Equal(o.modTime) && v.size == o.size
}

// DayPath は日付に対応する日次ファイルの場所を返す。
func (s *Store) DayPath(date time.Time) string {
	return filepath.Join(s.root, "days", date.Format("2006"), date.Format(yamlfile.DateLayout)+".yaml")
}

// SettingsPath は settings.yaml の場所を返す。
func (s *Store) SettingsPath() string {
	return filepath.Join(s.root, "settings.yaml")
}

// ItemsPath は items.yaml の場所を返す。
func (s *Store) ItemsPath() string {
	return filepath.Join(s.root, "items.yaml")
}

// LoadDay は日次ファイルを読む。
// ファイルがなければ空の日を返し、ファイルは作らない。
func (s *Store) LoadDay(date time.Time) (dayfile.Day, Version, error) {
	path := s.DayPath(date)
	data, v, err := readFile(path)
	if err != nil {
		return dayfile.Day{}, Version{}, err
	}
	if !v.exists {
		return dayfile.Day{Date: yamlfile.DateOnly(date)}, v, nil
	}

	day, err := dayfile.Parse(data)
	if err != nil {
		return dayfile.Day{}, Version{}, fmt.Errorf("%s: %w", path, err)
	}
	if !day.Date.Equal(yamlfile.DateOnly(date)) {
		return dayfile.Day{}, Version{}, fmt.Errorf("%s: 中の日付 %s がファイル名と違います", path, day.Date.Format(yamlfile.DateLayout))
	}
	return day, v, nil
}

// LoadDays は from から to まで (両端を含む) の日次ファイルを、日付の古い順に読む。
// ファイルのない日は飛ばす。
// 読めないファイルは除いて skipped に理由を返す。
// 古い日のファイルが 1 つ壊れているだけで、左ペインなどが全部出なくなるのを避けるため。
func (s *Store) LoadDays(from, to time.Time) (days []dayfile.Day, skipped []error) {
	for d := yamlfile.DateOnly(from); !d.After(yamlfile.DateOnly(to)); d = d.AddDate(0, 0, 1) {
		day, v, err := s.LoadDay(d)
		if err != nil {
			skipped = append(skipped, err)
			continue
		}
		if v.exists {
			days = append(days, day)
		}
	}
	return days, skipped
}

// PreviousDay は、date より前で日次ファイルがある一番近い日を読む。
// 読めないファイルは飛ばしてさらに前を探し、skipped に理由を返す。
// err は、データフォルダの中を調べられなかったときだけ返す。
func (s *Store) PreviousDay(date time.Time) (day dayfile.Day, found bool, skipped []error, err error) {
	dates, err := s.DayDates()
	if err != nil {
		return dayfile.Day{}, false, nil, err
	}
	limit := yamlfile.DateOnly(date)
	for i := len(dates) - 1; i >= 0; i-- {
		if !dates[i].Before(limit) {
			continue
		}
		day, _, err := s.LoadDay(dates[i])
		if err != nil {
			skipped = append(skipped, err)
			continue
		}
		return day, true, skipped, nil
	}
	return dayfile.Day{}, false, skipped, nil
}

// DayDates は、日次ファイルがある日付を古い順に返す。
// 名前が日次ファイルの形でないものは無視する。
func (s *Store) DayDates() ([]time.Time, error) {
	root := filepath.Join(s.root, "days")
	years, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s を調べられません: %w", root, err)
	}
	var dates []time.Time
	for _, y := range years {
		if !y.IsDir() {
			continue
		}
		dir := filepath.Join(root, y.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("%s を調べられません: %w", dir, err)
		}
		for _, f := range files {
			name, ok := strings.CutSuffix(f.Name(), ".yaml")
			if !ok || f.IsDir() {
				continue
			}
			d, err := time.Parse(yamlfile.DateLayout, name)
			if err != nil || d.Format("2006") != y.Name() {
				continue
			}
			dates = append(dates, d)
		}
	}
	slices.SortFunc(dates, time.Time.Compare)
	return dates, nil
}

// SaveDay は日次ファイルを書く。
// base は読み込んだときの Version。
// ファイルがその後に変わっていたら、書かずに ErrConflict を返す。
func (s *Store) SaveDay(day dayfile.Day, base Version) (Version, error) {
	data, err := dayfile.Marshal(day)
	if err != nil {
		return Version{}, err
	}
	return s.save(s.DayPath(day.Date), data, base)
}

// LoadSettings は settings.yaml を読む。
// ファイルがなければ省略時の設定を返し、ファイルは作らない。
func (s *Store) LoadSettings() (settings.Settings, Version, error) {
	path := s.SettingsPath()
	data, v, err := readFile(path)
	if err != nil {
		return settings.Settings{}, Version{}, err
	}
	if !v.exists {
		return settings.Default(), v, nil
	}

	st, err := settings.Parse(data)
	if err != nil {
		return settings.Settings{}, Version{}, fmt.Errorf("%s: %w", path, err)
	}
	return st, v, nil
}

// SaveSettings は settings.yaml を書く。
// base は読み込んだときの Version。
// ファイルがその後に変わっていたら、書かずに ErrConflict を返す。
func (s *Store) SaveSettings(st settings.Settings, base Version) (Version, error) {
	data, err := settings.Marshal(st)
	if err != nil {
		return Version{}, err
	}
	return s.save(s.SettingsPath(), data, base)
}

// LoadItems は items.yaml を読む。
// ファイルがなければ空の一覧を返し、ファイルは作らない。
func (s *Store) LoadItems() ([]items.Item, Version, error) {
	path := s.ItemsPath()
	data, v, err := readFile(path)
	if err != nil {
		return nil, Version{}, err
	}
	if !v.exists {
		return nil, v, nil
	}

	list, err := items.Parse(data)
	if err != nil {
		return nil, Version{}, fmt.Errorf("%s: %w", path, err)
	}
	return list, v, nil
}

// SaveItems は items.yaml を書く。
// base は読み込んだときの Version。
// ファイルがその後に変わっていたら、書かずに ErrConflict を返す。
func (s *Store) SaveItems(list []items.Item, base Version) (Version, error) {
	data, err := items.Marshal(list)
	if err != nil {
		return Version{}, err
	}
	return s.save(s.ItemsPath(), data, base)
}

// TemplatesPath は、出力と出力セットを置くフォルダの場所を返す。
func (s *Store) TemplatesPath() string {
	return filepath.Join(s.root, "templates")
}

// LoadTemplates は出力と出力セットを読む。
// フォルダがなければ空を返す。
// 読めないファイルは除いて、skipped に理由を返す。
// err は、フォルダの中を調べられなかったときだけ返す。
func (s *Store) LoadTemplates() (lib output.Library, skipped []error, err error) {
	files, skipped, err := s.TemplateFiles()
	if err != nil {
		return output.Library{}, nil, err
	}
	lib, errs := output.Load(files)
	return lib, append(skipped, errs...), nil
}

// TemplateFiles は、templates/ の中のファイルを、ファイル名の順に読む。
// フォルダがなければ空を返す。読めないファイルは除いて、skipped に理由を返す。
// err は、フォルダの中を調べられなかったときだけ返す。
func (s *Store) TemplateFiles() (files []output.File, skipped []error, err error) {
	dir := s.TemplatesPath()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("%s を調べられません: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			skipped = append(skipped, fmt.Errorf("%s を読めません: %w", path, err))
			continue
		}
		files = append(files, output.File{Name: e.Name(), Data: data})
	}
	return files, skipped, nil
}

// WriteTemplate は、templates/ にファイルを書く。フォルダがなければ作る。
// 他で変わっていないかは、呼ぶ側が中身を比べて確かめる。
func (s *Store) WriteTemplate(fileName string, data []byte) error {
	path := filepath.Join(s.TemplatesPath(), fileName)
	at := time.Now()
	before, _ := stat(path)
	if err := atomicfile.Write(path, data); err != nil {
		return err
	}
	if v, err := stat(path); err == nil {
		s.remember(path, at, before, v)
	}
	return nil
}

// RemoveTemplate は、templates/ のファイルを消す。
func (s *Store) RemoveTemplate(fileName string) error {
	path := filepath.Join(s.TemplatesPath(), fileName)
	at := time.Now()
	before, _ := stat(path)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("%s を消せません: %w", path, err)
	}
	// 消したことも覚える。ないファイルの状態は「ない」なので、消した知らせも自分のものと分かる。
	s.remember(path, at, before, Version{})
	return nil
}

// readFile はファイルの中身と、読んだ時点の Version を返す。
// ファイルがなければ中身は空。
func readFile(path string) ([]byte, Version, error) {
	v, err := stat(path)
	if err != nil || !v.exists {
		return nil, v, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, Version{}, fmt.Errorf("%s を読めません: %w", path, err)
	}
	return data, v, nil
}

// save は、ファイルが base から変わっていなければ data を書く。
// 他の PC などで変わっていたら、上書きせずに ErrConflict を返す。
func save(path string, data []byte, base Version) (Version, error) {
	current, err := stat(path)
	if err != nil {
		return Version{}, err
	}
	if !current.same(base) {
		return Version{}, ErrConflict
	}
	if err := atomicfile.Write(path, data); err != nil {
		return Version{}, err
	}
	return stat(path)
}

func stat(path string) (Version, error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Version{}, nil
	}
	if err != nil {
		return Version{}, fmt.Errorf("%s を確かめられません: %w", path, err)
	}
	return Version{exists: true, modTime: info.ModTime(), size: info.Size()}, nil
}
