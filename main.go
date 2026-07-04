package main

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

// 리셋/레벨선택 요청은 JS 버튼(또는 데스크톱 no-op)에서 설정한다.
var resetRequested bool
var restartRequested bool
var levelSelectRequested bool

func requestReset()       { resetRequested = true }
func requestRestart()     { restartRequested = true }
func requestLevelSelect() { levelSelectRequested = true }

// gameState 는 게임 전체의 화면 상태를 나타낸다.
type gameState int

const (
	statePlaying     gameState = iota // 레벨 플레이 중
	stateLevelClear                   // 레벨 클리어 요약 화면
	stateLevelSelect                  // 레벨 선택 화면
	stateAllClear                     // 전체 클리어
)

type Game struct {
	levelIdx int
	lv       Level
	ed       *Editor

	// navigate 상태
	keyPos   map[[2]int]bool // 아직 안 주운 열쇠 위치
	keysNeed int

	state gameState
	inbuf []rune

	strokes int // 이번 레벨에서 누른 키 수(막힌 키 포함)

	// 레벨 클리어 화면에 표시할 스냅샷(advance() 시점에 고정)
	clearStrokes int
	clearPar     int
	clearStars   int
	clearBest    int // 갱신 전 best (0 이면 이전 클리어 기록 없음)

	// 레벨 선택 화면 커서
	selRow, selCol int

	store    ProgressStore
	progress map[string]LevelProgress
}

func NewGame() *Game {
	g := &Game{}
	g.store = newProgressStore()
	g.progress = g.store.Load()
	if len(levels) > 0 {
		first := g.progress[levels[0].ID]
		first.Unlocked = true
		g.progress[levels[0].ID] = first
	}
	g.loadLevel(0)
	return g
}

func (g *Game) loadLevel(idx int) {
	g.levelIdx = idx
	g.lv = levels[idx]
	g.keyPos = map[[2]int]bool{}
	g.keysNeed = 0
	g.strokes = 0
	g.state = statePlaying

	if g.lv.Kind == "navigate" {
		lines := make([]string, len(g.lv.Map))
		sr, sc := 0, 0
		for r, row := range g.lv.Map {
			b := []rune(row)
			for c, ch := range b {
				switch ch {
				case '@':
					sr, sc = r, c
					b[c] = '.'
				case 'K':
					g.keyPos[[2]int{r, c}] = true
					g.keysNeed++
				}
			}
			lines[r] = string(b)
		}
		g.ed = NewEditor(lines)
		g.ed.row, g.ed.col = sr, sc
	} else {
		g.ed = NewEditor(append([]string(nil), g.lv.Map...))
	}
	g.syncDOM()
}

func (g *Game) pestsLeft() int {
	n := 0
	for _, l := range g.ed.Lines() {
		n += strings.Count(l, "*")
	}
	return n
}

// navigateAllows 는 navigate 레벨에서 편집 명령을 막고 이동+x 만 허용한다.
func navigateAllows(e *Editor, k Key) bool {
	if e.await != "" || e.op != 0 || e.pendObj != 0 || e.mode != ModeNormal {
		return true // 명령 진행 중(찾기 대상 등)
	}
	if k.R == 0 {
		return true // esc 등 특수키
	}
	if k.R >= '1' && k.R <= '9' {
		return true
	}
	switch k.R {
	case 'h', 'j', 'k', 'l', 'w', 'b', 'e', 'W', 'B', 'E',
		'0', '^', '$', 'f', 'F', 't', 'T', ';', ',', 'g', 'G', 'x':
		return true
	}
	return false
}

func (g *Game) feed(k Key) {
	g.strokes++
	if g.lv.Kind == "navigate" && !navigateAllows(g.ed, k) {
		return
	}
	g.ed.Feed(k)
}

// currentPar 는 현재 레벨의 검증된 Solution 기준 타수(par)를 반환한다.
func (g *Game) currentPar() int {
	return len(parseKeys(g.lv.Solution))
}

// stars 는 par 대비 타수로 별점(1~3)을 계산한다.
func stars(strokes, par int) int {
	if par <= 0 {
		return 1
	}
	switch {
	case strokes <= par:
		return 3
	case float64(strokes) <= float64(par)*1.5:
		return 2
	default:
		return 1
	}
}

