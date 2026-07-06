package game

import "strings"

// Cmd is one "command + description" line shown in the top-right HINT panel.
// Instead of handing over the full answer sequence, it lists the commands used in
// this level and what they do, so the player composes the solution themselves.
type Cmd struct {
	K string // key (e.g. "w", "dw", "f{char}")
	D string // English description
}

// LevelMeta 는 레벨의 표시 전용 데이터 — 게임 규칙(par/승리 판정)에 쓰이지
// 않으므로 Level 과 분리해 wasm 에 싣지 않는다. 커리큘럼 레벨의 실제 값은
// levels_meta.go(!js 빌드 태그)에 있고, 웹은 tools/genmeta 가 생성한
// web/src/levels_meta.js 를 통해 같은 데이터를 읽는다.
type LevelMeta struct {
	Title string
	Hint  string
	Cmds  []Cmd
}

// Level is one stage. Two kinds:
//
//	"navigate" — move the cursor to collect keys (K) and remove bugs (*) with x, then reach exit ($).
//	             map glyphs: @ start · . floor · K key · * bug · $ exit · (space) word separator.
//	"edit"     — transform the buffer to match Target exactly (VimGolf style).
//	             Solution is the verified answer used by tests to confirm solvability (hidden in-game).
//
// Title/Hint 는 :drill 이 런타임에 생성하는 문제에만 채워진다 — 커리큘럼
// 레벨의 표시 데이터(제목·힌트·명령 팔레트)는 LevelMeta(levels_meta.go)가
// 소유한다. 여기 두면 wasm 페이로드에 실리기 때문이다.
type Level struct {
	ID       string // "1-1", "3-4" 등 — par 산출/저장소 키/월드 그룹핑/메타 조회에 쓰인다.
	Title    string // 런타임 생성 레벨(:drill) 전용
	Hint     string // 런타임 생성 레벨(:drill) 전용
	Kind     string // "navigate" | "edit"
	Map      []string
	Target   []string // edit only
	Solution string   // verification (navigate/edit 공통) + par 산출 기준, 3★ 클리어 전까지 게임 내 비공개
}

