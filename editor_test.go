package main

import (
	"strings"
	"testing"
)

// feedKeys 는 "diw", "cw bye<esc>" 같은 입력 문자열을 파싱해 엔진에 흘려보낸다.
// 토큰화 자체는 프로덕션 코드 keys.go 의 parseKeys 에 위임한다.
func feedKeys(e *Editor, s string) {
	for _, k := range parseKeys(s) {
		e.Feed(k)
	}
}

// run 은 단일 줄 버퍼에서 키를 실행하고 결과 줄을 돌려준다.
func run(t *testing.T, start, keys string) string {
	t.Helper()
	e := NewEditor([]string{start})
	feedKeys(e, keys)
	return strings.Join(e.Lines(), "\n")
}

func eq(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", name, got, want)
	}
}

func TestDeleteMotions(t *testing.T) {
	eq(t, "dw", run(t, "hello world", "dw"), "world")
	eq(t, "dw-lastword", run(t, "hello", "dw"), "")
	eq(t, "de", run(t, "hello world", "de"), " world")
	eq(t, "d$", run(t, "hello world", "6ld$"), "hello ")
	eq(t, "D", run(t, "hello world", "6lD"), "hello ")
	eq(t, "x", run(t, "abc", "x"), "bc")
	eq(t, "3x", run(t, "abcde", "3x"), "de")
	eq(t, "dfx", run(t, "hello", "dfl"), "lo")
	eq(t, "dtx", run(t, "hello", "dtl"), "llo")
	eq(t, "d2w", run(t, "one two three", "d2w"), "three")
}

func TestChangeMotions(t *testing.T) {
	eq(t, "cw=ce", run(t, "hello world", "cwbye<esc>"), "bye world")
	eq(t, "ciw", run(t, "foo bar baz", "8lciwX<esc>"), "foo bar X")
	eq(t, "cc", run(t, "hello", "ccnew<esc>"), "new")
	eq(t, "C", run(t, "hello world", "6lCthere<esc>"), "hello there")
}

func TestTextObjects(t *testing.T) {
	eq(t, "diw-mid", run(t, "foo bar baz", "4ldiw"), "foo  baz")
	eq(t, "daw", run(t, "foo bar baz", "4ldaw"), "foo baz")
	eq(t, "ci(", run(t, "foo(bar)baz", "4lci(X<esc>"), "foo(X)baz")
	eq(t, "di(", run(t, "foo(bar)baz", "5ldi("), "foo()baz")
	eq(t, "da(", run(t, "foo(bar)baz", "5lda("), "foobaz")
	eq(t, "ci-quote", run(t, "say \"hi\" now", "6lci\"X<esc>"), "say \"X\" now")
	eq(t, "ci{", run(t, "x{a,b}y", "3lci{Z<esc>"), "x{Z}y")
	eq(t, "ci[", run(t, "x[a,b]y", "3lci[Z<esc>"), "x[Z]y")
}

func TestYankPaste(t *testing.T) {
	// yiw 로 단어 복사 후 끝에 붙여넣기
	eq(t, "yy-p", strings.Join(func() []string {
		e := NewEditor([]string{"line1", "line2"})
		feedKeys(e, "yyp")
		return e.Lines()
	}(), "|"), "line1|line1|line2")
	eq(t, "yiw-p", run(t, "ab cd", "yiw$p"), "ab cdab")
	eq(t, "x-p-swap", run(t, "ab", "xp"), "ba")
}

func TestInsertModes(t *testing.T) {
	eq(t, "i", run(t, "bc", "iA<esc>"), "Abc")
	eq(t, "a", run(t, "bc", "aA<esc>"), "bAc")
	eq(t, "A", run(t, "bc", "AX<esc>"), "bcX")
	eq(t, "I", run(t, "  bc", "IX<esc>"), "  Xbc")
	eq(t, "o", strings.Join(func() []string {
		e := NewEditor([]string{"a"})
		feedKeys(e, "oX<esc>")
		return e.Lines()
	}(), "|"), "a|X")
	eq(t, "O", strings.Join(func() []string {
		e := NewEditor([]string{"a"})
		feedKeys(e, "OX<esc>")
		return e.Lines()
	}(), "|"), "X|a")
}

func TestReplaceAndTilde(t *testing.T) {
	eq(t, "r", run(t, "abc", "rz"), "zbc")
	eq(t, "2rz", run(t, "abc", "2rz"), "zzc")
	eq(t, "tilde", run(t, "abc", "~"), "Abc")
}

func TestUndoRedo(t *testing.T) {
	e := NewEditor([]string{"hello"})
	feedKeys(e, "x") // "ello"
	feedKeys(e, "x") // "llo"
	feedKeys(e, "u") // "ello"
	eq(t, "undo1", strings.Join(e.Lines(), ""), "ello")
	feedKeys(e, "u") // "hello"
	eq(t, "undo2", strings.Join(e.Lines(), ""), "hello")
	feedKeys(e, "<c-r>") // redo "ello"
	eq(t, "redo", strings.Join(e.Lines(), ""), "ello")
}

func TestDotRepeat(t *testing.T) {
	// x 를 . 으로 반복
	eq(t, "dot-x", run(t, "abcdef", "x.."), "def")
	// dw 를 반복
	eq(t, "dot-dw", run(t, "a b c d", "dw."), "c d")
	// 삽입 변경 반복: ciwX 후 다음 단어에서 .
	eq(t, "dot-insert", run(t, "aa bb", "ciwX<esc>w."), "X X")
}

func TestCountWithLinewise(t *testing.T) {
	e := NewEditor([]string{"a", "b", "c", "d"})
	feedKeys(e, "2dd")
	eq(t, "2dd", strings.Join(e.Lines(), "|"), "c|d")
}

func TestVisualMode(t *testing.T) {
	eq(t, "v-l-d", run(t, "hello", "vlld"), "lo")
	eq(t, "viw-d", run(t, "foo bar baz", "4lviwd"), "foo  baz")
	// V 줄 선택 삭제
	e := NewEditor([]string{"a", "b", "c"})
	feedKeys(e, "Vjd")
	eq(t, "V-j-d", strings.Join(e.Lines(), "|"), "c")
}

func TestFindAndMotion(t *testing.T) {
	e := NewEditor([]string{"hello world"})
	feedKeys(e, "fw")
	if e.col != 6 {
		t.Errorf("fw: col=%d want 6", e.col)
	}
	feedKeys(e, "0tw")
	if e.col != 5 {
		t.Errorf("tw: col=%d want 5", e.col)
	}
	feedKeys(e, "$")
	if e.col != 10 {
		t.Errorf("$: col=%d want 10", e.col)
	}
	feedKeys(e, "0")
	if e.col != 0 {
		t.Errorf("0: col=%d want 0", e.col)
	}
}
