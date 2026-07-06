// macro_test.go — 매크로(q/@/@@)와 % 모션(matchBracket/gotoPercentLine) 검증.
package enginetest

import (
	"strings"
	"testing"

	. "vimquest/internal/engine"
)

// TestMacroUnrecordedRegisterIsNoop 은 한 번도 기록하지 않은 레지스터를
// 재생해도 패닉 없이 아무 일도 안 일어나는지 확인한다.
func TestMacroUnrecordedRegisterIsNoop(t *testing.T) {
	e := NewEditor([]string{"hello"})
	feedKeys(e, "@a")
	eq(t, "unrecorded @a", strings.Join(e.Lines(), "\n"), "hello")
	if e.Row() != 0 || e.Col() != 0 {
		t.Fatalf("커서가 움직임: row=%d col=%d", e.Row(), e.Col())
	}
}

// TestMacroRecordReplay 는 record→replay 왕복의 최소 예시.
func TestMacroRecordReplay(t *testing.T) {
	e := NewEditor([]string{"aaa", "aaa", "aaa"})
	feedKeys(e, "qaxjq") // 레지스터 a: 글자 하나 지우고 아래 줄로
	feedKeys(e, "@a")    // 한 번 더 재생
	got := strings.Join(e.Lines(), "\n")
	want := "aa\naa\naaa"
	eq(t, "record+replay", got, want)
}

// TestMacroRecordingBoundaryExcludesQAndRegister 는 시작/종료 "q"와 레지스터
// 문자 자체가 recordBuf(=매크로 내용)에 안 남는지 확인한다 — 재생 시 "q"/"a"가
// 실제 커맨드로 실행되면 버퍼가 달라지므로 이걸로 검증할 수 있다.
func TestMacroRecordingBoundaryExcludesQAndRegister(t *testing.T) {
	e := NewEditor([]string{"hello"})
	feedKeys(e, "qaxq") // 기록 내용은 정확히 "x" 하나여야 한다
	feedKeys(e, "@a")   // "x"만 재생돼야 함 — "q"나 "a"가 섞였으면 패닉하거나 다른 결과가 남
	got := strings.Join(e.Lines(), "\n")
	want := "llo" // "hello" 에서 x 두 번(기록 중 1회 + 재생 1회)
	eq(t, "record boundary", got, want)
}

// TestMacroCountRepeat 는 "3@a" 같은 count 반복을 검증한다.
func TestMacroCountRepeat(t *testing.T) {
	e := NewEditor([]string{"aaaaaaa"})
	feedKeys(e, "qaxq") // 레지스터 a: 글자 하나 지움("x")
	feedKeys(e, "3@a")  // 3번 반복
	got := strings.Join(e.Lines(), "\n")
	want := "aaa" // 기록 중 1회 + 재생 3회 = 4글자 삭제(7-4=3)
	eq(t, "count repeat", got, want)
}

// TestMacroAtAtRepeatsLastRegister 는 "@@" 가 마지막으로 재생한 레지스터를
// 다시 재생하는지 확인한다.
func TestMacroAtAtRepeatsLastRegister(t *testing.T) {
	e := NewEditor([]string{"aaaaa"})
	feedKeys(e, "qaxq") // 기록 1회 삭제
	feedKeys(e, "@a")   // 재생 1회
	feedKeys(e, "@@")   // 마지막 레지스터(a) 재생 1회 더
	got := strings.Join(e.Lines(), "\n")
	want := "aa" // 5 - 1(기록) - 1(@a) - 1(@@) = 2
	eq(t, "@@ repeats last register", got, want)
}

// TestMacroRecursionGuardStops 는 자기 참조 매크로가 macroDepth 상한에서
// 패닉 없이 멈추는지 확인한다(무한 재귀 방지).
func TestMacroRecursionGuardStops(t *testing.T) {
	e := NewEditor([]string{strings.Repeat("a", 500)})
	feedKeys(e, "qax@aq") // 레지스터 a 안에서 자기 자신(@a)을 호출
	feedKeys(e, "@a")     // 재생 시작 — macroDepth 상한에서 멈춰야 함(패닉 금지)
	if e.LineCount() == 0 {
		t.Fatalf("버퍼가 비정상")
	}
}

// TestMacroDotRepeatsLastInnerCommandNotWholeMacro 는 매크로 재생 후 "."이
// 매크로 전체가 아니라 매크로 안 마지막 변경 커맨드만 반복하는지 확인한다 —
// 실제 vim과 동일한 핵심 동작.
func TestMacroDotRepeatsLastInnerCommandNotWholeMacro(t *testing.T) {
	e := NewEditor([]string{"aaaa", "aaaa"})
	feedKeys(e, "qaxxq") // 레지스터 a: 글자 두 개 지움("xx") — 기록 중 실행돼 첫 줄이 이미 "aa"
	feedKeys(e, "@a")    // 재생 — 이동 없는 매크로라 같은(첫) 줄에서 "aa"->"" 로 마저 삭제
	feedKeys(e, "j.")    // 다음 줄로 이동 후 "." — "xx" 전체가 아니라 마지막 "x" 하나만 반복돼야 함
	got := strings.Join(e.Lines(), "\n")
	want := "\naaa" // 첫 줄은 비고, 둘째 줄은 x 한 번만 지워짐(마지막 내부 커맨드만 반복)
	eq(t, "dot repeats last inner command", got, want)
}

