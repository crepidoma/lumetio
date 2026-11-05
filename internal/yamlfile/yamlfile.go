// Package yamlfile は、データフォルダの YAML ファイルに共通する読み書きの作法をまとめる。
package yamlfile

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
	"golang.org/x/text/unicode/norm"
)

// DateLayout は YAML に書く日付の形。画面や MCP とのやり取りも、この形にそろえる。
const DateLayout = "2006-01-02"

// DateOnly は、時刻を捨てて UTC の 0 時の日付にする。
// 日付どうしを比べるときに、時刻やタイムゾーンの違いを持ち込まないため。
func DateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// FormatDate は日付を DateLayout で書く。ゼロ値 (日付がない) なら空にする。
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(DateLayout)
}

// Decode は YAML を v に読む。
// 知らないキーはエラーにする。
// 読み飛ばすと、保存し直したときに消えてしまうため。
func Decode(data []byte, v any) error {
	// macOS と Windows で濁点などの表現が違っても、同じ文字列として扱う。
	data = norm.NFC.Bytes(data)

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	// 空のファイルは、何も書かれていない中身として読む。
	if err := dec.Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// Encode は root を最上位とする YAML を書く。
func Encode(root *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Mapping は空のマッピングを作る。
// キーは足した順に書かれる。
func Mapping() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode}
}

// Add はマッピングにキーと値を足す。
func Add(m *yaml.Node, key string, value *yaml.Node) {
	m.Content = append(m.Content, Text(key), value)
}

// Sequence は空の並びを作る。
// 要素は改行とインデントで書かれる。
func Sequence() *yaml.Node {
	return &yaml.Node{Kind: yaml.SequenceNode}
}

// Text は文字列のノードを作る。
// 複数行は手で直しやすいようにブロック (|) で書く。
func Text(s string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
	if strings.Contains(s, "\n") {
		n.Style = yaml.LiteralStyle
	}
	return n
}

// Bool は真偽値のノードを作る。
func Bool(b bool) *yaml.Node {
	v := "false"
	if b {
		v = "true"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: v}
}

// Int は整数のノードを作る。
func Int(i int) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(i)}
}

// Date は日付のノードを作る。
// 引用符で囲まずに書く。
func Date(t time.Time) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!timestamp", Value: t.Format(DateLayout)}
}

// FlowList は文字列の並びを 1 行 ([a, b]) で書くノードを作る。
func FlowList(items []string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
	for _, s := range items {
		n.Content = append(n.Content, Text(s))
	}
	return n
}
