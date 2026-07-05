//go:build !js

package main

// main.go — Ebiten 프론트엔드(데스크톱 전용, Phase 4 L2 부터). 여기 있는 모든 것은 (a) 입력 폴링을 Key 이벤트로
// 변환해 game.go 의 Input(k Key) 에 넘기거나, (b) 화면에 그리는 일만 한다.
// 게임 규칙/상태 전환은 전부 game.go 에 있다 — Phase 4 L2 의 TinyGo 웹 빌드가
// 같은 game.go 를 재사용하기 위한 경계다.

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

const (
	screenW  = 960
	screenH  = 600
	charW    = 14 // basicfont 7px * 2배
	lineH    = 28
	gameName = "VimQuest"
)

var face = text.NewGoXFace(basicfont.Face7x13)

var (
	colBG     = color.RGBA{0x1e, 0x20, 0x2a, 0xff}
	colFloor  = color.RGBA{0x4a, 0x4f, 0x5e, 0xff}
	colKey    = color.RGBA{0xf4, 0xd0, 0x3f, 0xff}
	colKeyDim = color.RGBA{0x6a, 0x60, 0x30, 0xff}
	colPest   = color.RGBA{0xe5, 0x4b, 0x4b, 0xff}
	colExit   = color.RGBA{0x4f, 0xc3, 0x6b, 0xff}
	colCursor = color.RGBA{0x3a, 0xa0, 0xd0, 0xff}
	colIns    = color.RGBA{0xf4, 0xd0, 0x3f, 0xff}
	colVisual = color.RGBA{0x55, 0x4a, 0x80, 0xff}
	colText   = color.RGBA{0xe8, 0xe8, 0xe8, 0xff}
	colMuted  = color.RGBA{0x8a, 0x90, 0xa0, 0xff}
	colMatch  = color.RGBA{0x35, 0x55, 0x40, 0xff}
)

// inbuf 는 Ebiten 프론트엔드 전용 입력 스크래치 버퍼(game.go 의 Game 에는 없음).
var inbuf []rune

// ───────────────────────── 입력(폴링 → Key 변환) ─────────────────────────

func (g *Game) Update() error {
	if restartRequested {
		restartRequested = false
		g.loadLevel(0)
		return nil
	}
	if resetRequested {
		resetRequested = false
		g.loadLevel(g.levelIdx)
		return nil
	}
	if levelSelectRequested {
		levelSelectRequested = false
		g.enterLevelSelect()
		return nil
	}

	switch g.state {
	case stateLevelClear:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
			g.Input(SpecialKey("cr"))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.Input(RuneKey('r'))
		}
	case stateLevelSelect:
		if inpututil.IsKeyJustPressed(ebiten.KeyH) {
			g.Input(RuneKey('h'))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyL) {
			g.Input(RuneKey('l'))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyJ) {
			g.Input(RuneKey('j'))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyK) {
			g.Input(RuneKey('k'))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
			g.Input(SpecialKey("cr"))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.Input(SpecialKey("esc"))
		}
	case stateAllClear:
		// 정적 화면 — 입력 없음(기존과 동일)
	default: // statePlaying
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.Input(SpecialKey("esc"))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
			g.Input(SpecialKey("cr"))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			g.Input(SpecialKey("bs"))
		}
		ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.Input(SpecialKey("c-r"))
		}

		// 타이핑된 문자
		inbuf = ebiten.AppendInputChars(inbuf[:0])
		for _, r := range inbuf {
			if ctrl {
				continue // Ctrl 조합은 문자로 처리하지 않음
			}
			g.Input(RuneKey(r))
		}
	}

	g.Tick()
	return nil
}

// ───────────────────────── 렌더링 ─────────────────────────

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)
	switch g.state {
	case stateAllClear:
		g.drawAllClear(screen)
	case stateLevelClear:
		g.drawLevelClear(screen)
	case stateLevelSelect:
		g.drawLevelSelect(screen)
	default:
		g.drawPlaying(screen)
	}
}

func (g *Game) drawAllClear(screen *ebiten.Image) {
	drawChar(screen, "ALL CLEAR!", 360, 250, colExit)
	drawChar(screen, "W1-W4 19 levels complete.", 300, 290, colText)
	drawChar(screen, "press the Restart button to replay", 250, 330, colMuted)
}

