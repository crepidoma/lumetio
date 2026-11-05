// Package output は出力と出力セットの定義を読み、その日の記録から出力を作る。
package output

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/crepidoma/lumetio/internal/datefmt"
	"github.com/crepidoma/lumetio/internal/yamlfile"
	"golang.org/x/text/unicode/norm"
)

// Output は出力 1 つ分の定義。
// テキストの出力か CSV の出力のどちらか。
type Output struct {
	Name      string
	File      datefmt.Format
	Extension string
	Encoding  Encoding
	Newline   Newline

	// text はテキストの出力の本文。CSV の出力なら nil。
	text *template.Template
	// csv は CSV の出力の定義。テキストの出力なら nil。
	csv *csvSpec
}

// IsCSV は CSV の出力か。
func (o Output) IsCSV() bool {
	return o.csv != nil
}

// Encoding は出力の文字コード。
type Encoding string

const (
	UTF8    Encoding = "utf-8"
	UTF8BOM Encoding = "utf-8-bom"
)

// bom は UTF-8 の BOM。
const bom = "\xef\xbb\xbf"

// Newline は出力の改行。
type Newline string

const (
	LF   Newline = "lf"
	CRLF Newline = "crlf"
)

// Rows は CSV の 1 行が何にあたるか。
type Rows string

const (
	RowsProjects Rows = "projects"
	RowsTasks    Rows = "tasks"
)

type csvSpec struct {
	header                 bool
	rows                   Rows
	skipRowsWithoutCounted bool
	columns                []csvColumn
}

type csvColumn struct {
	header string
	value  *template.Template
}

// Set は出力セット。
type Set struct {
	Name string
	// Outputs は出力の名前の並び。並びが画面のタブの順。
	Outputs []string
}

type rawFront struct {
	File      string `yaml:"file"`
	Extension string `yaml:"extension"`
	Encoding  string `yaml:"encoding"`
	Newline   string `yaml:"newline"`
}

type rawCSV struct {
	Kind                   string         `yaml:"kind"`
	File                   string         `yaml:"file"`
	Encoding               string         `yaml:"encoding"`
	Newline                string         `yaml:"newline"`
	Header                 *bool          `yaml:"header"`
	Rows                   string         `yaml:"rows"`
	SkipRowsWithoutCounted bool           `yaml:"skip_rows_without_counted"`
	Columns                []rawCSVColumn `yaml:"columns"`
}

type rawCSVColumn struct {
	Header string `yaml:"header"`
	Value  string `yaml:"value"`
}

type rawSet struct {
	Kind    string   `yaml:"kind"`
	Outputs []string `yaml:"outputs"`
}

// Parse は、テキストか CSV の出力の中身を、種類に合わせて読む。
// 出力セットは出力ではないので、ParseSet で読む。
func Parse(kind Kind, name string, data []byte) (Output, error) {
	switch kind {
	case KindText:
		return ParseText(name, data)
	case KindCSV:
		return ParseCSV(name, data)
	}
	return Output{}, fmt.Errorf("種類 %q は出力ではありません", kind)
}

// ParseText はテキストの出力 (*.tmpl) の中身を読む。
// name は、ファイル名から種類と拡張子を除いた名前。
func ParseText(name string, data []byte) (Output, error) {
	src := normalize(data)
	front, body, frontLines, ok := splitFront(src)
	if !ok {
		return Output{}, fmt.Errorf("先頭に --- で囲んだ設定がありません")
	}
	var raw rawFront
	if err := yamlfile.Decode([]byte(front), &raw); err != nil {
		return Output{}, fmt.Errorf("設定を読めません: %w", err)
	}
	o, err := common(name, raw.File, raw.Extension, raw.Encoding, raw.Newline, LF)
	if err != nil {
		return Output{}, err
	}

	// 設定の行の分だけ改行を含むコメントを前に置き、エラーに出る行番号をファイルの行に合わせる。
	// コメントは何も出さない。
	body = "{{/*" + strings.Repeat("\n", frontLines) + "*/}}" + body
	o.text, err = parseTemplate(name, body)
	if err != nil {
		return Output{}, err
	}
	return o, nil
}

// ParseCSV は CSV の出力 (*.csv.yaml) の中身を読む。
func ParseCSV(name string, data []byte) (Output, error) {
	var raw rawCSV
	if err := yamlfile.Decode(data, &raw); err != nil {
		return Output{}, fmt.Errorf("読めません: %w", err)
	}
	if err := checkKind(raw.Kind, "csv"); err != nil {
		return Output{}, err
	}
	// 拡張子は csv に決まっているので、ファイルには書かない。
	o, err := common(name, raw.File, "csv", raw.Encoding, raw.Newline, CRLF)
	if err != nil {
		return Output{}, err
	}

	spec := &csvSpec{header: true, skipRowsWithoutCounted: raw.SkipRowsWithoutCounted}
	if raw.Header != nil {
		spec.header = *raw.Header
	}
	switch Rows(raw.Rows) {
	case RowsProjects, RowsTasks:
		spec.rows = Rows(raw.Rows)
	case "":
		return Output{}, fmt.Errorf("rows がありません。projects か tasks を書きます")
	default:
		return Output{}, fmt.Errorf("rows の %q は使えません。projects か tasks を書きます", raw.Rows)
	}
	if len(raw.Columns) == 0 {
		return Output{}, fmt.Errorf("columns に列がありません")
	}
	for i, rc := range raw.Columns {
		if rc.Value == "" {
			return Output{}, fmt.Errorf("%d 列目に value がありません", i+1)
		}
		value, err := parseTemplate(fmt.Sprintf("%d 列目", i+1), normalize([]byte(rc.Value)))
		if err != nil {
			return Output{}, err
		}
		spec.columns = append(spec.columns, csvColumn{header: rc.Header, value: value})
	}
	o.csv = spec
	return o, nil
}

