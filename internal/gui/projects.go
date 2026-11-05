package gui

import (
	"errors"
	"fmt"
	"time"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/settings"
	"github.com/crepidoma/lumetio/internal/yamlfile"
)

// ProjectRow は、設定画面で直すプロジェクトの一覧の 1 行。
type ProjectRow struct {
	Code string `json:"code"`
	Name string `json:"name"`
	// Start と Until は yyyy-MM-dd。空ならその日はない。
	Start string `json:"start"`
	Until string `json:"until"`
	// Expired は、読んだときに期限が今日より前だったこと。設定画面で、終わったプロジェクトを隠すため。保存するときは使わない。
	Expired bool `json:"expired"`
}

func toProjectRows(ps []settings.Project, today time.Time) []ProjectRow {
	out := []ProjectRow{}
	for _, p := range ps {
		out = append(out, ProjectRow{
			Code: p.Code, Name: p.Name,
			Start: yamlfile.FormatDate(p.Start), Until: yamlfile.FormatDate(p.Until),
			Expired: app.Expired(p, today),
		})
	}
	return out
}

func fromProjectRows(rows []ProjectRow) ([]settings.Project, error) {
	out := []settings.Project{}
	for _, r := range rows {
		p := settings.Project{Code: r.Code, Name: r.Name}
		var err error
		if p.Start, err = parseRowDate(r.Code, "開始日", r.Start); err != nil {
			return nil, err
		}
		if p.Until, err = parseRowDate(r.Code, "期限", r.Until); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// parseRowDate は、コードが code の行の yyyy-MM-dd を読む。空ならゼロ値にする。
// 行は番号でなくコードで指す。画面では期限切れの行を隠すので、番号が画面の並びと合わないため。
func parseRowDate(code, label, s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	d, err := time.Parse(yamlfile.DateLayout, s)
	if err != nil {
		return time.Time{}, &app.InputError{Message: fmt.Sprintf("コード %q の%s %q を読めません", code, label, s)}
	}
	return d, nil
}

// ProjectRows は、期間の外のものも含めたプロジェクトの一覧を返す。
func (s *Service) ProjectRows() ([]ProjectRow, error) {
	if s.startErr != nil {
		return nil, s.startErr
	}
	ps, err := s.app().AllProjects()
	if err != nil {
		return nil, err
	}
	return toProjectRows(ps, s.app().Today()), nil
}

// SaveProjects はプロジェクトの一覧を書く。base は設定画面で読んだときの一覧。
func (s *Service) SaveProjects(base, rows []ProjectRow) error {
	if s.startErr != nil {
		return s.startErr
	}
	b, err := fromProjectRows(base)
	if err != nil {
		return err
	}
	ps, err := fromProjectRows(rows)
	if err != nil {
		return err
	}
	return s.app().SaveProjects(b, ps)
}

// ChooseCSV は、取り込む CSV を選ぶダイアログを出す。やめたら空を返す。
func (s *Service) ChooseCSV() (string, error) {
	if s.env.ChooseCSV == nil {
		return "", errors.New("ファイルを選べません")
	}
	return s.env.ChooseCSV()
}

// ReadCSV は、取り込む CSV を UTF-8 として読み、行ごとの値を返す。
func (s *Service) ReadCSV(path string) ([][]string, error) {
	return app.ReadCSV(path)
}

// ProjectImport は、CSV からプロジェクトを取り込むときの指定。
// 列は 0 から数え、-1 ならその列を使わない。
type ProjectImport struct {
	SkipHeader  bool   `json:"skipHeader"`
	CodeColumn  int    `json:"codeColumn"`
	NameColumn  int    `json:"nameColumn"`
	StartColumn int    `json:"startColumn"`
	UntilColumn int    `json:"untilColumn"`
	DateFormat  string `json:"dateFormat"`
	Replace     bool   `json:"replace"`
}

// ImportProjects は、CSV の行を、編集中の一覧 current に取り込んだ一覧を返す。書くのは SaveProjects。
func (s *Service) ImportProjects(rows [][]string, opt ProjectImport, current []ProjectRow) ([]ProjectRow, error) {
	if s.startErr != nil {
		return nil, s.startErr
	}
	cur, err := fromProjectRows(current)
	if err != nil {
		return nil, err
	}
	ps, err := app.ImportProjects(rows, app.ProjectImport(opt), cur)
	if err != nil {
		return nil, err
	}
	out := toProjectRows(ps, s.app().Today())
	// CSV で変わらなかった行は、期限切れかを編集中の値のままにする。直している途中で期限を変えた行が、取り込みで隠れないようにするため。
	for i, r := range out {
		for _, c := range current {
			if c.Code == r.Code && c.Name == r.Name && c.Start == r.Start && c.Until == r.Until {
				out[i].Expired = c.Expired
				break
			}
		}
	}
	return out, nil
}