func (g *Game) drawPlaying(screen *ebiten.Image) {
	// 상단 HUD
	hud := "level " + itoa(g.levelIdx+1) + "/" + itoa(len(levels))
	if g.state == stateDrill {
		hud = "DRILL   streak " + itoa(g.drillStreak)
	}
	if g.lv.Kind == "navigate" {
		hud += "   keys " + itoa(g.keysNeed-len(g.keyPos)) + "/" + itoa(g.keysNeed) +
			"   bugs " + itoa(g.pestsLeft())
	} else {
		hud += "   [EDIT]  transform LEFT to match RIGHT"
	}
	drawChar(screen, hud, 60, 50, colMuted)

	originY := 130
	if g.lv.Kind == "navigate" {
		g.drawBuffer(screen, g.ed.Lines(), 60, originY, true, nil)
	} else {
		drawChar(screen, "CURRENT", 60, float64(originY-26), colText)
		drawChar(screen, "TARGET", 540, float64(originY-26), colExit)
		g.drawBuffer(screen, g.ed.Lines(), 60, originY, true, g.lv.Target)
		g.drawTarget(screen, g.lv.Target, 540, originY)
		// 가운데 구분선
		drawRect(screen, 510, float64(originY-10), 2, 300, colFloor)
	}

	// 하단 상태바: ex-command 입력 중이면 명령줄로 대체, 아니면 모드+명령+par
	var bar string
	if g.exMode {
		bar = ":" + string(g.exBuf)
	} else {
		bar = g.ed.ModeName()
		if g.ed.pendingStr != "" {
			bar += "   cmd: " + g.ed.pendingStr
		}
		bar += "   last: " + g.ed.lastKey
		bar += "   keys " + itoa(g.strokes) + " / par " + itoa(g.currentPar())
		if g.state == stateDrill {
			bar += "   total " + itoa(g.drillTotalKeys) + "/" + itoa(g.drillTotalPar)
		}
	}

	// visual bell: 막힌 키 입력 시 1~2프레임 상태바를 반전(실제 Vim의 visualbell 어법)
	barY := screenH - 46
	if g.bellTTL > 0 {
		drawRect(screen, 40, float64(barY-6), screenW-80, 32, colText)
		drawChar(screen, bar, 60, float64(barY), colBG)
	} else {
		drawChar(screen, bar, 60, float64(barY), colText)
	}
}

// drawLevelClear 는 레벨 클리어 요약 화면을 렌더한다.
func (g *Game) drawLevelClear(screen *ebiten.Image) {
	drawChar(screen, "LEVEL "+g.lv.ID+" CLEAR!", 340, 220, colExit)
	drawChar(screen, "your keys : "+itoa(g.clearStrokes), 340, 260, colText)

	starStr := strings.Repeat("*", g.clearStars) + strings.Repeat("-", 3-g.clearStars)
	drawChar(screen, "par       : "+itoa(g.clearPar)+"   "+starStr, 340, 290, colText)

	bestLine := "best      : " + itoa(g.clearBest)
	if g.clearBest == 0 || g.clearStrokes < g.clearBest {
		bestLine += " -> " + itoa(g.clearStrokes) + " (NEW!)"
	}
	drawChar(screen, bestLine, 340, 320, colMuted)

	if g.clearStars == 3 {
		drawChar(screen, "solution  : "+g.lv.Solution, 340, 350, colKey)
	}
	drawChar(screen, "[Enter] next   [r] retry", 340, 390, colMuted)
}

// drawLevelSelect 는 월드×레벨 그리드를 렌더한다. h/l = 월드 이동, j/k = 레벨 이동.
func (g *Game) drawLevelSelect(screen *ebiten.Image) {
	drawChar(screen, "SELECT LEVEL", 60, 50, colText)
	drawChar(screen, "h/l world   j/k level   Enter play   Esc back", 60, 80, colMuted)

	worlds := worldGroups()
	const colW = 220
	for wi, group := range worlds {
		ox := 60 + wi*colW
		drawChar(screen, "W"+itoa(wi+1), float64(ox), 130, colExit)
		for li, idx := range group {
			lv := levels[idx]
			oy := 130 + 40 + li*36
			prog := g.progress[lv.ID]

			label := lv.ID
			col := colMuted
			if prog.Unlocked {
				col = colText
				label += " " + strings.Repeat("*", prog.Stars) + strings.Repeat("-", 3-prog.Stars)
			} else {
				label += " LOCK"
			}
			if wi == g.selRow && li == g.selCol {
				drawRect(screen, float64(ox-4), float64(oy-2), float64(colW-24), 24, colVisual)
			}
			drawChar(screen, label, float64(ox), float64(oy), col)
		}
	}
}

