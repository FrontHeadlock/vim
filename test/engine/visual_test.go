package enginetest

import (
	"testing"

	. "vimquest/internal/engine"
)

// TestVisualCountMotion 은 비주얼 모드에서 count 접두 모션(예: 3l)이 count
// 번 반복되는지 확인한다(비주얼 모드가 count 를 무시하면 "3l" 이 한 칸만
// 이동해 실제 Vim 과 다르게 동작한다).
func TestVisualCountMotion(t *testing.T) {
	eq(t, "v3ld", run(t, "abcdef", "v3ld"), "ef")
}

// TestVisualCountLinewiseFallback 은 여러 줄에 걸친 count 모션(3j)이 비주얼
// charwise → linewise 대체 경로와 함께 정확한 줄 수를 지우는지 확인한다.
func TestVisualCountLinewiseFallback(t *testing.T) {
	e := NewEditor([]string{"l0", "l1", "l2", "l3", "l4"})
	feedKeys(e, "v3jd")
	got := e.Lines()
	if len(got) != 1 || got[0] != "l4" {
		t.Fatalf("v3jd 결과 = %v, want [\"l4\"]", got)
	}
}

// TestVisualEscClearsCount 는 비주얼 모드에서 입력하다 만 count 가 esc 로
// 취소된 뒤 다음 Normal 커맨드로 새지 않는지 확인한다("v2<esc>dw" 가
// "2dw"(단어 2개 삭제)처럼 동작하면 안 됨 — count 는 Normal/Visual 이
// 공유하는 필드라 비주얼 쪽에서 지우지 않으면 새어나간다).
func TestVisualEscClearsCount(t *testing.T) {
	eq(t, "v2<esc>dw", run(t, "aaa bbb ccc", "v2<esc>dw"), "bbb ccc")
}

// TestVisualGMovesToFirstNonBlank 은 비주얼 모드의 G(gotoLineOr 공유)가 Normal
// 모드의 G 와 동일하게 목표 줄의 첫 비공백 열로 이동하는지 확인한다 — 실제
// Vim 과 동일한 동작이다.
func TestVisualGMovesToFirstNonBlank(t *testing.T) {
	e := NewEditor([]string{"a", "  bc"})
	feedKeys(e, "vG")
	if e.Row() != 1 || e.Col() != 2 {
		t.Fatalf("vG 후 row=%d col=%d want 1,2 (첫 비공백 열)", e.Row(), e.Col())
	}
}

// TestVisualCountG 는 비주얼 모드에서 count 를 곁들인 G("2G")가 해당 줄로
// 이동하는지 확인한다.
func TestVisualCountG(t *testing.T) {
	e := NewEditor([]string{"a", "b", "c"})
	feedKeys(e, "v2G")
	if e.Row() != 1 {
		t.Fatalf("v2G 후 row=%d want 1", e.Row())
	}
}