// ParseSet は出力セット (*.set.yaml) の中身を読む。
func ParseSet(name string, data []byte) (Set, error) {
	var raw rawSet
	if err := yamlfile.Decode(data, &raw); err != nil {
		return Set{}, fmt.Errorf("読めません: %w", err)
	}
	if err := checkKind(raw.Kind, "set"); err != nil {
		return Set{}, err
	}
	if len(raw.Outputs) == 0 {
		return Set{}, fmt.Errorf("outputs に出力がありません")
	}
	for i, o := range raw.Outputs {
		if strings.TrimSpace(o) == "" {
			return Set{}, fmt.Errorf("outputs の %d 件目が空です", i+1)
		}
		// 同じ出力が 2 つあると、同じファイルを 2 回書くことになる。
		if slices.Contains(raw.Outputs[:i], o) {
			return Set{}, fmt.Errorf("outputs に %q が重なっています", o)
		}
	}
	return Set{Name: name, Outputs: raw.Outputs}, nil
}

// checkKind は kind がファイル名の種類と合っているか確かめる。
// 種類を後から増やす余地を残すため、kind を必須にしている。
func checkKind(kind, want string) error {
	if kind == "" {
		return fmt.Errorf("kind がありません。%s を書きます", want)
	}
	if kind != want {
		return fmt.Errorf("kind の %q がファイル名の種類と違います。%s を書きます", kind, want)
	}
	return nil
}

// common はテキストと CSV に共通する出力の設定を読む。
func common(name, file, extension, encoding, newline string, defaultNewline Newline) (Output, error) {
	o := Output{Name: name, Extension: extension, Encoding: UTF8, Newline: defaultNewline}

	if file == "" {
		return Output{}, fmt.Errorf("file がありません")
	}
	f, err := datefmt.Parse(file)
	if err != nil {
		return Output{}, fmt.Errorf("file: %w", err)
	}
	o.File = f

	if extension == "" {
		return Output{}, fmt.Errorf("extension がありません")
	}
	if strings.HasPrefix(extension, ".") {
		return Output{}, fmt.Errorf("extension は . を付けずに書きます")
	}
	// file と extension はそのままファイル名になるので、フォルダの区切りや OS が受け付けない文字を入れさせない。
	for _, s := range []string{file, extension} {
		if i := strings.IndexAny(s, `/\:*?"<>|`); i >= 0 {
			return Output{}, fmt.Errorf("ファイル名に %q は使えません", s[i:i+1])
		}
	}

	switch Encoding(encoding) {
	case "":
	case UTF8, UTF8BOM:
		o.Encoding = Encoding(encoding)
	default:
		return Output{}, fmt.Errorf("encoding の %q は使えません。utf-8 か utf-8-bom を書きます", encoding)
	}
	switch Newline(newline) {
	case "":
	case LF, CRLF:
		o.Newline = Newline(newline)
	default:
		return Output{}, fmt.Errorf("newline の %q は使えません。lf か crlf を書きます", newline)
	}
	return o, nil
}

// parseTemplate はテンプレートを解釈する。
// 関数の中身は出力のたびに差し替えるので、ここでは名前だけを知らせる。
func parseTemplate(name, src string) (*template.Template, error) {
	t, err := template.New(name).Option("missingkey=error").Funcs(funcs(dayFuncs{})).Parse(src)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// normalize はファイルの中身を、改行が LF の NFC の文字列にする。
// macOS と Windows で濁点などの表現や改行が違っても、同じテンプレートとして扱うため。
func normalize(data []byte) string {
	data = bytes.TrimPrefix(data, []byte(bom))
	s := norm.NFC.String(string(data))
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// splitFront は、先頭の --- で囲んだ設定と本文を分ける。
// lines は設定が占める行数 (--- の 2 行を含む)。
func splitFront(src string) (front, body string, lines int, ok bool) {
	rest, ok := strings.CutPrefix(src, "---\n")
	if !ok {
		return "", "", 0, false
	}
	if rest == "---" {
		return "", "", 2, true
	}
	if after, ok := strings.CutPrefix(rest, "---\n"); ok {
		return "", after, 2, true
	}
	front, body, ok = strings.Cut(rest, "\n---\n")
	if !ok {
		// 本文がなく、閉じる --- でファイルが終わる。
		front, ok = strings.CutSuffix(rest, "\n---")
		if !ok {
			return "", "", 0, false
		}
	}
	return front, body, strings.Count(front, "\n") + 3, true
}
