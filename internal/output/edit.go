package output

import (
	"fmt"
	"strings"

	"github.com/crepidoma/lumetio/internal/yamlfile"
	"go.yaml.in/yaml/v3"
	"golang.org/x/text/unicode/norm"
)

// Kind はテンプレートのファイルの種類。
type Kind string

const (
	KindText Kind = "text"
	KindCSV  Kind = "csv"
	KindSet  Kind = "set"
)

// suffixes はファイル名の種類と拡張子。
var suffixes = []struct {
	suffix string
	kind   Kind
}{
	{".tmpl", KindText},
	{".csv.yaml", KindCSV},
	{".set.yaml", KindSet},
}

// KindOf は、ファイル名から名前と種類を返す。名前の形に合わなければ ok は false。
// 名前は NFC にそろえる。macOS と Windows で濁点などの表現が違っても、同じ名前として扱うため。
func KindOf(fileName string) (name string, kind Kind, ok bool) {
	fileName = norm.NFC.String(fileName)
	for _, s := range suffixes {
		if name, ok := strings.CutSuffix(fileName, s.suffix); ok && name != "" {
			return name, s.kind, true
		}
	}
	return "", "", false
}

// FileName は、名前と種類からファイル名を作る。
func FileName(name string, kind Kind) string {
	for _, s := range suffixes {
		if s.kind == kind {
			return name + s.suffix
		}
	}
	return name
}

// CheckName は、出力や出力セットの名前として使えるかを確かめる。
// 名前はそのままファイル名になるので、フォルダの区切りや OS が受け付けない文字を入れさせない。
func CheckName(name string) error {
	if name == "" {
		return fmt.Errorf("名前が空です")
	}
	if i := strings.IndexAny(name, `/\:*?"<>|`); i >= 0 {
		return fmt.Errorf("名前に %q は使えません", name[i:i+1])
	}
	return nil
}

// TextSpec は、テキストの出力 (*.tmpl) を画面で直すときの中身。
type TextSpec struct {
	File      string
	Extension string
	Encoding  Encoding
	Newline   Newline
	Body      string
}

// CSVSpec は、CSV の出力 (*.csv.yaml) を画面で直すときの中身。拡張子は csv に決まっている。
type CSVSpec struct {
	File                   string
	Encoding               Encoding
	Newline                Newline
	Header                 bool
	Rows                   Rows
	SkipRowsWithoutCounted bool
	Columns                []CSVColumn
}

// CSVColumn は CSV の列 1 つ。
type CSVColumn struct {
	Header string
	Value  string
}

// DecodeText は、テキストの出力の中身を画面で直す形にする。省略した項目は省略時の値で埋める。
func DecodeText(data []byte) (TextSpec, error) {
	front, body, _, ok := splitFront(normalize(data))
	if !ok {
		return TextSpec{}, fmt.Errorf("先頭に --- で囲んだ設定がありません")
	}
	var raw rawFront
	if err := yamlfile.Decode([]byte(front), &raw); err != nil {
		return TextSpec{}, fmt.Errorf("設定を読めません: %w", err)
	}
	return TextSpec{
		File:      raw.File,
		Extension: raw.Extension,
		Encoding:  orDefault(Encoding(raw.Encoding), UTF8),
		Newline:   orDefault(Newline(raw.Newline), LF),
		Body:      body,
	}, nil
}

// MarshalText は、テキストの出力の中身を書く。省略時の値と同じ項目は書かない。
func MarshalText(s TextSpec) ([]byte, error) {
	front := yamlfile.Mapping()
	yamlfile.Add(front, "file", text(s.File))
	yamlfile.Add(front, "extension", yamlfile.Text(s.Extension))
	if s.Encoding != "" && s.Encoding != UTF8 {
		yamlfile.Add(front, "encoding", yamlfile.Text(string(s.Encoding)))
	}
	if s.Newline != "" && s.Newline != LF {
		yamlfile.Add(front, "newline", yamlfile.Text(string(s.Newline)))
	}
	data, err := yamlfile.Encode(front)
	if err != nil {
		return nil, err
	}
	return []byte("---\n" + string(data) + "---\n" + s.Body), nil
}

// DecodeCSV は、CSV の出力の中身を画面で直す形にする。省略した項目は省略時の値で埋める。
func DecodeCSV(data []byte) (CSVSpec, error) {
	var raw rawCSV
	if err := yamlfile.Decode(data, &raw); err != nil {
		return CSVSpec{}, fmt.Errorf("読めません: %w", err)
	}
	s := CSVSpec{
		File:                   raw.File,
		Encoding:               orDefault(Encoding(raw.Encoding), UTF8),
		Newline:                orDefault(Newline(raw.Newline), CRLF),
		Header:                 raw.Header == nil || *raw.Header,
		Rows:                   Rows(raw.Rows),
		SkipRowsWithoutCounted: raw.SkipRowsWithoutCounted,
		Columns:                []CSVColumn{},
	}
	for _, c := range raw.Columns {
		s.Columns = append(s.Columns, CSVColumn{Header: c.Header, Value: c.Value})
	}
	return s, nil
}

// MarshalCSV は、CSV の出力の中身を書く。省略時の値と同じ項目は書かない。
func MarshalCSV(s CSVSpec) ([]byte, error) {
	root := yamlfile.Mapping()
	yamlfile.Add(root, "kind", yamlfile.Text("csv"))
	yamlfile.Add(root, "file", text(s.File))
	if s.Encoding != "" && s.Encoding != UTF8 {
		yamlfile.Add(root, "encoding", yamlfile.Text(string(s.Encoding)))
	}
	if s.Newline != "" && s.Newline != CRLF {
		yamlfile.Add(root, "newline", yamlfile.Text(string(s.Newline)))
	}
	if !s.Header {
		yamlfile.Add(root, "header", yamlfile.Bool(false))
	}
	yamlfile.Add(root, "rows", yamlfile.Text(string(s.Rows)))
	if s.SkipRowsWithoutCounted {
		yamlfile.Add(root, "skip_rows_without_counted", yamlfile.Bool(true))
	}
	columns := yamlfile.Sequence()
	for _, c := range s.Columns {
		n := yamlfile.Mapping()
		yamlfile.Add(n, "header", text(c.Header))
		yamlfile.Add(n, "value", text(c.Value))
		columns.Content = append(columns.Content, n)
	}
	yamlfile.Add(root, "columns", columns)
	return yamlfile.Encode(root)
}

// DecodeSet は、出力セットに入っている出力の名前を返す。
func DecodeSet(data []byte) ([]string, error) {
	var raw rawSet
	if err := yamlfile.Decode(data, &raw); err != nil {
		return nil, fmt.Errorf("読めません: %w", err)
	}
	if raw.Outputs == nil {
		return []string{}, nil
	}
	return raw.Outputs, nil
}

// MarshalSet は、出力セットの中身を書く。
func MarshalSet(outputs []string) ([]byte, error) {
	root := yamlfile.Mapping()
	yamlfile.Add(root, "kind", yamlfile.Text("set"))
	yamlfile.Add(root, "outputs", yamlfile.FlowList(outputs))
	return yamlfile.Encode(root)
}

func orDefault[T ~string](v, def T) T {
	if v == "" {
		return def
	}
	return v
}

// text は文字列のノードを作る。' を含む値は "…" で囲む。
// 書式の '…' を YAML の '…' で囲むと ” が並んで読みにくいため。
func text(s string) *yaml.Node {
	n := yamlfile.Text(s)
	if n.Style == 0 && strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		n.Style = yaml.DoubleQuotedStyle
	}
	return n
}
