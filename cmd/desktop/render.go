package main

// render.go — Ebiten 렌더러. internal/game 의 읽기 전용 뷰(view.go)만 보고
// 그린다. 웹 렌더러(web/renderer.js)와 기능 동등을 유지할 것 — 둘 다 게임
// 규칙을 모르고, 상태를 바꾸는 코드가 여기 들어오면 안 된다.

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"

	"vimquest/internal/engine"
	"vimquest/internal/game"
)

const (
	charW = 14 // basicfont 7px * 2배
	lineH = 28
)

var face = text.NewGoXFace(basicfont.Face7x13)

// whitePixel 은 drawRect 가 재사용하는 1×1 흰색 이미지 — 사각형 하나(커서·
// 비주얼 하이라이트·매치 배경 등, 프레임당 수십 회)를 그릴 때마다 새 GPU
// 텍스처를 만드는 대신, 이 이미지 하나를 GeoM.Scale + ColorScale 로 원하는
// 크기·색으로 찍어내 텍스처 할당을 0으로 만든다.
var whitePixel = newWhitePixel()

func newWhitePixel() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return img
}

// 이 팔레트의 canonical 출처는 web/src/renderer.js 의 COL(+ index.html 의
// CSS 변수, 3벌째) — 값을 바꿀 땐 세 곳 모두 손으로 맞춰야 한다(코드 생성은 과함).
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

func (a *app) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)
	switch a.g.State() {
	case game.StateAllClear:
		a.drawAllClear(screen)
	case game.StateLevelClear:
		a.drawLevelClear(screen)
	case game.StateLevelSelect:
		a.drawLevelSelect(screen)
	default: // StatePlaying / StateDrill
		a.drawPlaying(screen)
	}
}

func (a *app) drawAllClear(screen *ebiten.Image) {
	drawChar(screen, "ALL CLEAR!", 360, 250, colExit)
	drawChar(screen, "W1-W"+strconv.Itoa(len(game.WorldGroups()))+" "+strconv.Itoa(game.LevelCount())+" levels complete.", 300, 290, colText)
	drawChar(screen, "[Enter] level select", 340, 330, colMuted)
}

func (a *app) drawPlaying(screen *ebiten.Image) {
	g := a.g
	lv := g.Level()
	ed := g.Editor()

	// 상단 HUD
	hud := "level " + strconv.Itoa(g.LevelIndex()+1) + "/" + strconv.Itoa(game.LevelCount())
	if g.State() == game.StateDrill {
		// lv.Title 이 생성기별 유형을 이미 담고 있다("DRILL"/"DRILL [w]"/...).
		hud = lv.Title + "   streak " + strconv.Itoa(g.DrillStreak())
	}
	if lv.Kind == "navigate" {
		hud += "   keys " + strconv.Itoa(g.KeysNeed()-g.KeysLeft()) + "/" + strconv.Itoa(g.KeysNeed()) +
			"   bugs " + strconv.Itoa(g.PestsLeft())
	} else {
		hud += "   [EDIT]  transform LEFT to match RIGHT"
	}
	drawChar(screen, hud, 60, 50, colMuted)

	originY := 130
	if lv.Kind == "navigate" {
		a.drawBuffer(screen, ed.Lines(), 60, originY, nil)
	} else {
		drawChar(screen, "CURRENT", 60, float64(originY-26), colText)
		drawChar(screen, "TARGET", 540, float64(originY-26), colExit)
		a.drawBuffer(screen, ed.Lines(), 60, originY, lv.Target)
		a.drawTarget(screen, lv.Target, 540, originY)
		// 가운데 구분선
		drawRect(screen, 510, float64(originY-10), 2, 300, colFloor)
	}

	// 하단 상태바: ex-command 입력 중이면 명령줄로 대체, 아니면 모드+명령+par
	var bar string
	if ex, active := g.ExLine(); active {
		bar = ":" + ex
	} else {
		bar = ed.ModeName()
		if ed.PendingString() != "" {
			bar += "   cmd: " + ed.PendingString()
		}
		bar += "   last: " + ed.LastKey()
		bar += "   keys " + strconv.Itoa(g.Strokes()) + " / par " + strconv.Itoa(g.Par())
		if g.State() == game.StateDrill {
			keys, par := g.DrillTotals()
			bar += "   total " + strconv.Itoa(keys) + "/" + strconv.Itoa(par)
		}
	}

	// visual bell: 막힌 키 입력 시 1~2프레임 상태바를 반전(실제 Vim의 visualbell 어법)
	barY := screenH - 46
	if g.BellActive() {
		drawRect(screen, 40, float64(barY-6), screenW-80, 32, colText)
		drawChar(screen, bar, 60, float64(barY), colBG)
	} else {
		drawChar(screen, bar, 60, float64(barY), colText)
	}
}