// recordClear 는 현재 레벨의 클리어 기록을 진행 상황에 반영하고 저장한다.
// 갱신 전 best(BestStrokes)를 반환한다(0 이면 최초 클리어).
func (g *Game) recordClear() int {
	prog := g.progress[g.lv.ID]
	prevBest := prog.BestStrokes
	if prevBest == 0 || g.strokes < prevBest {
		prog.BestStrokes = g.strokes
	}
	if s := stars(g.strokes, g.currentPar()); s > prog.Stars {
		prog.Stars = s
	}
	prog.Unlocked = true
	g.progress[g.lv.ID] = prog

	if g.levelIdx+1 < len(levels) {
		next := levels[g.levelIdx+1]
		np := g.progress[next.ID]
		np.Unlocked = true
		g.progress[next.ID] = np
	}
	g.store.Save(g.progress)
	return prevBest
}

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
		return g.updateLevelClear()
	case stateLevelSelect:
		return g.updateLevelSelect()
	case stateAllClear:
		return nil
	default:
		return g.updatePlaying()
	}
}

func (g *Game) updatePlaying() error {
	// 특수키
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.feed(SpecialKey("esc"))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		g.feed(SpecialKey("cr"))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.feed(SpecialKey("bs"))
	}
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.feed(SpecialKey("c-r"))
	}

	// 타이핑된 문자
	g.inbuf = ebiten.AppendInputChars(g.inbuf[:0])
	for _, r := range g.inbuf {
		if ctrl {
			continue // Ctrl 조합은 문자로 처리하지 않음
		}
		g.feed(RuneKey(r))
	}

	g.checkWin()
	g.syncDOM()
	return nil
}

func (g *Game) updateLevelClear() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		g.loadLevel(g.levelIdx + 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.loadLevel(g.levelIdx)
	}
	return nil
}

// enterLevelSelect 는 레벨 선택 화면으로 전환하며 커서를 현재 레벨 위치로 맞춘다.
func (g *Game) enterLevelSelect() {
	g.state = stateLevelSelect
	g.selRow, g.selCol = 0, 0
	worlds := worldGroups()
	for wi, group := range worlds {
		for li, idx := range group {
			if idx == g.levelIdx {
				g.selRow, g.selCol = wi, li
			}
		}
	}
	g.syncDOM()
}

