package main

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

// 렌더링 상수 (보드는 2배 스케일 모노스페이스로 그린다)
const (
	screenW  = 960
	screenH  = 600
	originX  = 60  // 보드 시작 X
	originY  = 160 // 보드 시작 Y
	charW    = 14  // basicfont 7px * 2배
	lineH    = 28
	gameName = "VimQuest"
)

var face = text.NewGoXFace(basicfont.Face7x13)

// 색상 팔레트
var (
	colBG     = color.RGBA{0x1e, 0x20, 0x2a, 0xff}
	colFloor  = color.RGBA{0x4a, 0x4f, 0x5e, 0xff}
	colKey    = color.RGBA{0xf4, 0xd0, 0x3f, 0xff} // 노랑
	colPest   = color.RGBA{0xe5, 0x4b, 0x4b, 0xff} // 빨강
	colExit   = color.RGBA{0x4f, 0xc3, 0x6b, 0xff} // 초록
	colCursor = color.RGBA{0x3a, 0xa0, 0xd0, 0xff} // 청록 커서
	colText   = color.RGBA{0xe8, 0xe8, 0xe8, 0xff}
)

// Game 은 Ebiten 게임 상태 전체를 담는다.
type Game struct {
	levelIdx  int
	buf       [][]rune // 현재 버퍼(가변) — x 로 글자를 지우면 반영됨
	cx, cy    int      // 커서 위치 (col, row)
	keys      int      // 획득한 열쇠 수
	keysNeed  int      // 필요한 열쇠 수
	pests     int      // 남은 버그 수
	lastInput string   // 입력 에코
	finished  bool     // 모든 레벨 클리어
	dirty     bool     // DOM 갱신 필요 플래그
}

func NewGame() *Game {
	g := &Game{}
	g.loadLevel(0)
	return g
}

// loadLevel 은 레벨 데이터를 가변 버퍼로 읽어들이고 커서/카운트를 초기화한다.
func (g *Game) loadLevel(idx int) {
	g.levelIdx = idx
	lv := levels[idx]
	g.buf = make([][]rune, len(lv.Map))
	g.keys, g.keysNeed, g.pests = 0, 0, 0
	for r, line := range lv.Map {
		g.buf[r] = []rune(line)
		for c, ch := range g.buf[r] {
			switch ch {
			case '@':
				g.cx, g.cy = c, r
				g.buf[r][c] = '.' // 시작 표시는 바닥으로
			case 'K':
				g.keysNeed++
			case '*':
				g.pests++
			}
		}
	}
	g.lastInput = ""
	g.dirty = true
}

// --- 좌표/이동 헬퍼 ---

func (g *Game) lineLen(row int) int { return len(g.buf[row]) }

func (g *Game) cellAt(row, col int) rune {
	if row < 0 || row >= len(g.buf) || col < 0 || col >= len(g.buf[row]) {
		return ' '
	}
	return g.buf[row][col]
}

func isSpace(r rune) bool { return r == ' ' }

// clampCol 은 행 길이에 맞춰 열을 보정한다 (빈 줄 대비).
func (g *Game) clampCol() {
	max := g.lineLen(g.cy) - 1
	if max < 0 {
		max = 0
	}
	if g.cx > max {
		g.cx = max
	}
	if g.cx < 0 {
		g.cx = 0
	}
}

// wordForward 는 Vim 의 w 모션 — 다음 단어 시작으로 이동(줄 넘어감).
func (g *Game) wordForward() {
	r, c := g.cy, g.cx
	line := g.buf[r]
	// 현재 단어(비공백) 건너뛰기
	for c < len(line) && !isSpace(line[c]) {
		c++
	}
	for {
		for c < len(line) && isSpace(line[c]) {
			c++
		}
		if c < len(line) {
			break // 단어 시작 찾음
		}
		// 다음 줄로
		if r+1 >= len(g.buf) {
			c = len(line) - 1
			if c < 0 {
				c = 0
			}
			break
		}
		r++
		line = g.buf[r]
		c = 0
		if len(line) > 0 && !isSpace(line[0]) {
			break
		}
	}
	g.cy, g.cx = r, c
	g.clampCol()
}

// wordBackward 는 Vim 의 b 모션 — 이전 단어 시작으로.
func (g *Game) wordBackward() {
	r, c := g.cy, g.cx
	c--
	for {
		if c < 0 {
			if r == 0 {
				c = 0
				break
			}
			r--
			c = len(g.buf[r]) - 1
			if c < 0 {
				c = 0
				continue
			}
		}
		if c >= 0 && c < len(g.buf[r]) && !isSpace(g.buf[r][c]) {
			// 단어 시작까지 더 뒤로
			for c > 0 && !isSpace(g.buf[r][c-1]) {
				c--
			}
			break
		}
		c--
	}
	g.cy, g.cx = r, c
	g.clampCol()
}

// wordEnd 는 Vim 의 e 모션 — 현재/다음 단어 끝으로.
func (g *Game) wordEnd() {
	r, c := g.cy, g.cx
	c++
	for {
		// 공백 건너뛰기
		for c < len(g.buf[r]) && isSpace(g.buf[r][c]) {
			c++
		}
		if c >= len(g.buf[r]) {
			if r+1 >= len(g.buf) {
				c = len(g.buf[r]) - 1
				break
			}
			r++
			c = 0
			continue
		}
		// 단어 끝까지
		for c+1 < len(g.buf[r]) && !isSpace(g.buf[r][c+1]) {
			c++
		}
		break
	}
	if c < 0 {
		c = 0
	}
	g.cy, g.cx = r, c
	g.clampCol()
}

