package engine

import (
	"strings"
	"testing"
)

// feedKeys 는 "diw", "cw bye<esc>" 같은 입력 문자열을 파싱해 엔진에 흘려보낸다.
// 토큰화 자체는 프로덕션 코드 keys.go 의 parseKeys 에 위임한다.
func feedKeys(e *Editor, s string) {
	for _, k := range ParseKeys(s) {
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

// TestOpFindCount 는 B3: opFind(d/c/y + f/F/t/T) 가 count 를 반영하는지
// 확인한다 — 예전엔 e.takeCount() 로 소비만 하고 버려서 "d2fl" 이 "dfl" 처럼
// 첫 번째 'l' 에서만 멈췄다(par 오염 위험, findChar 통합과 함께 수정).
func TestOpFindCount(t *testing.T) {
	eq(t, "d2fl", run(t, "hello", "d2fl"), "o")
	eq(t, "dF(", run(t, "foo(bar)", "$dF("), "foo)")
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

// TestUndoCapLimitsDepth 는 B1: undo 스택이 undoCap 을 넘지 않는지 확인한다
// (200회 편집 후에도 앞쪽이 잘려나가 상한을 유지해야 함).
func TestUndoCapLimitsDepth(t *testing.T) {
	e := NewEditor([]string{strings.Repeat("x", 250)})
	for i := 0; i < 200; i++ {
		feedKeys(e, "x")
	}
	if len(e.undo) > undoCap {
		t.Fatalf("len(e.undo)=%d want <= %d", len(e.undo), undoCap)
	}
}

// TestNoOpInsertDoesNotPushUndo 는 B2: "i<esc>" 처럼 버퍼를 실제로 바꾸지
// 않은 insert 가 undo 스택에 스냅샷을 남기지 않는지 확인한다 — 예전엔 이게
// 쌓여서 그 다음 "u" 가 커서만 되돌리고 "아무 일도 안 하는 것처럼" 보였다.
func TestNoOpInsertDoesNotPushUndo(t *testing.T) {
	e := NewEditor([]string{"hello"})
	feedKeys(e, "i<esc>")
	if len(e.undo) != 0 {
		t.Fatalf("무변경 insert 후 len(e.undo)=%d want 0", len(e.undo))
	}
}

// TestNoOpReplaceDoesNotPushUndo 는 B2 일반화: 'r' 로 같은 문자를 치환해도
// (insert 뿐 아니라 전체 무변경 커밋에 적용) undo 스택이 늘지 않는지 확인한다.
func TestNoOpReplaceDoesNotPushUndo(t *testing.T) {
	e := NewEditor([]string{"abc"})
	e.SetCursor(0, 0)
	feedKeys(e, "ra") // col0 의 'a' 를 그대로 'a' 로 치환(무변경)
	if len(e.undo) != 0 {
		t.Fatalf("무변경 치환 후 len(e.undo)=%d want 0", len(e.undo))
	}
}

// TestChangeWordDotRepeatTwice 는 B2 undoPending 누수 회귀 테스트: dot 재생이
// insert 종료(finishInsertDot)를 거치면서 undoPending 이 소비되지 않고 다음
// 진짜 커맨드로 새어 들어가 dot 을 엉뚱한 키로 덮어쓰던 결함을 잡는다
// ("cwbar<esc>w.w." 가 세 단어 전부를 바꿔야 한다 — 안 그러면 두 번째 "." 가
// 무력화된다).
func TestChangeWordDotRepeatTwice(t *testing.T) {
	got := run(t, "foo foo foo", "cwbar<esc>w.w.")
	eq(t, "cw-dot-twice", got, "bar bar bar")
}

// TestHugeCountInVisualDoesNotHang 은 F3 fuzz 로 발견한 결함의 회귀 테스트:
// Visual 모드의 count 누적(B7)이 normal.go 의 maxCount 상한을 상속하지 않아,
// "V" + 긴 숫자열 + 모션이 motionOnce 반복을 그대로 O(count) 로 돌려 멈췄다 —
// count 누적 로직이 두 곳에 복제되면서 상한만 한쪽에 안 걸린 사례.
func TestHugeCountInVisualDoesNotHang(t *testing.T) {
	e := NewEditor([]string{"hello world"})
	feedKeys(e, "V2000000000w") // 상한 없으면 이 한 줄이 테스트를 멈춘다
	if e.count != 0 {
		t.Fatalf("count=%d, 모션 소비 후 0 이어야 함", e.count)
	}
}

// TestHugeCountDoesNotHang 은 F3 fuzz 로 발견한 결함의 회귀 테스트: count
// 접두사에 상한이 없으면 "2000000000B" 같은 입력이 doMotion 의 O(count) 루프를
// 그대로 실행해 멈춘다(웹 빌드에선 탭이 얼어붙음). maxCount 로 잘려야 한다.
func TestHugeCountDoesNotHang(t *testing.T) {
	e := NewEditor([]string{"hello world"})
	feedKeys(e, "2000000000B") // 상한 없으면 이 한 줄이 테스트를 멈춘다
	if e.count != 0 {
		t.Fatalf("count=%d, 모션 소비 후 0 이어야 함", e.count)
	}
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

// TestGotoLineWithCount 는 {N}G 가 실제로 N번째 줄로 이동하는지 확인한다.
// (과거엔 e.count 가 takeCount() 로 이미 리셋된 뒤 참조돼 count 유무와 무관하게
// 항상 마지막 줄로 이동하는 결함이 있었다.)
func TestGotoLineWithCount(t *testing.T) {
	e := NewEditor([]string{"a", "b", "c", "d", "e"})
	feedKeys(e, "4G")
	if e.row != 3 {
		t.Fatalf("4G: row=%d want 3", e.row)
	}
}

func TestGotoLineNoCountGoesLast(t *testing.T) {
	e := NewEditor([]string{"a", "b", "c"})
	feedKeys(e, "G")
	if e.row != 2 {
		t.Fatalf("G(count 없음): row=%d want 2", e.row)
	}
}

func TestGotoLineTopWithCount(t *testing.T) {
	e := NewEditor([]string{"a", "b", "c", "d", "e"})
	e.row = 4
	feedKeys(e, "2gg")
	if e.row != 1 {
		t.Fatalf("2gg: row=%d want 1", e.row)
	}
}

func TestGotoLineTopNoCountGoesFirst(t *testing.T) {
	e := NewEditor([]string{"a", "b", "c"})
	e.row = 2
	feedKeys(e, "gg")
	if e.row != 0 {
		t.Fatalf("gg(count 없음): row=%d want 0", e.row)
	}
}

func TestSearch(t *testing.T) {
	e := NewEditor([]string{"foo bar target baz"})
	feedKeys(e, "/target<cr>")
	if e.col != 8 { // "target" 시작 열
		t.Fatalf("search cursor col=%d want 8", e.col)
	}
}

func TestSearchRepeat(t *testing.T) {
	e := NewEditor([]string{"x x x target x x"})
	feedKeys(e, "/x<cr>")
	before := e.col
	feedKeys(e, "n")
	if e.col == before {
		t.Fatal("n 이 다음 매치로 이동하지 않음")
	}
	feedKeys(e, "N")
	if e.col != before {
		t.Fatalf("N 이 이전 매치로 되돌아가지 않음: col=%d want %d", e.col, before)
	}
}

func TestSearchBackward(t *testing.T) {
	e := NewEditor([]string{"target middle target"})
	e.col = len(e.line()) - 1
	feedKeys(e, "?target<cr>")
	if e.col != 0 && e.col != 14 {
		t.Fatalf("역검색 실패: col=%d", e.col)
	}
}

// TestParseKeysUTF8 는 A4: 멀티바이트 문자가 바이트 단위로 쪼개지지 않고
// rune 하나당 Key 하나로 파싱되는지 확인한다.
func TestParseKeysUTF8(t *testing.T) {
	keys := ParseKeys("한글<esc>")
	if len(keys) != 3 {
		t.Fatalf("len(keys)=%d want 3 (got %+v)", len(keys), keys)
	}
	if keys[0].R != '한' || keys[1].R != '글' {
		t.Fatalf("keys[0..1]=%+v want 한,글", keys[:2])
	}
	if keys[2].S != "esc" {
		t.Fatalf("keys[2]=%+v want esc", keys[2])
	}
}

func TestSearchEscCancels(t *testing.T) {
	e := NewEditor([]string{"abc"})
	r0, c0 := e.row, e.col
	feedKeys(e, "/xyz<esc>")
	if e.row != r0 || e.col != c0 || e.searching {
		t.Fatal("esc 취소 후 상태가 원위치가 아니거나 searching 이 남아있음")
	}
}
