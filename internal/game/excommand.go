package game

// excommand.go — ":" 명령줄(ex-command) pseudo-mode. 검색과 같은 골격
// (bool 플래그 + 버퍼 + esc/cr/bs)을 재사용하되, :q/:restart 처럼 Editor
// 밖의 Game 상태를 조작해야 해서 엔진이 아닌 게임에 둔다. 진입(':')과
// 모드 분기는 game.go 의 feed() 가 맡고, 여기는 입력 수집(feedEx)과
// 해석·실행(runExCommand)만 소유한다 — drill.go/arena_levels.go 와 같은
// "하위 상태 머신은 자기 파일을 갖는다" 결.

import (
	"strconv"
	"strings"

	"vimquest/internal/engine"
	"vimquest/internal/platform"
)

// feedEx 는 ":" 명령줄 입력을 처리하는 Game 레벨 pseudo-mode.
// 검색과 같은 골격(bool 플래그 + 버퍼 + esc/cr/bs)을 재사용하되,
// :q/:restart 처럼 Editor 밖의 Game 상태를 조작해야 해서 엔진이 아닌
// 게임에 둔다.
func (g *Game) feedEx(k engine.Key) {
	switch k.S {
	case "esc":
		g.exMode = false
		g.exBuf = nil
		return
	case "cr":
		cmd := string(g.exBuf)
		g.exMode = false
		g.exBuf = nil
		g.runExCommand(cmd)
		return
	case "bs":
		if len(g.exBuf) > 0 {
			g.exBuf = g.exBuf[:len(g.exBuf)-1]
		}
		return
	}
	if k.R != 0 {
		g.exBuf = append(g.exBuf, k.R)
	}
}

// runExCommand 는 확정된 ex-command 문자열을 해석해 실행한다.
// 인식하지 못한 명령은 조용히 무시한다(터미널처럼 무반응 — 에러 팝업 없음).
func (g *Game) runExCommand(cmd string) {
	switch {
	case cmd == "q" || cmd == "levels":
		if g.state == StateDrill {
			// 드릴 중엔 바로 레벨 선택으로 나가지 않고, 이번 세션 통계를 한 번
			// 보여준다 — drillStreak/drillTotalKeys/drillTotalPar 는 세션 내내
			// 누적돼 온 값이라 여기서 다시 계산할 필요 없이 그대로 읽으면 된다.
			g.state = StateDrillSummary
		} else {
			g.EnterLevelSelect()
		}
	case cmd == "restart" || cmd == "e!":
		g.RestartCurrent()
	case cmd == "help":
		platform.ShowOverlay("intro")
	case cmd == "drill" || strings.HasPrefix(cmd, "drill "):
		// 아레나 도중 모드 전환은 무시한다(미인식 명령과 같은 무반응 원칙) —
		// 허용하면 아레나 패널·타이머 아래에 드릴이 깔리는 화면 분열이 생긴다.
		// 런 포기는 :q(레벨 선택 이탈) 하나만 열어 둔다.
		if g.state == StateArena {
			return
		}
		// ":drill w"/":drill f"/":drill x" — 인자별로 다른 생성기(drill.go).
		g.enterDrill(strings.TrimSpace(strings.TrimPrefix(cmd, "drill")))
	default:
		if n, err := strconv.Atoi(cmd); err == nil && n > 0 {
			g.ed.GotoLine(n)
		}
	}
}
