package game

// snapshot.go — JS 렌더러(web/renderer.js)와의 데이터 계약.
// 값은 syscall/js 의 ValueOf 제약 때문에 map[string]any / []any / 원시 타입만
// 쓸 수 있다. 데스크톱 렌더러는 view.go 의 타입 접근자를 쓰므로 이 파일의
// 소비자는 웹뿐이지만, 계약 자체는 js 와 무관한 순수 데이터라 여기(게임
// 패키지)에 둔다 — 렌더러는 게임 규칙을 몰라도 된다(듀얼 프론트엔드 드리프트
// 방지).

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// Snapshot 은 프론트엔드가 게임 상태를 그리는 데 필요한 전부를 담은
// 순수 데이터 스냅샷을 만든다.
//
// 3★ 미만이면 solution 을 빈 문자열로 내려보낸다 — 스포일러 방지 규칙은
// 엔진이 소유하고 클라이언트가 아니다.
func (g *Game) Snapshot() map[string]any {
	base := map[string]any{
		"level":      g.levelIdx + 1,
		"levelCount": len(levels),
	}

	switch g.state {
	case StateAllClear:
		base["state"] = "allclear"
		base["worldCount"] = len(WorldGroups())
		return base

	case StateLevelClear:
		base["state"] = "clear"
		base["id"] = g.lv.ID
		base["clearStrokes"] = g.clear.Strokes
		base["clearPar"] = g.clear.Par
		base["clearStars"] = g.clear.Stars
		base["clearBest"] = g.clear.Best
		solution := ""
		if g.clear.Stars == 3 {
			solution = g.lv.Solution
		}
		base["solution"] = solution
		return base

	case StateLevelSelect:
		base["state"] = "select"
		worlds := WorldGroups()
		wOut := make([]any, len(worlds))
		for wi, group := range worlds {
			lvOut := make([]any, len(group))
			for li, idx := range group {
				lv := levels[idx]
				prog := g.progress[lv.ID]
				lvOut[li] = map[string]any{
					"id":       lv.ID,
					"unlocked": prog.Unlocked,
					"stars":    prog.Stars,
				}
			}
			wOut[wi] = lvOut
		}
		base["worlds"] = wOut
		base["selRow"] = g.selRow
		base["selCol"] = g.selCol
		return base
	}

	// StatePlaying / StateDrill
	base["state"] = "playing"
	if g.state == StateDrill {
		base["state"] = "drill"
		base["drill"] = map[string]any{
			"streak":    g.drillStreak,
			"totalKeys": g.drillTotalKeys,
			"totalPar":  g.drillTotalPar,
		}
	}
	base["kind"] = g.lv.Kind
	base["id"] = g.lv.ID
	base["title"] = g.lv.Title
	base["lines"] = toAnySlice(g.ed.Lines())
	base["row"] = g.ed.Row()
	base["col"] = g.ed.Col()
	base["mode"] = g.ed.ModeName()
	base["pending"] = g.ed.PendingString()
	base["last"] = g.ed.LastKey()
	base["strokes"] = g.strokes
	base["par"] = g.Par()
	base["exMode"] = g.exMode
	base["exBuf"] = string(g.exBuf)
	base["bell"] = g.bellTTL > 0

	if g.lv.Kind == "navigate" {
		base["keys"] = g.keysNeed - len(g.keyPos)
		base["keysNeed"] = g.keysNeed
		base["bugs"] = g.PestsLeft()
		kp := make([]any, 0, len(g.keyPos))
		for pos := range g.keyPos {
			kp = append(kp, map[string]any{"row": pos[0], "col": pos[1]})
		}
		base["keyPos"] = kp
	} else {
		base["target"] = toAnySlice(g.lv.Target)
	}

	r1, c1, r2, c2, line, ok := g.ed.VisualSpan()
	base["visual"] = map[string]any{"r1": r1, "c1": c1, "r2": r2, "c2": c2, "line": line, "ok": ok}

	effs := make([]any, len(g.effects))
	for i, e := range g.effects {
		glyph := ""
		if e.Glyph != 0 {
			glyph = string(e.Glyph)
		}
		effs[i] = map[string]any{"row": e.Row, "col": e.Col, "glyph": glyph, "invert": e.Invert}
	}
	base["effects"] = effs
	base["effectsAlive"] = len(g.effects) > 0 || g.bellTTL > 0

	return base
}
