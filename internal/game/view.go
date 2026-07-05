package game

// view.go — 데스크톱 렌더러(cmd/desktop)가 게임 상태를 타입 그대로 읽는
// 전용 창구. 웹 렌더러(web/renderer.js)는 대신 snapshot.go 의 Snapshot() 을
// 쓴다. 어느 쪽이든 읽기 전용이고, 상태를 바꾸는 경로는 Input()/Tick() 과
// LoadLevel/RestartCurrent/EnterLevelSelect 뿐이다.

import (
	"vimquest/internal/engine"
	"vimquest/internal/store"
)

// ClearStats 는 레벨 클리어 화면에 표시할 통계(advance() 시점에 고정).
type ClearStats struct {
	Strokes int
	Par     int
	Stars   int
	Best    int    // 갱신 전 best (0 이면 이전 클리어 기록 없음)
	IsNew   bool   // true 면 이번 클리어가 신기록(렌더러는 재계산하지 않고 이 값만 읽는다)
	Yours   string // 내가 실제로 입력한 키 시퀀스(문자열화) — 별점 무관하게 항상 채워짐
}

func (g *Game) State() State { return g.state }

// Editor 는 편집 엔진 핸들 (공개 API는 읽기 전용, SetCursor 제외).
func (g *Game) Editor() *engine.Editor { return g.ed }

func (g *Game) Level() Level { return g.lv }

func (g *Game) LevelIndex() int { return g.levelIdx }

func (g *Game) Strokes() int { return g.strokes }

// KeysNeed/KeysLeft 는 navigate 레벨의 전체/남은 열쇠 수.
func (g *Game) KeysNeed() int { return g.keysNeed }
func (g *Game) KeysLeft() int { return len(g.keyPos) }

// HasKeyAt 은 (r,c) 위치에 열쇠가 있는지.
func (g *Game) HasKeyAt(r, c int) bool { return g.keyPos[[2]int{r, c}] }

// BellActive 는 visual bell 활성 여부.
func (g *Game) BellActive() bool { return g.bellTTL > 0 }

// ExLine 은 ex-command 입력 중인 버퍼와 상태.
func (g *Game) ExLine() (string, bool) { return string(g.exBuf), g.exMode }

func (g *Game) LastClear() ClearStats { return g.clear }

// Selection 은 레벨 선택 화면의 커서 위치.
func (g *Game) Selection() (world, level int) { return g.selWorld, g.selLevel }

func (g *Game) ProgressFor(id string) store.LevelProgress { return g.progress[id] }

func (g *Game) DrillStreak() int { return g.drillStreak }

func (g *Game) DrillTotals() (keys, par int) { return g.drillTotalKeys, g.drillTotalPar }

// RowSpan 은 한 줄 안에서 강조할 [C1,C2] 구간(둘 다 포함, rune 인덱스).
type RowSpan struct{ Row, C1, C2 int }

// VisualRows 는 현재 비주얼 선택을 렌더러가 바로 쓸 수 있는 행별 구간
// 목록으로 반환한다 — 게임이 한 번만 계산해 두 렌더러가 각자 기하 계산을
// 복제하지 않게 한다(NEW! 판정 통합과 같은 원칙). 선택이 없으면 nil.
func (g *Game) VisualRows() []RowSpan {
	r1, c1, r2, c2, lineMode, ok := g.ed.VisualSpan()
	if !ok {
		return nil
	}
	lines := g.ed.Lines()
	out := make([]RowSpan, 0, r2-r1+1)
	for r := r1; r <= r2 && r < len(lines); r++ {
		lineLen := len([]rune(lines[r]))
		start, end := 0, lineLen-1
		if !lineMode {
			switch {
			case r1 == r2:
				start, end = c1, c2
			case r == r1:
				start = c1
			case r == r2:
				end = c2
			}
		}
		if end < start {
			continue // 빈 줄 등 — 강조할 칸 없음
		}
		out = append(out, RowSpan{Row: r, C1: start, C2: end})
	}
	return out
}

// MatchedRows 는 edit 레벨에서 현재 버퍼의 각 줄이 목표(Target)의 같은 줄과
// 정확히 일치하는지를 나타낸다 — 게임이 한 번만 계산해 두 렌더러가 각자
// line==target[r] 비교를 복제하지 않게 한다. navigate 레벨에서는 nil.
func (g *Game) MatchedRows() []bool {
	if g.lv.Kind != "edit" {
		return nil
	}
	lines := g.ed.Lines()
	out := make([]bool, len(lines))
	for i, l := range lines {
		out[i] = i < len(g.lv.Target) && l == g.lv.Target[i]
	}
	return out
}