// move 는 hjkl 한 칸 이동 (버퍼 경계로 제한).
func (g *Game) move(dx, dy int) {
	nx, ny := g.cx+dx, g.cy+dy
	if ny < 0 || ny >= len(g.buf) {
		return
	}
	g.cy = ny
	g.cx = nx
	g.clampCol()
}

// onEnter 는 커서가 새 칸에 들어왔을 때 열쇠 획득 등을 처리한다.
func (g *Game) onEnter() {
	switch g.cellAt(g.cy, g.cx) {
	case 'K':
		g.buf[g.cy][g.cx] = '.'
		g.keys++
		g.dirty = true
	case '$':
		if g.keys >= g.keysNeed && g.pests == 0 {
			g.advance()
		}
	}
}

// deleteUnder 는 Vim 의 x — 커서 아래 글자 삭제(버그 제거).
func (g *Game) deleteUnder() {
	if g.cellAt(g.cy, g.cx) == '*' {
		g.buf[g.cy][g.cx] = '.'
		g.pests--
		g.dirty = true
	}
}

// advance 는 다음 레벨로(없으면 종료).
func (g *Game) advance() {
	if g.levelIdx+1 < len(levels) {
		g.loadLevel(g.levelIdx + 1)
	} else {
		g.finished = true
		g.dirty = true
	}
}

// --- Ebiten 인터페이스 ---

func (g *Game) Update() error {
	if g.finished {
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.finished = false
			g.loadLevel(0)
		}
		return nil
	}

	moved := false
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyH):
		g.move(-1, 0)
		g.lastInput, moved = "h", true
	case inpututil.IsKeyJustPressed(ebiten.KeyL):
		g.move(1, 0)
		g.lastInput, moved = "l", true
	case inpututil.IsKeyJustPressed(ebiten.KeyJ):
		g.move(0, 1)
		g.lastInput, moved = "j", true
	case inpututil.IsKeyJustPressed(ebiten.KeyK):
		g.move(0, -1)
		g.lastInput, moved = "k", true
	case inpututil.IsKeyJustPressed(ebiten.KeyW):
		g.wordForward()
		g.lastInput, moved = "w", true
	case inpututil.IsKeyJustPressed(ebiten.KeyB):
		g.wordBackward()
		g.lastInput, moved = "b", true
	case inpututil.IsKeyJustPressed(ebiten.KeyE):
		g.wordEnd()
		g.lastInput, moved = "e", true
	case inpututil.IsKeyJustPressed(ebiten.KeyX):
		g.deleteUnder()
		g.lastInput = "x"
		g.dirty = true
	case inpputJust(ebiten.KeyR):
		g.loadLevel(g.levelIdx) // 현재 레벨 리셋
		g.lastInput = "r"
	}

	if moved {
		g.onEnter()
		g.dirty = true
	}

	if g.dirty {
		g.syncDOM()
		g.dirty = false
	}
	return nil
}

// inpputJust 는 가독성용 래퍼.
func inpputJust(k ebiten.Key) bool { return inpututil.IsKeyJustPressed(k) }

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)

	// 보드 그리기 (글자별 색상 + 커서 하이라이트)
	for r := 0; r < len(g.buf); r++ {
		for c := 0; c < len(g.buf[r]); c++ {
			ch := g.buf[r][c]
			px := float64(originX + c*charW)
			py := float64(originY + r*lineH)

			if c == g.cx && r == g.cy {
				drawRect(screen, px-1, py-2, charW, lineH-4, colCursor)
			}

			var col color.Color = colFloor
			switch ch {
			case 'K':
				col = colKey
			case '*':
				col = colPest
			case '$':
				col = colExit
			case '.':
				col = colFloor
			default:
				col = colText
			}
			if ch == ' ' {
				continue
			}
			drawChar(screen, string(ch), px, py, col)
		}
	}

	// 상단/하단 ASCII HUD (한국어는 DOM 에서 표시)
	hud := "level " + itoa(g.levelIdx+1) + "/" + itoa(len(levels))
	hud += "   keys " + itoa(g.keys) + "/" + itoa(g.keysNeed)
	hud += "   bugs " + itoa(g.pests)
	drawChar(screen, hud, 60, 60, colText)

	status := "-- NORMAL --   last: " + g.lastInput
	drawChar(screen, status, 60, screenH-50, colText)

	if g.finished {
		drawRect(screen, 0, 0, screenW, screenH, color.RGBA{0, 0, 0, 0xcc})
		drawChar(screen, "ALL CLEAR!  press R to replay", 260, 300, colExit)
	}
}

func (g *Game) Layout(int, int) (int, int) { return screenW, screenH }

// --- 그리기 헬퍼 ---

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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b strings.Builder
	var digits []byte
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	if neg {
		b.WriteByte('-')
	}
	for i := len(digits) - 1; i >= 0; i-- {
		b.WriteByte(digits[i])
	}
	return b.String()
}

// syncDOM 은 한국어 상태(제목/힌트/진행/클리어)를 HTML 로 밀어넣는다.
func (g *Game) syncDOM() {
	if g.finished {
		domSet("level-title", "🎉 전부 클리어!")
		domSet("hint", "축하합니다! W1 이동의 숲을 통과했어요. R 키로 다시 플레이할 수 있어요.")
		domSet("status", "")
		return
	}
	lv := levels[g.levelIdx]
	domSet("level-title", lv.Title)
	domSet("hint", lv.Hint)
	status := "열쇠 " + itoa(g.keys) + "/" + itoa(g.keysNeed)
	if g.pests > 0 {
		status += "   ·   버그 " + itoa(g.pests) + "마리 남음"
	} else if g.keysNeed > 0 && g.keys >= g.keysNeed {
		status += "   ·   이제 $(출구)로 가세요!"
	}
	domSet("status", status)
}

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle(gameName)
	g := NewGame()
	g.syncDOM()
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
}
