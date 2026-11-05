package output

import (
	"reflect"
	"testing"
)

func TestKindOf(t *testing.T) {
	tests := []struct {
		file string
		name string
		kind Kind
		ok   bool
	}{
		{"日報テキスト.tmpl", "日報テキスト", KindText, true},
		{"工数CSV.csv.yaml", "工数CSV", KindCSV, true},
		{"日報.set.yaml", "日報", KindSet, true},
		{"メモ.yaml", "", "", false},
		{".tmpl", "", "", false},
	}
	for _, tt := range tests {
		name, kind, ok := KindOf(tt.file)
		if name != tt.name || kind != tt.kind || ok != tt.ok {
			t.Errorf("KindOf(%q) = %q, %q, %v", tt.file, name, kind, ok)
		}
		if ok && FileName(name, kind) != tt.file {
			t.Errorf("FileName(%q, %q) = %q", name, kind, FileName(name, kind))
		}
	}
}

func TestCheckName(t *testing.T) {
	if err := CheckName("日報 2"); err != nil {
		t.Errorf("使える名前: %v", err)
	}
	for _, bad := range []string{"", "a/b", `a\b`, "a:b", "a?"} {
		if CheckName(bad) == nil {
			t.Errorf("%q を使える名前とした", bad)
		}
	}
}

// 画面の形にして書き直しても、同じ出力として読める。省略時の値と同じ項目は書かない。
func TestTextRoundTrip(t *testing.T) {
	src := "---\nfile: \"'report_'yyyyMMdd\"\nextension: txt\nnewline: crlf\n---\n{{date \"yyyy/MM/dd\"}} の日報\n"
	s, err := DecodeText([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := TextSpec{File: "'report_'yyyyMMdd", Extension: "txt", Encoding: UTF8, Newline: CRLF, Body: "{{date \"yyyy/MM/dd\"}} の日報\n"}
	if s != want {
		t.Errorf("DecodeText = %+v", s)
	}
	data, err := MarshalText(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != src {
		t.Errorf("MarshalText =\n%s\nwant\n%s", data, src)
	}
	if _, err := ParseText("日報", data); err != nil {
		t.Errorf("書いたものを読めない: %v", err)
	}
}

func TestCSVRoundTrip(t *testing.T) {
	src := `kind: csv
file: "'report_'yyyyMMdd"
header: false
rows: projects
skip_rows_without_counted: true
columns:
  - header: code
    value: '{{.code}}'
  - header: hours
    value: '{{duration .minutes "H"}}'
`
	s, err := DecodeCSV([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := CSVSpec{
		File: "'report_'yyyyMMdd", Encoding: UTF8, Newline: CRLF,
		Header: false, Rows: RowsProjects, SkipRowsWithoutCounted: true,
		Columns: []CSVColumn{{"code", "{{.code}}"}, {"hours", `{{duration .minutes "H"}}`}},
	}
	if !reflect.DeepEqual(s, want) {
		t.Errorf("DecodeCSV = %+v", s)
	}
	data, err := MarshalCSV(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != src {
		t.Errorf("MarshalCSV =\n%s\nwant\n%s", data, src)
	}
	if _, err := ParseCSV("工数", data); err != nil {
		t.Errorf("書いたものを読めない: %v", err)
	}
}

func TestSetRoundTrip(t *testing.T) {
	data, err := MarshalSet([]string{"日報テキスト", "工数CSV"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "kind: set\noutputs: [日報テキスト, 工数CSV]\n"; string(data) != want {
		t.Errorf("MarshalSet =\n%s", data)
	}
	got, err := DecodeSet(data)
	if err != nil || !reflect.DeepEqual(got, []string{"日報テキスト", "工数CSV"}) {
		t.Errorf("DecodeSet = %v, %v", got, err)
	}
}
