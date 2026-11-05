package output

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/crepidoma/lumetio/internal/datefmt"
	"github.com/crepidoma/lumetio/internal/durfmt"
)

// dayFuncs は、出力する日によって中身が変わる関数を持つ。
type dayFuncs struct {
	date time.Time
}

// funcs はテンプレートで使える関数。
// Go の組み込みの関数で書けるものは足さない。
func funcs(d dayFuncs) template.FuncMap {
	return template.FuncMap{
		"date":     d.formatDate,
		"duration": duration,
		"where":    where,
		"has":      has,
		"join":     strings.Join,
	}
}

func (d dayFuncs) formatDate(layout string) (string, error) {
	f, err := datefmt.Parse(layout)
	if err != nil {
		return "", err
	}
	return f.Format(d.date), nil
}

func duration(minutes int, layout string) (string, error) {
	f, err := durfmt.Parse(layout)
	if err != nil {
		return "", err
	}
	return f.Format(minutes), nil
}

// where は、一覧から項目が値に合う要素を残す。
// 項目の値が一覧なら値を含むもの、それ以外は値が等しいものを残す。
// 項目は project.code のように . でたどれる。
func where(list []map[string]any, key string, value any) ([]map[string]any, error) {
	result := []map[string]any{}
	for _, m := range list {
		field, err := lookup(m, key)
		if err != nil {
			return nil, err
		}
		if tags, ok := field.([]string); ok {
			if s, ok := value.(string); ok && slices.Contains(tags, s) {
				result = append(result, m)
			}
			continue
		}
		if reflect.DeepEqual(field, value) {
			result = append(result, m)
		}
	}
	return result, nil
}

func lookup(m map[string]any, key string) (any, error) {
	var v any = m
	for name := range strings.SplitSeq(key, ".") {
		mm, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%q をたどれません", key)
		}
		if v, ok = mm[name]; !ok {
			return nil, fmt.Errorf("%q という項目はありません", key)
		}
	}
	return v, nil
}

func has(list []string, value string) bool {
	return slices.Contains(list, value)
}
