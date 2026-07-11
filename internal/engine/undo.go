package engine

// undo.go — undo/redo 스택.

// UndoCap 은 undo 스택 깊이 상한(B1). 웹 빌드는 -gc=leaking 이라 세션 내
// 회수가 없어(build.sh 참고, drill.go 의 DrillMaxRounds 와 같은 이유) 매 변경마다
// 전체 버퍼 클론을 상한 없이 쌓으면 세션이 길어질수록 누적된다. 퍼즐 버퍼는
// 수 줄이라 100 이면 충분히 넉넉하다.
const UndoCap = 100

type snapshot struct {
	lines [][]rune
	row   int
	col   int
}

func cloneLines(src [][]rune) [][]rune {
	out := make([][]rune, len(src))
	for i, l := range src {
		c := make([]rune, len(l))
		copy(c, l)
		out[i] = c
	}
	return out
}

func linesEqual(a, b [][]rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if string(a[i]) != string(b[i]) {
			return false
		}
	}
	return true
}

func (e *Editor) pushUndo() {
	e.undo = append(e.undo, snapshot{cloneLines(e.lines), e.row, e.col})
	if len(e.undo) > UndoCap {
		e.undo = e.undo[len(e.undo)-UndoCap:]
	}
	e.redo = nil
	// changed 는 여기서 단정하지 않는다 — 실제로 바뀌었는지는 커맨드 경계에서
	// commitUndoIfChanged 가 버퍼를 비교해 확정한다. 그러지 않으면 "i<esc>"
	// (무변경 insert) 하나로도 undo 스택에 스냅샷이 쌓여, 그 다음 "u" 가
	// 커서만 되돌리고 아무 일도 안 한 것처럼 보인다.
	e.undoPending = true
}

// commitUndoIfChanged 는 이번 커맨드 동안 pushUndo 가 호출됐다면, 버퍼가
// 실제로 그 스냅샷과 달라졌을 때만 유지한다. 안 달라졌으면(무변경 커밋)
// 스냅샷을 버리고 false 를 돌려준다 — 호출자는 이 경우 dot 도 갱신하지 않아야
// 한다. pushUndo 가 아예 호출되지 않았으면(예: 순수 이동, yank) 즉시 false.
func (e *Editor) commitUndoIfChanged() bool {
	if !e.undoPending {
		return false
	}
	e.undoPending = false
	last := e.undo[len(e.undo)-1]
	if linesEqual(e.lines, last.lines) {
		e.undo = e.undo[:len(e.undo)-1]
		return false
	}
	return true
}

func (e *Editor) undoOp() {
	if len(e.undo) == 0 {
		return
	}
	e.redo = append(e.redo, snapshot{cloneLines(e.lines), e.row, e.col})
	s := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.lines = s.lines
	e.row, e.col = s.row, s.col
	e.clamp()
}

func (e *Editor) redoOp() {
	if len(e.redo) == 0 {
		return
	}
	e.undo = append(e.undo, snapshot{cloneLines(e.lines), e.row, e.col})
	s := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.lines = s.lines
	e.row, e.col = s.row, s.col
	e.clamp()
}
