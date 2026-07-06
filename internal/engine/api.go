package engine

// api.go — 패키지 밖(게임 규칙·렌더러)용 공개 API.
//
// 내부 필드를 직접 노출하지 않는 이유: 과거 게임 코드가 row/col 을 직접
// 대입하면서 dcol(수직 모션의 목표 열)을 빼먹어 커서가 엉뚱한 열로 튀는
// 버그가 실제로 있었다. 좌표 이동은 SetCursor 하나로만 가능하게 잠근다.

// VisualSpan 은 렌더링용 선택 범위를 반환(없으면 ok=false).
func (e *Editor) VisualSpan() (r1, c1, r2, c2 int, lineMode, ok bool) {
	if e.mode != ModeVisual && e.mode != ModeVisualLine {
		return 0, 0, 0, 0, false, false
	}
	r1, c1, r2, c2 = e.vrow, e.vcol, e.row, e.col
	if r1 > r2 || (r1 == r2 && c1 > c2) {
		r1, c1, r2, c2 = r2, c2, r1, c1
	}
	return r1, c1, r2, c2, e.mode == ModeVisualLine, true
}

// Row/Col 은 현재 커서 위치.
func (e *Editor) Row() int { return e.row }
func (e *Editor) Col() int { return e.col }

// SetCursor 는 커서를 (row,col) 로 옮기면서 j/k 의 목표 열(dcol)도 함께 맞춘다.
func (e *Editor) SetCursor(row, col int) {
	e.row, e.col = row, col
	e.clamp()
	e.dcol = e.col
}

// Cell 은 (row,col) 의 문자를 돌려준다. 버퍼 범위 밖이면 ok=false.
func (e *Editor) Cell(row, col int) (rune, bool) {
	if row < 0 || row >= len(e.lines) || col < 0 || col >= len(e.lines[row]) {
		return 0, false
	}
	return e.lines[row][col], true
}

// SetCell 은 (row,col) 문자를 제자리에서 치환한다 — 길이를 바꾸지 않으므로
// 같은 줄의 다른 좌표(열쇠·출구 등)가 절대 밀리지 않는다. 범위 밖이면 no-op.
// deleteChars 같은 편집 명령과 달리 커서·undo 스택을 건드리지 않는다 — navigate
// 게임 규칙(버그 처치)처럼 "텍스트 편집이 아닌 게임판 상태 변경"에 쓰기 위한 API 다.
func (e *Editor) SetCell(row, col int, r rune) {
	if row < 0 || row >= len(e.lines) || col < 0 || col >= len(e.lines[row]) {
		return
	}
	e.lines[row][col] = r
}

// LineCount 는 버퍼의 줄 수. Lines() 와 달리 복사 없이 크기만 알려준다
// (테스트 하네스·fuzz 불변식 검사처럼 매 키마다 조회하는 소비자용).
func (e *Editor) LineCount() int { return len(e.lines) }

// LineLen 은 row 줄의 rune 수. 범위 밖이면 0.
func (e *Editor) LineLen(row int) int {
	if row < 0 || row >= len(e.lines) {
		return 0
	}
	return len(e.lines[row])
}

// UndoDepth 는 undo 스택 깊이 — 항상 UndoCap 이하다(불변식 검사용).
func (e *Editor) UndoDepth() int { return len(e.undo) }

// Mode 는 현재 편집 모드(Normal/Insert/Visual/VisualLine).
func (e *Editor) Mode() Mode { return e.mode }

// Searching 은 검색 쿼리(/ ?) 입력 중인지 여부.
func (e *Editor) Searching() bool { return e.searching }

// MidCommand 는 여러 키로 이뤄진 명령이 진행 중인지 알려준다
// (f/t/r/g 인자 대기, 연산자(d/c/y) 대기, 텍스트 객체 한정자(i/a) 대기).
func (e *Editor) MidCommand() bool { return e.await != "" || e.op != 0 || e.pendObj != 0 }

// PendingString 은 상태바에 표시할 입력 중인 명령 문자열(예: "2d", "/foo").
func (e *Editor) PendingString() string { return e.pendingStr }

// LastKey 는 마지막으로 입력된 키의 표시용 문자열.
func (e *Editor) LastKey() string { return e.lastKey }

// Recording 은 현재 녹화 중인 매크로 레지스터(0=미기록) — 상태바에
// "recording @a" 처럼 표시하기 위한 접근자.
func (e *Editor) Recording() rune { return e.recording }
