package main

import (
	"errors"
	"testing"
)

// ダイアログをやめても誤りにしない。ほかの誤りはそのまま返す。
func TestIgnoreCancel(t *testing.T) {
	if path, err := ignoreCancel("", errors.New("cancelled by user")); path != "" || err != nil {
		t.Errorf("やめたとき: %q, %v", path, err)
	}
	if path, err := ignoreCancel(`C:\data`, nil); path != `C:\data` || err != nil {
		t.Errorf("選んだとき: %q, %v", path, err)
	}
	if _, err := ignoreCancel("", errors.New("開けません")); err == nil {
		t.Error("ほかの誤りを消した")
	}
}
