package game

// dom.go — 캔버스 밖 한국어 UI(#level-title/#hint/#status/#solve-cmds)를
// 현재 게임 상태와 동기화한다. 실제 DOM 접근은 platform 패키지가 맡고,
// 데스크톱 빌드에서는 전부 no-op 이 된다.

import (
	"strconv"
	"strings"

	"vimquest/internal/platform"
)

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
	itoa := strconv.Itoa
	switch g.state {
	case StateAllClear:
		platform.SetText("level-title", "🎉 ALL CLEAR!")
		platform.SetText("hint", "Congratulations! You cleared all "+itoa(len(levels))+" levels across W1-W"+itoa(len(WorldGroups()))+". Now go practice in real Vim!")
		platform.SetText("status", "")
		platform.SetHTML("solve-cmds", "")
		return
	case StateLevelClear:
		platform.SetText("level-title", "LEVEL "+g.lv.ID+" CLEAR!")
		platform.SetText("hint", "Press Enter for the next level, or r to retry this one.")
		platform.SetText("status", "keys "+itoa(g.clear.Strokes)+" / par "+itoa(g.clear.Par))
		platform.SetHTML("solve-cmds", "")
		return
	case StateLevelSelect:
		platform.SetText("level-title", "SELECT LEVEL")
		platform.SetText("hint", "h/l move between worlds, j/k move within a world, Enter to play, Esc to go back.")
		platform.SetText("status", "")
		platform.SetHTML("solve-cmds", "")
		return
	}

	platform.SetText("level-title", g.lv.Title)
	platform.SetText("hint", g.lv.Hint)
	platform.SetHTML("solve-cmds", cmdsHTML(g.lv.Cmds))
	parInfo := "   ·   keys " + itoa(g.strokes) + "/par " + itoa(g.Par())
	if g.state == StateDrill {
		parInfo += "   ·   streak " + itoa(g.drillStreak) + "   ·   total " + itoa(g.drillTotalKeys) + "/" + itoa(g.drillTotalPar)
	}
	if g.lv.Kind == "navigate" {
		s := "keys " + itoa(g.keysNeed-len(g.keyPos)) + "/" + itoa(g.keysNeed)
		if p := g.PestsLeft(); p > 0 {
			s += "   ·   " + itoa(p) + " bug(s) left"
		} else if len(g.keyPos) == 0 {
			s += "   ·   now head to $ (exit)!"
		}
		platform.SetText("status", s+parInfo)
	} else {
		platform.SetText("status", "Make CURRENT match TARGET — a line turns green when it matches!"+parInfo)
	}
}
