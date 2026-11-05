package app

import (
	"bytes"
	"encoding/csv"
	"os"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/crepidoma/lumetio/internal/datefmt"
	"github.com/crepidoma/lumetio/internal/settings"
)

// AllProjects は、期間の外のものも含めたプロジェクトの一覧を、settings.yaml の並びで返す。
func (a *App) AllProjects() ([]settings.Project, error) {
	st, _, err := a.store.LoadSettings()
	if err != nil {
		return nil, err
	}
	return st.Projects, nil
}

// SaveProjects はプロジェクトの一覧を書く。コードと名前は前後の空白を除き、NFC にそろえる。
// base は設定画面で読んだときの一覧。他で一覧が変わっていたら書かずに ErrConflict を返す。
// 比べるのは一覧だけにし、settings.yaml のほかの項目は、その時点の中身のまま残す。
func (a *App) SaveProjects(base, projects []settings.Project) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	projects = trimProjects(projects)
	if err := checkProjects(projects, a.Today()); err != nil {
		return err
	}
	return a.updateSettings(
		func(st settings.Settings) bool { return sameProjects(st.Projects, base) },
		func(st *settings.Settings) { st.Projects = projects },
	)
}

func trimProjects(ps []settings.Project) []settings.Project {
	out := make([]settings.Project, 0, len(ps))
	for _, p := range ps {
		p.Code = cleanName(p.Code)
		p.Name = cleanName(p.Name)
		out = append(out, p)
	}
	return out
}

// checkProjects は、settings.yaml として読めない一覧 (コードが空、同じコード、開始日が期限より後) を入力の誤りにする。
// 誤りのある行は、行の番号でなく中身で指す。画面では期限切れの行を隠したり末尾に出したりするので、番号が画面の並びと合わないため。
// 重なった相手が期限切れなら、そう書く。画面では期限切れの行を隠すので、相手の行が見えないことがあるため。
func checkProjects(ps []settings.Project, today time.Time) error {
	codes := map[string]settings.Project{}
	for _, p := range ps {
		if p.Code == "" {
			if p.Name != "" {
				return inputErrorf("名前 %q のコードが空です", p.Name)
			}
			return inputErrorf("コードが空の行があります")
		}
		if q, ok := codes[p.Code]; ok {
			if Expired(q, today) || Expired(p, today) {
				return inputErrorf("コード %q が、期限切れのプロジェクトと重なっています", p.Code)
			}
			return inputErrorf("コード %q が重なっています", p.Code)
		}
		codes[p.Code] = p
		if !p.Start.IsZero() && !p.Until.IsZero() && p.Start.After(p.Until) {
			return inputErrorf("コード %q の開始日が期限より後です", p.Code)
		}
	}
	return nil
}

// Expired は、期限が today より前か。期限の日はまだ使える。
// 設定画面で隠す行と、保存の誤りで期限切れと書く行を、同じ決まりで選ぶため。
func Expired(p settings.Project, today time.Time) bool {
	return !p.Until.IsZero() && today.After(p.Until)
}

func sameProjects(a, b []settings.Project) bool {
	return slices.EqualFunc(a, b, func(x, y settings.Project) bool {
		return x.Code == y.Code && x.Name == y.Name && x.Start.Equal(y.Start) && x.Until.Equal(y.Until)
	})
}

// ReadCSV は CSV を UTF-8 として読む。
func ReadCSV(path string) ([][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Excel で UTF-8 として保存すると先頭に付く BOM は、1 列目の値に混ざらないよう除く。
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		return nil, inputErrorf("UTF-8 の CSV ではありません")
	}
	// macOS で作った CSV は濁点などを分けて書くことがあるので、NFC にそろえる。重なりの確かめと、settings.yaml を読み直したときの結果を合わせるため。
	r := csv.NewReader(strings.NewReader(nfc(string(data))))
	// 行ごとに列の数が違っても読む。足りない列は空として扱う。
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, inputErrorf("CSV を読めません: %v", err)
	}
	return rows, nil
}

// ProjectImport は、CSV からプロジェクトを取り込むときの指定。
// 列は 0 から数え、-1 ならその列を使わない。
type ProjectImport struct {
	SkipHeader  bool
	CodeColumn  int
	NameColumn  int
	StartColumn int
	UntilColumn int
	// DateFormat は、開始日と期限の列の日付の書式。
	DateFormat string
	// Replace なら一覧ごと置き換え、そうでなければ今の一覧に足す。
	Replace bool
}

// ImportProjects は、CSV の行をプロジェクトとして取り込んだ一覧を返す。
// 読めない行が 1 つでもあれば、何も取り込まずに、何行目が読めないかを返す。
func ImportProjects(rows [][]string, opt ProjectImport, current []settings.Project) ([]settings.Project, error) {
	if opt.CodeColumn < 0 {
		return nil, inputErrorf("コードの列を選んでください")
	}
	var layout datefmt.Format
	if opt.StartColumn >= 0 || opt.UntilColumn >= 0 {
		f, err := datefmt.Parse(opt.DateFormat)
		if err == nil {
			err = f.CheckReadable()
		}
		if err != nil {
			return nil, inputErrorf("日付の書式: %v", err)
		}
		layout = f
	}

	first := 0
	if opt.SkipHeader {
		first = 1
	}
	var imported []settings.Project
	index := map[string]int{}
	for i := first; i < len(rows); i++ {
		row := rows[i]
		// Excel は、値のない行を ,,, のように書き出すことがある。
		if allBlank(row) {
			continue
		}
		cell := func(c int) string {
			if c < 0 || c >= len(row) {
				return ""
			}
			return row[c]
		}
		p := settings.Project{
			Code: strings.TrimSpace(cell(opt.CodeColumn)),
			Name: strings.TrimSpace(cell(opt.NameColumn)),
		}
		if p.Code == "" {
			return nil, inputErrorf("%d 行目のコードが空です", i+1)
		}
		date := func(column int, label string) (time.Time, error) {
			s := cell(column)
			if strings.TrimSpace(s) == "" {
				return time.Time{}, nil
			}
			d, err := layout.ParseDate(s)
			if err != nil {
				return time.Time{}, inputErrorf("%d 行目の%s %q を %s で読めません", i+1, label, s, opt.DateFormat)
			}
			return d, nil
		}
		var err error
		if p.Start, err = date(opt.StartColumn, "開始日"); err != nil {
			return nil, err
		}
		if p.Until, err = date(opt.UntilColumn, "期限"); err != nil {
			return nil, err
		}
		if !p.Start.IsZero() && !p.Until.IsZero() && p.Start.After(p.Until) {
			return nil, inputErrorf("%d 行目の開始日が期限より後です", i+1)
		}
		// CSV の中に同じコードが何行かあれば、後の行を使う。並びは最初に出てきた位置のまま。
		if j, ok := index[p.Code]; ok {
			imported[j] = p
			continue
		}
		index[p.Code] = len(imported)
		imported = append(imported, p)
	}
	// 取り込むものがないまま置き換えると、一覧が空になってしまう。
	if len(imported) == 0 {
		return nil, inputErrorf("取り込む行がありません")
	}

	if opt.Replace {
		return imported, nil
	}
	out := slices.Clone(current)
	pos := map[string]int{}
	for i, p := range out {
		pos[p.Code] = i
	}
	for _, p := range imported {
		if j, ok := pos[p.Code]; ok {
			out[j] = p
			continue
		}
		pos[p.Code] = len(out)
		out = append(out, p)
	}
	return out, nil
}

func allBlank(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
