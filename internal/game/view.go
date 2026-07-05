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
	Best    int // 갱신 전 best (0 이면 이전 클리어 기록 없음)
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
func (g *Game) Selection() (world, level int) { return g.selRow, g.selCol }

func (g *Game) ProgressFor(id string) store.LevelProgress { return g.progress[id] }

func (g *Game) DrillStreak() int { return g.drillStreak }

func (g *Game) DrillTotals() (keys, par int) { return g.drillTotalKeys, g.drillTotalPar }
