package enginetest

import (
	"testing"

	. "vimquest/internal/engine"
)

// growth_test.go — 버퍼 기하급수 성장 경로의 결정적 회귀 테스트.
//
// fuzz 하네스(fuzz_test.go)는 yank→paste 반복이 만드는 기하급수 성장을
// 성장 상한으로 조기 통과 처리한다 — 느린 CI 러너에서 -fuzztime 마감을
// 넘겨 거짓 FAIL 이 나는 것을 막기 위한 의도적 설계다. 그 대가로 fuzz 가
// 이 경로를 깊게 타지 않게 되므로, 여기서 같은 성장 경로를 작은 규모로
// 고정해 매 CI 마다 반드시 실행되게 박아둔다: 성장 자체가 정확한 배수로
// 일어나는지(로직), 성장 후에도 커서·undo 불변식이 유지되는지(안전성).

// TestLinewisePasteDoublingKeepsInvariants 는 줄 단위 성장 경로를 고정한다:
// "ggVGyp" 는 버퍼 전체를 linewise yank 해 아래에 붙이므로 한 번에 줄 수가
// 정확히 2배가 된다. 6회 반복해 5줄 → 320줄.
func TestLinewisePasteDoublingKeepsInvariants(t *testing.T) {
	e := NewEditor([]string{"a", "b", "c", "d", "e"})
	for i := 0; i < 6; i++ {
		before := e.LineCount()
		feedKeys(e, "ggVGyp")
		checkInvariants(t, e)
		if got := e.LineCount(); got != before*2 {
			t.Fatalf("round %d: 줄 수 %d → %d, want 정확히 2배(%d)", i, before, got, before*2)
		}
	}
	if got := e.LineCount(); got != 5*(1<<6) {
		t.Fatalf("6회 더블링 후 줄 수 = %d, want %d", got, 5*(1<<6))
	}
}

// TestCharwisePasteDoublingKeepsInvariants 는 한 줄 안의 성장 경로를 고정한다:
// "0y$p" 는 줄 전체를 charwise yank 해 줄 안에 다시 붙이므로 한 번에 줄
// 길이가 정확히 2배가 된다. 10회 반복해 6 rune → 6,144 rune.
func TestCharwisePasteDoublingKeepsInvariants(t *testing.T) {
	e := NewEditor([]string{"abcdef"})
	for i := 0; i < 10; i++ {
		before := e.LineLen(0)
		feedKeys(e, "0y$p")
		checkInvariants(t, e)
		if got := e.LineLen(0); got != before*2 {
			t.Fatalf("round %d: 줄 길이 %d → %d, want 정확히 2배(%d)", i, before, got, before*2)
		}
	}
	if got := e.LineLen(0); got != 6*(1<<10) {
		t.Fatalf("10회 더블링 후 줄 길이 = %d, want %d", got, 6*(1<<10))
	}
}