func (g *Game) updateLevelSelect() error {
	worlds := worldGroups()
	if len(worlds) == 0 {
		return nil
	}
	if g.selRow >= len(worlds) {
		g.selRow = len(worlds) - 1
	}
	if g.selCol >= len(worlds[g.selRow]) {
		g.selCol = len(worlds[g.selRow]) - 1
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyH) && g.selRow > 0 {
		g.selRow--
		if g.selCol >= len(worlds[g.selRow]) {
			g.selCol = len(worlds[g.selRow]) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) && g.selRow < len(worlds)-1 {
		g.selRow++
		if g.selCol >= len(worlds[g.selRow]) {
			g.selCol = len(worlds[g.selRow]) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyJ) && g.selCol < len(worlds[g.selRow])-1 {
		g.selCol++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyK) && g.selCol > 0 {
		g.selCol--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		idx := worlds[g.selRow][g.selCol]
		if g.progress[levels[idx].ID].Unlocked {
			g.loadLevel(idx)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = statePlaying
		g.syncDOM()
	}
	return nil
}

// worldGroups 는 levels 를 Level.ID 접두어(월드 번호) 기준으로 묶어
// [월드][월드 내 레벨] = levels 인덱스 형태로 반환한다.
func worldGroups() [][]int {
	var groups [][]int
	var cur []int
	curWorld := ""
	for i, lv := range levels {
		world := lv.ID
		if idx := strings.IndexByte(lv.ID, '-'); idx >= 0 {
			world = lv.ID[:idx]
		}
		if world != curWorld {
			if len(cur) > 0 {
				groups = append(groups, cur)
			}
			cur = nil
			curWorld = world
		}
		cur = append(cur, i)
	}
	if len(cur) > 0 {
		groups = append(groups, cur)
	}
	return groups
}

func (g *Game) checkWin() {
	if g.lv.Kind == "navigate" {
		pos := [2]int{g.ed.row, g.ed.col}
		delete(g.keyPos, pos) // 열쇠 위로 오면 획득
		cell := g.cellAt(g.ed.row, g.ed.col)
		if len(g.keyPos) == 0 && g.pestsLeft() == 0 && cell == '$' {
			g.advance()
		}
		return
	}
	// edit: 목표와 완전 일치
	cur := g.ed.Lines()
	if len(cur) != len(g.lv.Target) {
		return
	}
	for i := range cur {
		if cur[i] != g.lv.Target[i] {
			return
		}
	}
	g.advance()
}

// advance 는 현재 레벨의 클리어 통계를 스냅샷하고 진행 상황을 저장한 뒤
// stateLevelClear(또는 마지막 레벨이면 stateAllClear)로 전환한다.
// 다음 레벨 로드는 updateLevelClear() 에서 플레이어가 Enter 를 누를 때 이뤄진다.
func (g *Game) advance() {
	g.clearStrokes = g.strokes
	g.clearPar = g.currentPar()
	g.clearStars = stars(g.strokes, g.clearPar)
	g.clearBest = g.recordClear()

	if g.levelIdx+1 < len(levels) {
		g.state = stateLevelClear
	} else {
		g.state = stateAllClear
	}
	g.syncDOM()
}

func (g *Game) cellAt(r, c int) rune {
	ls := g.ed.lines
	if r < 0 || r >= len(ls) || c < 0 || c >= len(ls[r]) {
		return ' '
	}
	return ls[r][c]
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

	// 하단 상태바: 모드 + 입력중 명령 + 마지막 키 + par 진행도
	bar := g.ed.ModeName()
	if g.ed.pendingStr != "" {
		bar += "   cmd: " + g.ed.pendingStr
	}
	bar += "   last: " + g.ed.lastKey
	bar += "   keys " + itoa(g.strokes) + " / par " + itoa(g.currentPar())
	drawChar(screen, bar, 60, screenH-46, colText)
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
			if ch == ' ' {
				continue
			}
			drawChar(screen, string(ch), px, py, cellColor(ch, g, r, c))
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i := len(digits) - 1; i >= 0; i-- {
		b.WriteByte(digits[i])
	}
	return b.String()
}

// ───────────────────────── DOM(한국어 UI) ─────────────────────────

// cmdsHTML 은 명령어 목록을 우측 패널용 HTML 로 만든다.
func cmdsHTML(cmds []Cmd) string {
	var b strings.Builder
	for _, c := range cmds {
		b.WriteString(`<div class="cmd"><span class="k">`)
		b.WriteString(htmlEscape(c.K))
		b.WriteString(`</span><span class="d">`)
		b.WriteString(htmlEscape(c.D))
		b.WriteString(`</span></div>`)
	}
	return b.String()
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}

func (g *Game) syncDOM() {
	switch g.state {
	case stateAllClear:
		domSet("level-title", "🎉 ALL CLEAR!")
		domSet("hint", "Congratulations! You cleared all 19 levels across W1-W4. Now go practice in real Vim!")
		domSet("status", "")
		domSetHTML("solve-cmds", "")
		return
	case stateLevelClear:
		domSet("level-title", "LEVEL "+g.lv.ID+" CLEAR!")
		domSet("hint", "Press Enter for the next level, or r to retry this one.")
		domSet("status", "keys "+itoa(g.clearStrokes)+" / par "+itoa(g.clearPar))
		domSetHTML("solve-cmds", "")
		return
	case stateLevelSelect:
		domSet("level-title", "SELECT LEVEL")
		domSet("hint", "h/l move between worlds, j/k move within a world, Enter to play, Esc to go back.")
		domSet("status", "")
		domSetHTML("solve-cmds", "")
		return
	}

	domSet("level-title", g.lv.Title)
	domSet("hint", g.lv.Hint)
	domSetHTML("solve-cmds", cmdsHTML(g.lv.Cmds))
	parInfo := "   ·   keys " + itoa(g.strokes) + "/par " + itoa(g.currentPar())
	if g.lv.Kind == "navigate" {
		s := "keys " + itoa(g.keysNeed-len(g.keyPos)) + "/" + itoa(g.keysNeed)
		if p := g.pestsLeft(); p > 0 {
			s += "   ·   " + itoa(p) + " bug(s) left"
		} else if len(g.keyPos) == 0 {
			s += "   ·   now head to $ (exit)!"
		}
		domSet("status", s+parInfo)
	} else {
		domSet("status", "Make CURRENT match TARGET — a line turns green when it matches!"+parInfo)
	}
}

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle(gameName)
	registerJSHooks()
	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}
