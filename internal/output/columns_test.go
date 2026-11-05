package output

import (
	"strings"
	"testing"
)

// 一覧から選んだ値で作ったテンプレートは、同じ値として読み戻せる。
func TestColumnValueRoundTrip(t *testing.T) {
	tests := []struct {
		rows   Rows
		key    string
		format string
		want   string
	}{
		{RowsProjects, "code", "", "{{.code}}"},
		{RowsTasks, "code", "", "{{.project.code}}"},
		{RowsTasks, "project", "", "{{.project.name}}"},
		{RowsTasks, "title", "", "{{.title}}"},
		{RowsProjects, "date", "yyyy/MM/dd", `{{date "yyyy/MM/dd"}}`},
		{RowsTasks, "time", "H'h'", `{{duration .minutes "H'h'"}}`},
		{RowsTasks, "tags", "", `{{join .tags " "}}`},
	}
	for _, tt := range tests {
		got, err := ColumnValue(tt.rows, tt.key, tt.format)
		if err != nil || got != tt.want {
			t.Errorf("ColumnValue(%s, %s, %q) = %q, %v, want %q", tt.rows, tt.key, tt.format, got, err, tt.want)
			continue
		}
		if key, format := DescribeColumn(tt.rows, got); key != tt.key || format != tt.format {
			t.Errorf("DescribeColumn(%s, %q) = %q, %q", tt.rows, got, key, format)
		}
	}
}

// 一覧の形と違うテンプレートは、選んだ値として読まない。
func TestDescribeColumnCustom(t *testing.T) {
	for _, v := range []string{"{{.code}}-{{.name}}", `{{date "yyyy" }}`, "{{.title}}", "固定の値", ""} {
		if key, _ := DescribeColumn(RowsProjects, v); key != "" {
			t.Errorf("DescribeColumn(%q) = %q", v, key)
		}
	}
}

func TestColumnValueErrors(t *testing.T) {
	for _, tt := range []struct {
		rows        Rows
		key, format string
	}{
		{RowsProjects, "title", ""},
		{RowsTasks, "ない", ""},
		{RowsTasks, "date", "yyyy-MM-dd HH"},
		{RowsTasks, "time", "x"},
	} {
		if _, err := ColumnValue(tt.rows, tt.key, tt.format); err == nil {
			t.Errorf("ColumnValue(%s, %s, %q) succeeded", tt.rows, tt.key, tt.format)
		}
	}
}

func TestColumnFields(t *testing.T) {
	var keys []string
	for _, f := range ColumnFields(RowsProjects) {
		keys = append(keys, f.Key)
	}
	if got := strings.Join(keys, " "); got != "date code project time minutes" {
		t.Errorf("ColumnFields(projects) = %s", got)
	}
}
