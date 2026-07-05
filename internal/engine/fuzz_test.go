package engine

import "testing"

// checkInvariants 는 임의 입력 뒤에도 항상 성립해야 하는 Editor 불변식을
// 검사한다(F3). 성질 자체(어떤 결과가 "옳은가")는 검증하지 않는다 — 그건
// editor_test.go 의 예제 테스트 몫이고, 여기서는 "절대 깨지면 안 되는 것"만 본다.
func checkInvariants(t *testing.T, e *Editor) {
	t.Helper()
	if len(e.lines) == 0 {
		t.Fatalf("버퍼에 줄이 0개 — 항상 최소 1줄이어야 한다")
	}
	if e.row < 0 || e.row >= len(e.lines) {
		t.Fatalf("row=%d 범위 밖(lines=%d)", e.row, len(e.lines))
	}
	if e.col < 0 || e.col > len(e.lines[e.row]) {
		t.Fatalf("col=%d 범위 밖(현재 줄 길이=%d)", e.col, len(e.lines[e.row]))
	}
	if len(e.undo) > undoCap {
		t.Fatalf("len(e.undo)=%d > undoCap=%d (B1 상한 위반)", len(e.undo), undoCap)
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
	f.Fuzz(func(t *testing.T, s string) {
		e := NewEditor(append([]string(nil), buf...))
		for _, k := range ParseKeys(s) {
			e.Feed(k)
			checkInvariants(t, e)
		}
	})
}