// WorldGroups 는 levels 를 ID 접두어가 바뀌는 지점 기준으로
// 묶는다 — 그래서 같은 월드의 레벨은 반드시 이 배열 안에서 서로 인접해야
// 한다(월드별로 나중에 추가한 레벨을 배열 맨 끝에 몰아 붙이면 그 월드가
// 두 조각으로 쪼개져 레벨 선택 화면에 엉뚱한 열이 하나 더 생긴다).
var levels = []Level{
	// ───────────────────────── W1  The Moving Woods (basic motion) ─────────────────────────
	{
		ID:   "1-1",
		Kind: "navigate",
		Map: []string{
			"@.........",
			"..........",
			"....K.....",
			"..........",
			".........$",
		},
		Solution: "jjlllljjlllll",
	},
	{
		ID:   "1-2",
		Kind: "navigate",
		Map: []string{
			"@start  the  long  road  K  to  the  exit  $",
		},
		Solution: "wwwwwwwww",
	},
	{
		ID:   "1-3",
		Kind: "navigate",
		Map: []string{
			"@...........K",
			".............",
			"K...........$",
		},
		Solution: "$0jj$",
	},
	{
		ID:   "1-4",
		Kind: "navigate",
		Map: []string{
			"@....K..........$",
		},
		Solution: "fKf$",
	},
	{
		ID:   "1-5",
		Kind: "navigate",
		Map: []string{
			"@..*......",
			".....K....",
			"..*.......",
			".........$",
		},
		Solution: "lllxjlljhhhxjlllllll",
	},
	{
		ID:       "1-6",
		Kind:     "navigate",
		Map:      []string{"K" + strings.Repeat(".", 14) + "@K" + strings.Repeat(".", 12) + "$"},
		Solution: "FKfKf$",
	},

	// ───────────────────────── W2  Jump Canyon (fast motion) ─────────────────────────
	{
		ID:   "2-1",
		Kind: "navigate",
		Map: []string{
			"@a.b.c.d.K.e.f.g.$",
		},
		Solution: "fKf$",
	},
	{
		ID:   "2-2",
		Kind: "navigate",
		Map: []string{
			"@.........",
			"....*.....",
			"..........",
			"......K...",
			"..........",
			"*.........",
			".........$",
		},
		Solution: "jllllxjjlljjhhhhhhxjlllllllll",
	},
	{
		ID:   "2-3",
		Kind: "navigate",
		Map: []string{
			"@one  two  three  K",
			"..................",
			"K  four  five  six  $",
		},
		Solution: "$0jj$",
	},
	{
		// 보너스: 1-6 과 같은 패턴("K"가 시작 뒤에 있어 F 로 되짚어야 하고,
		// 반대편 K/$ 까지는 f 로 건너뛰는 게 유리) — hjkl 로만 풀면 훨씬
		// 비효율적임을 TestBonusLandmarkLevelsNaiveSolveIsWorse 가 보증한다.
		ID:       "2-4",
		Kind:     "navigate",
		Map:      []string{"K" + strings.Repeat(".", 8) + "@K" + strings.Repeat(".", 16) + "$"},
		Solution: "FKfKf$",
	},

	// ───────────────────────── W3  The Editing Dungeon (operators + Insert) ─────────────────────────
	{
		ID:       "3-1",
		Kind:     "edit",
		Map:      []string{"hellllo world"},
		Target:   []string{"hello world"},
		Solution: "flxx",
	},
	{
		ID:       "3-2",
		Kind:     "edit",
		Map:      []string{"good line", "DELETE THIS", "another good", "DELETE THIS"},
		Target:   []string{"good line", "another good"},
		Solution: "jddjdd",
	},
	{
		ID:       "3-3",
		Kind:     "edit",
		Map:      []string{"hello cruel world"},
		Target:   []string{"hello world"},
		Solution: "wdw",
	},
	{
		ID:       "3-4",
		Kind:     "edit",
		Map:      []string{"Hello"},
		Target:   []string{"Hello, World!"},
		Solution: "A, World!<esc>",
	},
	{
		ID:       "3-5",
		Kind:     "edit",
		Map:      []string{"fix THIS word"},
		Target:   []string{"fix that word"},
		Solution: "wcwthat<esc>",
	},
	{
		ID:       "3-6",
		Kind:     "edit",
		Map:      []string{"foo foo foo"},
		Target:   []string{"bar bar bar"},
		Solution: "cwbar<esc>w.w.",
	},
	{
		// 보너스: 연산자+count("d6w")로 단어 6개를 한 번에 지우는 것과, count
		// 없이 "dw"를 여섯 번 반복하는 것의 타수 차이를 보여준다.
		ID:       "3-7",
		Kind:     "edit",
		Map:      []string{"remove one two three four five words here"},
		Target:   []string{"remove here"},
		Solution: "wd6w",
	},

	// ───────────────────────── W4  Temple of Text Objects (advanced editing) ─────────────────────────
	{
		ID:       "4-1",
		Kind:     "edit",
		Map:      []string{"one BAD two"},
		Target:   []string{"one two"},
		Solution: "wdaw",
	},
	{
		ID:       "4-2",
		Kind:     "edit",
		Map:      []string{"call(oldArg)"},
		Target:   []string{"call(newArg)"},
		Solution: "f(ci(newArg<esc>",
	},
	{
		ID:       "4-3",
		Kind:     "edit",
		Map:      []string{"title = \"old\""},
		Target:   []string{"title = \"new\""},
		Solution: "f\"ci\"new<esc>",
	},
	{
		ID:       "4-4",
		Kind:     "edit",
		Map:      []string{"duplicate me"},
		Target:   []string{"duplicate me", "duplicate me"},
		Solution: "yyp",
	},
	{
		ID:       "4-5",
		Kind:     "edit",
		Map:      []string{"x = OLD", "y = OLD"},
		Target:   []string{"x = 42", "y = 42"},
		Solution: "$ciw42<esc>j$.",
	},
	{
		// 보너스: 괄호 안 내용을 text object(di()로 한 번에 비우는 것과,
		// text object 없이 문자 하나씩 x 로 지우는 것의 타수 차이를 보여준다.
		ID:       "4-6",
		Kind:     "edit",
		Map:      []string{"keep(one two three four five)keep"},
		Target:   []string{"keep()keep"},
		Solution: "f(di(",
	},

	// ───────────────────────── W5  Search Swamp (search) ─────────────────────────
	{
		ID:   "5-1",
		Kind: "navigate",
		Map: []string{
			"@..........................K...................$",
		},
		Solution: "/K<cr>/$<cr>",
	},
	{
		ID:   "5-2",
		Kind: "navigate",
		Map: []string{
			"@.....K.......K.......$",
		},
		Solution: "/K<cr>n/$<cr>",
	},
	{
		ID:   "5-3",
		Kind: "navigate",
		Map: []string{
			"@.........K.............$",
		},
		Solution: "/$<cr>?K<cr>/$<cr>",
	},
	{
		ID:   "5-4",
		Kind: "navigate",
		Map: []string{
			"@....K.........K.............K.......$",
		},
		Solution: "/K<cr>nn/$<cr>",
	},
	{
		// 보너스: 1-6 과 같은 K-behind/K-ahead 패턴 — hjkl 로만 풀면 훨씬
		// 비효율적임을 TestBonusLandmarkLevelsNaiveSolveIsWorse 가 보증한다.
		ID:       "5-5",
		Kind:     "navigate",
		Map:      []string{"K" + strings.Repeat(".", 15) + "@K" + strings.Repeat(".", 25) + "$"},
		Solution: "FKfKf$",
	},

	// ───────────────────────── W6  Precision Peaks (count, F/t, edit count) ─────────────────────────
	{
		ID:   "6-1",
		Kind: "navigate",
		Map: []string{
			"@one two three K four five six $",
		},
		Solution: "4w4w",
	},
	{
		ID:   "6-2",
		Kind: "navigate",
		Map: []string{
			"K.......@..........$",
		},
		Solution: "FK$",
	},
	{
		ID:   "6-3",
		Kind: "navigate",
		Map: []string{
			"@..........KX.................$",
		},
		Solution: "tX$",
	},
	{
		ID:       "6-4",
		Kind:     "edit",
		Map:      []string{"keep BAD WORDS here"},
		Target:   []string{"keep here"},
		Solution: "wd2w",
	},
	{
		ID:   "6-5",
		Kind: "navigate",
		Map: []string{
			"@one two three K x.y.z.w.$",
		},
		Solution: "4wf$",
	},
	{
		// 보너스: 1-6 과 같은 K-behind/K-ahead 패턴 — hjkl 로만 풀면 훨씬
		// 비효율적임을 TestBonusLandmarkLevelsNaiveSolveIsWorse 가 보증한다.
		ID:       "6-6",
		Kind:     "navigate",
		Map:      []string{"K" + strings.Repeat(".", 10) + "@K" + strings.Repeat(".", 20) + "$"},
		Solution: "FKfKf$",
	},

	// ───────────────────────── W7  Visual Valley (visual mode + text objects) ─────────────────────────
	{
		ID:       "7-1",
		Kind:     "edit",
		Map:      []string{"keep THIS OUT keep"},
		Target:   []string{"keep  keep"},
		Solution: "wvEEd",
	},
	{
		ID:       "7-2",
		Kind:     "edit",
		Map:      []string{"dup ME now"},
		Target:   []string{"dup ME ME now"},
		Solution: "wyawP",
	},
	{
		ID:       "7-3",
		Kind:     "edit",
		Map:      []string{"fix THIS please"},
		Target:   []string{"fix OK please"},
		Solution: "wviwcOK<esc>",
	},
	{
		ID:       "7-4",
		Kind:     "edit",
		Map:      []string{"cut BAD here fix OLD there"},
		Target:   []string{"cut here fix NEW there"},
		Solution: "wvawdwwciwNEW<esc>",
	},
	{
		// 보너스: Visual 로 여러 단어를 한 번에 선택해 지우는 것과, 문자
		// 하나씩 x 로 지우는 것의 타수 차이를 보여준다.
		ID:       "7-5",
		Kind:     "edit",
		Map:      []string{"keep one two three four keep"},
		Target:   []string{"keep  keep"},
		Solution: "wveeeed",
	},

	// ───────────────────────── W8  Yank & Undo Ruins (xp, ddp, p/P, u/Ctrl-r) ─────────────────────────
	{
		ID:       "8-1",
		Kind:     "edit",
		Map:      []string{"abcd"},
		Target:   []string{"bacd"},
		Solution: "xp",
	},
	{
		ID:       "8-2",
		Kind:     "edit",
		Map:      []string{"first", "second"},
		Target:   []string{"second", "first"},
		Solution: "ddp",
	},
	{
		ID:       "8-3",
		Kind:     "edit",
		Map:      []string{"one", "two"},
		Target:   []string{"one", "one", "two"},
		Solution: "yyjP",
	},
	{
		ID:       "8-4",
		Kind:     "edit",
		Map:      []string{"keep", "BAD1", "BAD2"},
		Target:   []string{"keep", "BAD2"},
		Solution: "jddddu",
	},
	{
		ID:       "8-5",
		Kind:     "edit",
		Map:      []string{"third", "first", "second"},
		Target:   []string{"first", "second", "third"},
		Solution: "ddjp",
	},
	{
		ID:       "8-6",
		Kind:     "edit",
		Map:      []string{"one BAD two", "x = OLD", "abcd"},
		Target:   []string{"one two", "x = 99", "bacd"},
		Solution: "wdawj$ciw99<esc>j0xp",
	},
	{
		// 보너스: yank+paste 로 줄을 복제하는 것과, 매번 새로 타이핑해서
		// 줄을 추가하는 것의 타수 차이를 보여준다.
		ID:       "8-7",
		Kind:     "edit",
		Map:      []string{"template line"},
		Target:   []string{"template line", "template line", "template line", "template line"},
		Solution: "yyppp",
	},

	// ───────────────────────── W9  Macro Mines (q/@/@@, %) ─────────────────────────
	{
		ID:   "9-1",
		Kind: "edit",
		Map: []string{
			"fix THIS line", "fix THIS line", "fix THIS line",
			"fix THIS line", "fix THIS line", "fix THIS line",
		},
		Target: []string{
			"fix that line", "fix that line", "fix that line",
			"fix that line", "fix that line", "fix that line",
		},
		Solution: "qawcwthat<esc>j0q5@a",
	},
	{
		ID:   "9-2",
		Kind: "edit",
		Map: []string{
			"one BAD two", "one BAD two", "one BAD two", "one BAD two",
			"one BAD two", "one BAD two", "one BAD two",
		},
		Target: []string{
			"one two", "one two", "one two", "one two",
			"one two", "one two", "one two",
		},
		Solution: "qawdawj0q6@a",
	},
	{
		ID:   "9-3",
		Kind: "edit",
		Map: []string{
			"process(data)", "process(data)", "process(data)",
			"process(data)", "process(data)",
		},
		Target: []string{
			"process(data);", "process(data);", "process(data);",
			"process(data);", "process(data);",
		},
		Solution: "qaf(%a;<esc>j0q4@a",
	},
	{
		ID:   "9-4",
		Kind: "edit",
		Map: []string{
			"cut BAD here fix OLD there", "cut BAD here fix OLD there",
			"cut BAD here fix OLD there", "cut BAD here fix OLD there",
			"cut BAD here fix OLD there", "cut BAD here fix OLD there",
		},
		Target: []string{
			"cut here fix NEW there", "cut here fix NEW there",
			"cut here fix NEW there", "cut here fix NEW there",
			"cut here fix NEW there", "cut here fix NEW there",
		},
		Solution: "qawvawdwwciwNEW<esc>j0q5@a",
	},
	{
		ID:   "9-5",
		Kind: "edit",
		Map: []string{
			"val = OLD", "val = OLD", "val = OLD", "val = OLD",
			"val = OLD", "val = OLD", "val = OLD", "val = OLD",
		},
		Target: []string{
			"val = 42", "val = 42", "val = 42", "val = 42",
			"val = 42", "val = 42", "val = 42", "val = 42",
		},
		Solution: "qa$ciw42<esc>j0q7@a",
	},
}

// LevelCount 는 전체 커리큘럼 레벨 수(렌더러의 "level N/M" 표기용).
func LevelCount() int { return len(levels) }

// LevelAt 은 i 번째 레벨 정의를 돌려준다(레벨 선택 화면 렌더용).
func LevelAt(i int) Level { return levels[i] }

// worldGroupsCache 는 WorldGroups() 의 계산 결과를 담아둔다. levels 는 런타임에
// 절대 바뀌지 않는 정적 슬라이스라 한 번만 계산하면 된다 — 캐싱이 없으면
// 전체 클리어 화면처럼 매 프레임(최대 60Hz) 불리는 경로에서 매번
// 재스캔+재할당된다. 단일 고루틴(Ebiten Update/Draw, 또는 wasm/JS 단일 스레드
// 이벤트 루프)에서만 호출되므로 락 없이 캐싱해도 안전하다.
var worldGroupsCache [][]int

// WorldGroups 는 levels 를 Level.ID 접두어(월드 번호) 기준으로 묶어
// [월드][월드 내 레벨] = levels 인덱스 형태로 반환한다.
func WorldGroups() [][]int {
	if worldGroupsCache != nil {
		return worldGroupsCache
	}
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
	worldGroupsCache = groups
	return groups
}
