package enginetest

import (
	"testing"

	. "vimquest/internal/engine"
)

// checkInvariants 는 임의 입력 뒤에도 항상 성립해야 하는 Editor 불변식을
// 검사한다. 성질 자체(어떤 결과가 "옳은가")는 검증하지 않는다 — 그건
// editor_test.go 의 예제 테스트 몫이고, 여기서는 "절대 깨지면 안 되는 것"만
// 본다. 전부 공개 검사 API(LineCount/LineLen/Row/Col/UndoDepth)로 확인한다.
func checkInvariants(t *testing.T, e *Editor) {
	t.Helper()
	if e.LineCount() == 0 {
		t.Fatalf("버퍼에 줄이 0개 — 항상 최소 1줄이어야 한다")
	}
	if e.Row() < 0 || e.Row() >= e.LineCount() {
		t.Fatalf("row=%d 범위 밖(lines=%d)", e.Row(), e.LineCount())
	}
	if e.Col() < 0 || e.Col() > e.LineLen(e.Row()) {
		t.Fatalf("col=%d 범위 밖(현재 줄 길이=%d)", e.Col(), e.LineLen(e.Row()))
	}
	if e.UndoDepth() > UndoCap {
		t.Fatalf("UndoDepth()=%d > UndoCap=%d (상한 위반)", e.UndoDepth(), UndoCap)
	}
}

// FuzzEditorNeverPanics 는 임의의 키 문자열을 먹여도 Editor 가 패닉하거나
// 불변식을 깨지 않는지 검사한다. engine 은 game 을 import 할 수 없어(단방향
// 의존) 레벨 데이터를 직접 시드로 쓸 수 없으므로, 실제 레벨 Solution 을
// 옮겨 적은 대표 시퀀스(모션·연산자·텍스트객체·삽입·검색·비주얼·undo/redo)를
// 시드 코퍼스로 쓴다 — 커리큘럼이 늘 때 이 목록도 함께 넓히는 것을 권장.
func FuzzEditorNeverPanics(f *testing.F) {
	seeds := []string{
		"jjlllljjlllll", "wwwwwwwww", "$0jj$", "fKf$", "lllxjlljhhhxjlllllll",
		"flxx", "jddjdd", "wdw", "A, World!<esc>", "wcwthat<esc>",
		"cwbar<esc>w.w.", "wdaw", "f(ci(newArg<esc>", `f"ci"new<esc>`,
		"yyp", "$ciw42<esc>j$.", "/K<cr>/$<cr>", "/K<cr>nn/$<cr>",
		"4w4w", "FK$", "tX$", "wd2w", "4wf$", "wvEEd", "wyawP",
		"wviwcOK<esc>", "wvawdwwciwNEW<esc>", "xp", "ddp", "yyjP",
		"jddddu", "ddjp", "wdawj$ciw99<esc>j0xp",
		"u<c-r>u<c-r>", "vjjd", "Vjjd", "ihello<esc>", "ohello<esc>",
		"3x", "2dd", "d2fl", "dF(", "gg", "G", "1G", "",
		"i<esc>u", "raa~~u", "cc<esc>.",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	buf := []string{"hello world", "foo(bar, baz)", "line one", "  indented", ""}
	// 성장 상한. yank→paste 반복은 버퍼를 기하급수적으로 불릴 수 있고("VGyp"
	// 반복 = 줄 수 2배, "y$p" 반복 = 한 줄 길이 2배), pushUndo 가 커맨드마다
	// 버퍼 전체를 복제하므로 버퍼가 커진 뒤엔 키 하나가 수 초까지 느려진다.
	// 느린 CI 러너에서 -fuzztime 마감 시점에 그런 입력을 물고 있으면 크래시
	// 없이도 "context deadline exceeded" 로 FAIL 한다(실제 발생). 큰 버퍼가
	// 패닉 불변식 검사에 새 정보를 주지는 않으므로, 상한에 닿으면 그 입력은
	// 거기서 통과 처리한다. paste 는 count 를 받지 않아 커맨드 1회 비용은
	// 유한하고, 성장은 키 사이에서만 누적된다 — 키마다 검사하면 충분하다.
	// 두 성장 경로(줄 수·현재 줄 길이) 모두 커서 줄에서 일어나므로 O(1) 검사.
	const fuzzMaxLines, fuzzMaxLineRunes = 2000, 1 << 16
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 4096 {
			return // 키 개수도 유계로 — 불변식 탐색에 이 이상은 불필요
		}
		e := NewEditor(append([]string(nil), buf...))
		for _, k := range ParseKeys(s) {
			e.Feed(k)
			checkInvariants(t, e)
			if e.LineCount() > fuzzMaxLines || e.LineLen(e.Row()) > fuzzMaxLineRunes {
				return
			}
		}
	})
}
