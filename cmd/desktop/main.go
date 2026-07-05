// VimQuest 데스크톱 프론트엔드(Ebiten). 여기 있는 모든 것은 (a) 입력 폴링을
// engine.Key 로 변환해 game.Input 에 넘기거나(이 파일), (b) 화면에 그리는
// 일(render.go)만 한다. 게임 규칙/상태 전환은 전부 internal/game 에 있다 —
// 웹 빌드(cmd/web, TinyGo)가 같은 game 패키지를 재사용하기 위한 경계다.
package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"vimquest/internal/engine"
	"vimquest/internal/game"
)

const (
	screenW  = 960
	screenH  = 600
	gameName = "VimQuest"
)

// app 은 *game.Game 을 ebiten.Game 인터페이스에 맞춘 어댑터.
type app struct {
	g     *game.Game
	inbuf []rune // 입력 폴링 스크래치 버퍼(프레임마다 재사용)
}

// Update 는 상태를 보지 않고 키를 전부 Game.Input 으로 나른다. 어떤 키가
// 어느 상태에서 유효한지는 Game.Input 이 단독으로 결정한다(F2) — 예전엔
// 여기서도 상태별 라우팅 표를 별도로 들고 있어서, 새 상태(StateAllClear)에
// 대응하는 케이스를 빠뜨리는 사고(소프트락)가 났다. 라우팅 표를 하나로
// 줄이면 그 사고 자체가 재발할 수 없다.
func (a *app) Update() error {
	g := a.g

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.Input(engine.SpecialKey("esc"))
	}
	if isEnterPressed() {
		g.Input(engine.SpecialKey("cr"))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.Input(engine.SpecialKey("bs"))
	}
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.Input(engine.SpecialKey("c-r"))
	}

	// 타이핑된 문자 — StateLevelClear 의 'r'(재시도), StateLevelSelect 의
	// hjkl 이동을 포함해 전부 여기서 나른다(상태별 특별 취급 없음).
	a.inbuf = ebiten.AppendInputChars(a.inbuf[:0])
	for _, r := range a.inbuf {
		if ctrl {
			continue // Ctrl 조합은 문자로 처리하지 않음
		}
		g.Input(engine.RuneKey(r))
	}

	g.Tick()
	return nil
}

func isEnterPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)
}

func (a *app) Layout(int, int) (int, int) { return screenW, screenH }

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle(gameName)
	if err := ebiten.RunGame(&app{g: game.New()}); err != nil {
		panic(err)
	}
}