// drawLevelClear 는 레벨 클리어 요약 화면을 렌더한다.
func (a *app) drawLevelClear(screen *ebiten.Image) {
	g := a.g
	cs := g.LastClear()
	drawChar(screen, "LEVEL "+g.Level().ID+" CLEAR!", 340, 220, colExit)
	drawChar(screen, "your keys : "+strconv.Itoa(cs.Strokes), 340, 260, colText)

	starStr := strings.Repeat("*", cs.Stars) + strings.Repeat("-", 3-cs.Stars)
	drawChar(screen, "par       : "+strconv.Itoa(cs.Par)+"   "+starStr, 340, 290, colText)

	bestLine := "best      : " + strconv.Itoa(cs.Best)
	if cs.IsNew {
		bestLine += " -> " + strconv.Itoa(cs.Strokes) + " (NEW!)"
	}
	drawChar(screen, bestLine, 340, 320, colMuted)

	// 내가 실제로 입력한 키 시퀀스 — 별점과 무관하게 항상 표시(제작자 solution
	// 과 달리 스포일러가 아니다).
	drawChar(screen, "yours     : "+cs.Yours, 340, 350, colText)
	if cs.Stars == 3 {
		drawChar(screen, "solution  : "+g.Level().Solution, 340, 380, colKey)
	}
	drawChar(screen, "[Enter] next   [r] retry", 340, 420, colMuted)
}

// drawLevelSelect 는 월드×레벨 그리드를 렌더한다. h/l = 월드 이동, j/k = 레벨 이동.
func (a *app) drawLevelSelect(screen *ebiten.Image) {
	drawChar(screen, "SELECT LEVEL", 60, 50, colText)
	drawChar(screen, "h/l world   j/k level   Enter play   Esc back", 60, 80, colMuted)

	selRow, selCol := a.g.Selection()
	const colW = 220
	for wi, group := range game.WorldGroups() {
		ox := 60 + wi*colW
		drawChar(screen, "W"+strconv.Itoa(wi+1), float64(ox), 130, colExit)
		for li, idx := range group {
			lv := game.LevelAt(idx)
			oy := 130 + 40 + li*36
			prog := a.g.ProgressFor(lv.ID)

			label := lv.ID
			col := colMuted
			if prog.Unlocked {
				col = colText
				label += " " + strings.Repeat("*", prog.Stars) + strings.Repeat("-", 3-prog.Stars)
			} else {
				label += " LOCK"
			}
			if wi == selRow && li == selCol {
				drawRect(screen, float64(ox-4), float64(oy-2), float64(colW-24), 24, colVisual)
			}
			drawChar(screen, label, float64(ox), float64(oy), col)
		}
	}
}

func (a *app) drawBuffer(screen *ebiten.Image, lines []string, ox, oy int, target []string) {
	g := a.g
	ed := g.Editor()
	insert := ed.Mode() == engine.ModeInsert

	visByRow := make(map[int]game.RowSpan)
	for _, vr := range g.VisualRows() {
		visByRow[vr.Row] = vr
	}
	matched := g.MatchedRows()

	for r, line := range lines {
		// edit: 목표 줄과 일치하면 배경을 초록빛으로
		if target != nil && r < len(matched) && matched[r] {
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
			if vr, hasVis := visByRow[r]; hasVis && c >= vr.C1 && c <= vr.C2 {
				drawRect(screen, px-1, py-2, charW, lineH-4, colVisual)
			}
			// 커서
			if r == ed.Row() && c == ed.Col() {
				cc := colCursor
				if insert {
					cc = colIns
				}
				drawRect(screen, px-1, py-2, charW, lineH-4, cc)
			}

			// 터미널식 피드백: 문자 치환/반전 — 실제 버퍼는 건드리지 않고 겹쳐 그린다
			displayCh := ch
			displayCol := a.cellColor(ch, r, c)
			if eff, ok := g.EffectAt(r, c); ok {
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
		// 빈 줄에서 커서 표시
		if r == ed.Row() && ed.Col() >= len(runes) {
			px := float64(ox + len(runes)*charW)
			py := float64(oy + r*lineH)
			cc := colCursor
			if insert {
				cc = colIns
			}
			drawRect(screen, px-1, py-2, charW, lineH-4, cc)
		}
	}
}

func (a *app) drawTarget(screen *ebiten.Image, lines []string, ox, oy int) {
	for r, line := range lines {
		for c, ch := range []rune(line) {
			if ch == ' ' {
				continue
			}
			drawChar(screen, string(ch), float64(ox+c*charW), float64(oy+r*lineH), colExit)
		}
	}
}

func (a *app) cellColor(ch rune, r, c int) color.Color {
	if a.g.Level().Kind == "navigate" {
		switch ch {
		case 'K':
			if a.g.HasKeyAt(r, c) {
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
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(col)
	screen.DrawImage(whitePixel, op)
}
