package output

import (
	"fmt"

	"golang.org/x/text/unicode/norm"
)

// File は templates/ の中のファイル 1 つ。
type File struct {
	// Name はフォルダを含まないファイル名。
	Name string
	Data []byte
}

// Library は templates/ から読んだ出力と出力セット。
type Library struct {
	// Outputs と Sets は、ファイル名の順に並ぶ。
	Outputs []Output
	Sets    []Set
}

// Load は templates/ の中のファイルを読む。
// 読めないファイルは除いて、skipped に理由を返す。
// 1 つのファイルの誤りで、ほかの出力まで使えなくならないようにするため。
// 名前の形に合わないファイルは、何も言わずに無視する。
func Load(files []File) (lib Library, skipped []error) {
	type named struct {
		file File
		name string
		kind Kind
	}
	var list []named
	count := map[string]int{}
	for _, f := range files {
		// 出力セットの outputs は NFC で読むので、ファイル名もそろえる (KindOf がそろえる)。
		if name, kind, ok := KindOf(f.Name); ok {
			list = append(list, named{f, name, kind})
			count[name]++
		}
	}

	for _, n := range list {
		// 名前が重なると、出力セットがどれを指すのか決まらない。
		if count[n.name] > 1 {
			skipped = append(skipped, fmt.Errorf("%s: 名前 %q のファイルがほかにもあります", n.file.Name, n.name))
			continue
		}
		var err error
		if n.kind == KindSet {
			var s Set
			if s, err = ParseSet(n.name, n.file.Data); err == nil {
				lib.Sets = append(lib.Sets, s)
			}
		} else {
			var o Output
			if o, err = Parse(n.kind, n.name, n.file.Data); err == nil {
				lib.Outputs = append(lib.Outputs, o)
			}
		}
		if err != nil {
			skipped = append(skipped, fmt.Errorf("%s: %w", n.file.Name, err))
		}
	}
	return lib, skipped
}

// Output は名前で出力を探す。
// 名前は NFC にそろえて比べる。
func (l Library) Output(name string) (Output, bool) {
	name = norm.NFC.String(name)
	for _, o := range l.Outputs {
		if o.Name == name {
			return o, true
		}
	}
	return Output{}, false
}

// SetOutputs は、出力セットに入っている出力を並びの順に返す。
// 名前は NFC にそろえて比べる。
// 存在しない出力や読めなかった出力を指していたら、エラーにする。
func (l Library) SetOutputs(name string) ([]Output, error) {
	name = norm.NFC.String(name)
	var set *Set
	for i := range l.Sets {
		if l.Sets[i].Name == name {
			set = &l.Sets[i]
			break
		}
	}
	if set == nil {
		return nil, fmt.Errorf("出力セット %q がないか、読めません", name)
	}
	outputs := make([]Output, 0, len(set.Outputs))
	for _, n := range set.Outputs {
		o, ok := l.Output(n)
		if !ok {
			return nil, fmt.Errorf("出力セット %q の出力 %q がないか、読めません", name, n)
		}
		outputs = append(outputs, o)
	}
	return outputs, nil
}
