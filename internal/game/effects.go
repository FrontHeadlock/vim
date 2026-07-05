package game

// effects.go — 터미널식 피드백(문자 치환/반전 연출, visual bell)과 그 수명 관리.
// 실제 버퍼 내용은 절대 바꾸지 않고 렌더링에서만 겹쳐 그린다 — "터미널이 낼 수
// 있는 것만 낸다"는 설계 원칙(파티클·이미지 에셋 금지)의 게임 쪽 절반이다.

import "vimquest/internal/platform"

// Effect 는 몇 프레임 동안 표시되는 문자 치환/반전 연출 한 건.
type Effect struct {
	Row, Col int
	Glyph    rune // 0 이면 문자 치환 없음(Invert 만 적용)
	Invert   bool
	TTL      int // 남은 프레임 수
}

// fireEvent 는 게임 이벤트(열쇠 획득/버그 처치/막힌 키/레벨 클리어)를
// 사운드(platform.Sfx)와 화면 연출(spawnEffect) 양쪽에 동시에 통지한다.
func (g *Game) fireEvent(name string, row, col int) {
	platform.Sfx(name)
	g.spawnEffect(name, row, col)
}

func (g *Game) spawnEffect(name string, row, col int) {
	switch name {
	case "bug":
		g.effects = append(g.effects, Effect{Row: row, Col: col, Glyph: 'x', TTL: 10})
	case "key":
		g.effects = append(g.effects, Effect{Row: row, Col: col, Invert: true, TTL: 6})
	case "blocked":
		g.bellTTL = 2
	}
}

// Tick 은 매 프레임 이펙트 TTL 을 줄이고 만료된 것을 정리한다.
// 키 입력과 무관하게 프론트엔드가 프레임마다(또는 이펙트가 살아있는 동안만) 호출한다.
func (g *Game) Tick() {
	if g.bellTTL > 0 {
		g.bellTTL--
	}
	live := g.effects[:0]
	for _, e := range g.effects {
		e.TTL--
		if e.TTL > 0 {
			live = append(live, e)
		}
	}
	g.effects = live
}

// EffectAt 은 (r,c) 에 걸린 활성 이펙트를 돌려준다(없으면 ok=false).
func (g *Game) EffectAt(r, c int) (Effect, bool) {
	for _, e := range g.effects {
		if e.Row == r && e.Col == c {
			return e, true
		}
	}
	return Effect{}, false
}
