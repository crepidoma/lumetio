package gui

import (
	"errors"
	"path/filepath"
	"slices"

	"github.com/crepidoma/lumetio/internal/output"
)

// OutputChoices は、右ペインで選べる出力セットと出力。
type OutputChoices struct {
	Sets    []string `json:"sets"`
	Outputs []string `json:"outputs"`
	// Default は最初に選ぶ名前。既定の出力がなければ一覧の先頭、何もなければ空。
	Default string `json:"default"`
	// Notices は、読めずに除いたテンプレートの知らせ。
	Notices []string `json:"notices"`
}

// OutputChoices は、右ペインで選べる出力セットと出力を返す。
func (s *Service) OutputChoices() (OutputChoices, error) {
	if s.startErr != nil {
		return OutputChoices{}, s.startErr
	}
	c, err := s.app().Choices()
	if err != nil {
		return OutputChoices{}, err
	}
	out := OutputChoices{Sets: nonNil(c.Sets), Outputs: nonNil(c.Outputs), Default: c.Default, Notices: nonNil(c.Skipped)}
	if out.Default == "" {
		if all := slices.Concat(out.Sets, out.Outputs); len(all) > 0 {
			out.Default = all[0]
		}
	}
	return out, nil
}

// PreviewOutput は、プレビューの出力 1 つ。
type PreviewOutput struct {
	Name     string `json:"name"`
	FileName string `json:"fileName"`
	Text     string `json:"text"`
	// Error は作れなかった理由。作れたら空。
	Error string `json:"error"`
}

// Preview は、出力または出力セットのプレビュー。
type Preview struct {
	Outputs []PreviewOutput `json:"outputs"`
	// Notices は、読めずに除いたテンプレートの知らせ。
	Notices []string `json:"notices"`
}

// Preview は、name の出力 (出力セットならそのすべて) を出力ごとに作る。作れない出力には理由を入れる。
func (s *Service) Preview(date, name string) (Preview, error) {
	d, err := s.date(date)
	if err != nil {
		return Preview{}, err
	}
	r, err := s.app().Preview(d, name)
	if err != nil {
		return Preview{}, err
	}
	out := Preview{Outputs: []PreviewOutput{}, Notices: nonNil(r.Skipped)}
	for _, o := range r.Outputs {
		out.Outputs = append(out.Outputs, PreviewOutput{Name: o.Name, FileName: o.FileName, Text: o.Text, Error: o.Err})
	}
	return out, nil
}

// ExportResult は出力した結果。
type ExportResult struct {
	// Written は書いたファイルの名前。
	Written []string `json:"written"`
	// Exists は、出力先にすでにあったファイルの名前。あれば何も書いていないので、扱いを選んで呼び直す。
	Exists []string `json:"exists"`
	// OpenErrors は、書いたファイルを標準のアプリで開けなかった理由。
	OpenErrors []string `json:"openErrors"`
}

// Export は、出力または出力セットを出力先に書き、書いたファイルを標準のアプリで開く。
// onConflict は、出力先に同じ名前のファイルがあるときの扱い。ask (書かずに知らせる)、overwrite、rename のどれか。
func (s *Service) Export(date, name, onConflict string) (ExportResult, error) {
	d, err := s.date(date)
	if err != nil {
		return ExportResult{}, err
	}
	// 画面は ask か overwrite か rename を送る。知らない名前は、上書きしないよう確かめる扱いにする。
	conflict, _ := output.ParseConflict(onConflict)
	out := ExportResult{Written: []string{}, Exists: []string{}, OpenErrors: []string{}}
	written, _, err := s.app().Export(d, name, conflict)
	var exists *output.ExistsError
	if errors.As(err, &exists) {
		out.Exists = exists.Files
		return out, nil
	}
	if err != nil {
		return ExportResult{}, err
	}
	out.Written = written
	// 書いた結果をすぐ確かめ、貼り付けや取り込みに使えるよう、それぞれ標準のアプリで開く。
	for _, f := range written {
		if err := s.env.Open(filepath.Join(s.app().OutputDir(), f)); err != nil {
			out.OpenErrors = append(out.OpenErrors, err.Error())
		}
	}
	return out, nil
}
