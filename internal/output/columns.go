package output

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/crepidoma/lumetio/internal/datefmt"
	"github.com/crepidoma/lumetio/internal/durfmt"
)

// FormatKind は、列の値に付ける書式の種類。
type FormatKind string

const (
	FormatNone     FormatKind = ""
	FormatDate     FormatKind = "date"
	FormatDuration FormatKind = "duration"
)

// ColumnField は、CSV の列の値として、テンプレートを書かずに一覧から選べるもの。
type ColumnField struct {
	Key   string
	Label string
	// Format は、選ぶときに書式も決めるか。日付と時間だけが書式を持つ。
	Format FormatKind
	// DefaultFormat は、選んだときに入れておく書式。
	DefaultFormat string
	// templates は、行の単位ごとに書くテンプレート。%s は書式 (引用符付き) に置き換える。その単位で使えなければない。
	templates map[Rows]string
}

// columnFields は、一覧に出す値。並びが一覧の順。
var columnFields = []ColumnField{
	{Key: "date", Label: "日付", Format: FormatDate, DefaultFormat: "yyyy/MM/dd",
		templates: map[Rows]string{RowsProjects: "{{date %s}}", RowsTasks: "{{date %s}}"}},
	{Key: "code", Label: "コード",
		templates: map[Rows]string{RowsProjects: "{{.code}}", RowsTasks: "{{.project.code}}"}},
	{Key: "project", Label: "プロジェクト名",
		templates: map[Rows]string{RowsProjects: "{{.name}}", RowsTasks: "{{.project.name}}"}},
	{Key: "title", Label: "タスク名",
		templates: map[Rows]string{RowsTasks: "{{.title}}"}},
	{Key: "time", Label: "時間", Format: FormatDuration, DefaultFormat: "H:mm",
		templates: map[Rows]string{RowsProjects: "{{duration .minutes %s}}", RowsTasks: "{{duration .minutes %s}}"}},
	{Key: "minutes", Label: "分",
		templates: map[Rows]string{RowsProjects: "{{.minutes}}", RowsTasks: "{{.minutes}}"}},
	{Key: "tags", Label: "タグ",
		templates: map[Rows]string{RowsTasks: `{{join .tags " "}}`}},
	{Key: "note", Label: "メモ",
		templates: map[Rows]string{RowsTasks: "{{.note}}"}},
}

// ColumnFields は、行の単位で使える値を、一覧の順に返す。
func ColumnFields(rows Rows) []ColumnField {
	var out []ColumnField
	for _, f := range columnFields {
		if _, ok := f.templates[rows]; ok {
			out = append(out, f)
		}
	}
	return out
}

// ColumnValue は、一覧から選んだ値と書式から、列の値のテンプレートを作る。
// 書式が読めないとき、その行の単位で使えない値のときは誤りにする。
func ColumnValue(rows Rows, key, format string) (string, error) {
	i := slices.IndexFunc(columnFields, func(f ColumnField) bool { return f.Key == key })
	if i < 0 {
		return "", fmt.Errorf("値 %q はありません", key)
	}
	f := columnFields[i]
	tmpl, ok := f.templates[rows]
	if !ok {
		return "", fmt.Errorf("%s は、行の単位が%sのときは使えません", f.Label, rowsLabel(rows))
	}
	switch f.Format {
	case FormatDate:
		if _, err := datefmt.Parse(format); err != nil {
			return "", fmt.Errorf("%sの書式: %w", f.Label, err)
		}
	case FormatDuration:
		if _, err := durfmt.Parse(format); err != nil {
			return "", fmt.Errorf("%sの書式: %w", f.Label, err)
		}
	default:
		return tmpl, nil
	}
	return fmt.Sprintf(tmpl, strconv.Quote(format)), nil
}

func rowsLabel(rows Rows) string {
	if rows == RowsTasks {
		return "記録"
	}
	return "プロジェクト"
}

// quotedPattern は、テンプレートの中の "…" の文字列。
const quotedPattern = `("(?:[^"\\]|\\.)*")`

// DescribeColumn は、列の値のテンプレートが、一覧のどの値と同じ形かを返す。
// どれとも違えば key は空。画面で、選んだ値として出すか、そのまま書いたテンプレートとして出すかを決めるため。
func DescribeColumn(rows Rows, value string) (key, format string) {
	for _, f := range columnFields {
		tmpl, ok := f.templates[rows]
		if !ok {
			continue
		}
		if f.Format == FormatNone {
			if value == tmpl {
				return f.Key, ""
			}
			continue
		}
		before, after, _ := strings.Cut(tmpl, "%s")
		re := regexp.MustCompile("^" + regexp.QuoteMeta(before) + quotedPattern + regexp.QuoteMeta(after) + "$")
		m := re.FindStringSubmatch(value)
		if m == nil {
			continue
		}
		if s, err := strconv.Unquote(m[1]); err == nil {
			return f.Key, s
		}
	}
	return "", ""
}
