package gui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/localconf"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/store"
)

// Locations は、データフォルダと出力先。どちらも PC ごとのローカル設定に置く。
type Locations struct {
	DataDir   string `json:"dataDir"`
	OutputDir string `json:"outputDir"`
}

// Locations は、今のデータフォルダと出力先を返す。
func (s *Service) Locations() (Locations, error) {
	if s.startErr != nil {
		return Locations{}, s.startErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.locations, nil
}

// DefaultLocations は、データフォルダと出力先の初期値を返す。分からない方は空。
func (s *Service) DefaultLocations() Locations {
	return s.env.DefaultLocations
}

// LocationsAtDefault は、データフォルダと出力先が、それぞれ初期値のフォルダを指しているか。
type LocationsAtDefault struct {
	DataDir   bool `json:"dataDir"`
	OutputDir bool `json:"outputDir"`
}

// AtDefault は、l がそれぞれ初期値のフォルダを指しているかを返す。初期値が分からない方は false。
func (s *Service) AtDefault(l Locations) LocationsAtDefault {
	def := s.env.DefaultLocations
	return LocationsAtDefault{
		DataDir:   def.DataDir != "" && localconf.SameDir(l.DataDir, def.DataDir),
		OutputDir: def.OutputDir != "" && localconf.SameDir(l.OutputDir, def.OutputDir),
	}
}

// ChooseFolder は、フォルダを選ぶダイアログを出す。やめたら空を返す。
func (s *Service) ChooseFolder(title, current string) (string, error) {
	if s.env.ChooseFolder == nil {
		return "", errors.New("フォルダを選べません")
	}
	// ないフォルダから開き始めると、ダイアログが開けない。出力先の初期値のように、書き出すまで作らないフォルダがあるため。
	return s.env.ChooseFolder(title, existingDir(current))
}

// existingDir は、dir か、その親のうち、あるフォルダを返す。どれもなければ空。
func existingDir(dir string) string {
	if dir == "" {
		return ""
	}
	for d := filepath.Clean(dir); ; {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// SaveLocations は、データフォルダと出力先をローカル設定に書き、以後の操作を新しい場所で行う。
// ファイルは移さない。画面は書いた後に、すべてを読み直す。
func (s *Service) SaveLocations(l Locations) error {
	if s.startErr != nil {
		return s.startErr
	}
	if s.env.Local == nil {
		return errors.New("ローカル設定を書けません")
	}
	// データフォルダは、OneDrive などで同じフォルダを指すために選ぶもの。ないフォルダを指すと、空のデータで動き始めてしまう。
	// 初期値は、初めて起動したときと同じく空のデータから始める場所なので、なくてもよい。
	if !localconf.SameDir(l.DataDir, s.env.DefaultLocations.DataDir) || l.DataDir == "" {
		if info, err := os.Stat(l.DataDir); err != nil || !info.IsDir() {
			return &app.InputError{Message: "データフォルダが見つかりません"}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.env.Local.SaveDirs(l.DataDir, l.OutputDir); err != nil {
		return err
	}
	s.current.Store(app.New(store.New(l.DataDir), l.OutputDir, nil))
	s.locations = l
	// 検索の候補は前のデータフォルダから作ったもの。
	s.candidates = nil
	// 場所はもう変わっているので、監視を始められなくても、そのことだけを知らせる。
	if err := s.watchLocked(); err != nil {
		return fmt.Errorf("場所は保存しましたが、%w", err)
	}
	return nil
}

// DurationFormat は、settings.yaml に書かれた長さの表記を返す。
func (s *Service) DurationFormat() (string, error) {
	if s.startErr != nil {
		return "", s.startErr
	}
	return s.app().DurationFormat()
}

// DefaultDurationFormat は、長さの表記の初期値を返す。
func (s *Service) DefaultDurationFormat() string {
	return settings.DefaultDurationFormat
}

// SaveDurationFormat は長さの表記を書く。base は設定画面で読んだときの表記。
func (s *Service) SaveDurationFormat(base, layout string) error {
	if s.startErr != nil {
		return s.startErr
	}
	return s.app().SaveDurationFormat(base, layout)
}

// durationSamples は、長さの表記を確かめるときに書いてみる長さ (分)。端数の出方が違うものを選んだ。
var durationSamples = []int{90, 75, 50}

// DurationSample は、ある長さを書式で書いた結果。
type DurationSample struct {
	Minutes int    `json:"minutes"`
	Text    string `json:"text"`
}

// PreviewDurationFormat は、入力中の長さの表記で、いくつかの長さを書いた結果を返す。
// 空なら、保存したときと同じく省略時の表記で書く。
func (s *Service) PreviewDurationFormat(layout string) ([]DurationSample, error) {
	_, f, err := settings.ParseDurationFormat(layout)
	if err != nil {
		return nil, &app.InputError{Message: err.Error()}
	}
	out := []DurationSample{}
	for _, m := range durationSamples {
		out = append(out, DurationSample{Minutes: m, Text: f.Format(m)})
	}
	return out, nil
}