// TestMacroAcrossInsertMode 는 Insert 모드를 넘나드는 매크로(qaihello<esc>q)의
// 기록/재생을 검증한다.
func TestMacroAcrossInsertMode(t *testing.T) {
	e := NewEditor([]string{"", ""})
	feedKeys(e, "qaihi<esc>q") // 레지스터 a: insert 로 "hi" 삽입
	feedKeys(e, "j@a")         // 다음 줄로 이동 후 재생
	got := strings.Join(e.Lines(), "\n")
	want := "hi\nhi"
	eq(t, "macro across insert mode", got, want)
}

// TestMatchBracketOnOpenAndClose 는 여는/닫는 괄호 위에서 %가 짝으로 이동하는지.
func TestMatchBracketOnOpenAndClose(t *testing.T) {
	e := NewEditor([]string{"foo(bar)baz"})
	e.SetCursor(0, 3) // '('
	feedKeys(e, "%")
	if e.Row() != 0 || e.Col() != 7 {
		t.Fatalf("( -> ) 실패: row=%d col=%d want 0,7", e.Row(), e.Col())
	}
	feedKeys(e, "%") // 다시 누르면 여는 괄호로 복귀
	if e.Row() != 0 || e.Col() != 3 {
		t.Fatalf(") -> ( 실패: row=%d col=%d want 0,3", e.Row(), e.Col())
	}
}

// TestMatchBracketScansLineForFirstBracket 는 괄호 위가 아닐 때 현재 줄에서
// 오른쪽으로 첫 괄호를 찾아 그 지점부터 적용하는지 확인한다.
func TestMatchBracketScansLineForFirstBracket(t *testing.T) {
	e := NewEditor([]string{"x = [1, 2, 3]"})
	e.SetCursor(0, 0)
	feedKeys(e, "%")
	if e.Row() != 0 || e.Col() != 12 {
		t.Fatalf("줄 내 첫 괄호 스캔 실패: row=%d col=%d want 0,12", e.Row(), e.Col())
	}
}

// TestMatchBracketMultiline 은 괄호가 여러 줄에 걸칠 때도 짝을 찾는지 확인한다.
func TestMatchBracketMultiline(t *testing.T) {
	e := NewEditor([]string{"func foo() {", "    bar()", "}"})
	e.SetCursor(0, 11) // '{'
	feedKeys(e, "%")
	if e.Row() != 2 || e.Col() != 0 {
		t.Fatalf("멀티라인 매치 실패: row=%d col=%d want 2,0", e.Row(), e.Col())
	}
}

// TestMatchBracketNotFoundIsNoop 은 짝을 못 찾으면 커서가 그대로인지 확인한다.
func TestMatchBracketNotFoundIsNoop(t *testing.T) {
	e := NewEditor([]string{"foo(bar"})
	e.SetCursor(0, 3)
	feedKeys(e, "%")
	if e.Row() != 0 || e.Col() != 3 {
		t.Fatalf("못 찾았는데 커서가 움직임: row=%d col=%d want 0,3", e.Row(), e.Col())
	}
}

// TestPercentWithCountGotoPercentLine 은 count가 있는 "%"(예: "50%")가 괄호와
// 무관하게 파일의 N% 지점 줄로 이동하는지 확인한다.
func TestPercentWithCountGotoPercentLine(t *testing.T) {
	e := NewEditor([]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"})
	feedKeys(e, "50%")
	// vim 공식: line = (n*총줄수 + 99) / 100 = (50*10+99)/100 = 5 (1-indexed) -> row=4
	if e.Row() != 4 {
		t.Fatalf("50%% 이동 실패: row=%d want 4", e.Row())
	}
}

// TestOperatorPlusPercentIsIgnored 는 d% 처럼 연산자 + % 조합이 지원되지
// 않음을 확인한다(설계 결정 — operator.go 는 현재 줄 안 컬럼 범위로만 charwise
// 스팬을 표현하므로 여러 줄에 걸친 % 를 연산자와 결합하지 않는다). 아무것도
// 지워지지 않고 패닉도 없어야 한다.
func TestOperatorPlusPercentIsIgnored(t *testing.T) {
	e := NewEditor([]string{"foo(bar)baz"})
	e.SetCursor(0, 3)
	feedKeys(e, "d%")
	got := strings.Join(e.Lines(), "\n")
	eq(t, "d% is a no-op", got, "foo(bar)baz")
}
