package output

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// ErrNotUTF8 は、出力先のファイルが UTF-8 として読めないこと。
// 出力した後に、手で別の文字コードで保存し直したときに起きる。
var ErrNotUTF8 = errors.New("UTF-8 として読めません")

// ReadFile は、出力先 dir に書いたファイル name の中身を読む。
// ファイルがなければ exists は false で、誤りにしない。
// 中身は先頭の BOM を除いて NFC にそろえ、改行は変えない。手で書き足した中身をなるべくそのまま返すため。
// NFC にそろえるのは、macOS で書き足した濁点などが、Windows と別の表現になっていることがあるため。
func ReadFile(dir, name string) (text string, exists bool, err error) {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", true, err
	}
	if !utf8.Valid(data) {
		return "", true, ErrNotUTF8
	}
	return norm.NFC.String(strings.TrimPrefix(string(data), bom)), true, nil
}