func (g *Game) drawBuffer(screen *ebiten.Image, lines []string, ox, oy int, withCursor bool, target []string) {
	vr1, vc1, vr2, vc2, vline, hasVis := g.ed.VisualSpan()
	for r, line := range lines {
		// edit: 목표 줄과 일치하면 배경을 초록빛으로
		if target != nil && r < len(target) && line == target[r] {
			w := len(line)
			if w < 1 {
				w = 1
			}
			drawRect(screen, float64(ox-2), float64(oy+r*lineH-2), float64(w*charW+4), lineH-2, colMatch)
		}
		runes := []rune(line)
		for c, ch := range runes {
			px := float64(ox + c*charW)
			py := float64(oy + r*lineH)

			// 비주얼 선택 하이라이트
			if hasVis && inVisual(r, c, vr1, vc1, vr2, vc2, vline) {
				drawRect(screen, px-1, py-2, charW, lineH-4, colVisual)
			}
			// 커서
			if withCursor && r == g.ed.row && c == g.ed.col {
				cc := colCursor
				if g.ed.mode == ModeInsert {
					cc = colIns
				}
				drawRect(screen, px-1, py-2, charW, lineH-4, cc)
			}

			// 터미널식 피드백(2.2): 문자 치환/반전 — 실제 버퍼는 건드리지 않고 겹쳐 그린다
			displayCh := ch
			displayCol := cellColor(ch, g, r, c)
			if eff, ok := g.effectAt(r, c); ok {
				if eff.Invert {
					drawRect(screen, px-1, py-2, charW, lineH-4, colText)
					displayCol = colBG
				}
				if eff.Glyph != 0 {
					displayCh = eff.Glyph
					displayCol = colPest // 처치 연출은 버그 색(빨강)으로 뚜렷하게
				}
			}

			if displayCh == ' ' {
				continue
			}
			drawChar(screen, string(displayCh), px, py, displayCol)
		}
		// 빈 줄에서 커서/매치 표시
		if withCursor && r == g.ed.row && g.ed.col >= len(runes) {
			px := float64(ox + len(runes)*charW)
			py := float64(oy + r*lineH)
			cc := colCursor
			if g.ed.mode == ModeInsert {
				cc = colIns
			}
			drawRect(screen, px-1, py-2, charW, lineH-4, cc)
		}
	}
}

func (g *Game) drawTarget(screen *ebiten.Image, lines []string, ox, oy int) {
	for r, line := range lines {
		for c, ch := range []rune(line) {
			if ch == ' ' {
				continue
			}
			drawChar(screen, string(ch), float64(ox+c*charW), float64(oy+r*lineH), colExit)
		}
	}
}

func inVisual(r, c, r1, c1, r2, c2 int, lineMode bool) bool {
	if r < r1 || r > r2 {
		return false
	}
	if lineMode {
		return true
	}
	if r1 == r2 {
		return c >= c1 && c <= c2
	}
	if r == r1 {
		return c >= c1
	}
	if r == r2 {
		return c <= c2
	}
	return true
}

func cellColor(ch rune, g *Game, r, c int) color.Color {
	if g.lv.Kind == "navigate" {
		switch ch {
		case 'K':
			if g.keyPos[[2]int{r, c}] {
				return colKey
			}
			return colKeyDim
		case '*':
			return colPest
		case '$':
			return colExit
		case '.':
			return colFloor
		}
	}
	return colText
}

func drawChar(screen *ebiten.Image, s string, x, y float64, col color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(col)
	text.Draw(screen, s, face, op)
}

func drawRect(screen *ebiten.Image, x, y, w, h float64, col color.Color) {
	img := ebiten.NewImage(int(w), int(h))
	img.Fill(col)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}

func (g *Game) Layout(int, int) (int, int) { return screenW, screenH }

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle(gameName)
	registerJSHooks()
	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}
