package game

import "strings"

// Cmd is one "command + description" line shown in the top-right HINT panel.
// Instead of handing over the full answer sequence, it lists the commands used in
// this level and what they do, so the player composes the solution themselves.
type Cmd struct {
	K string // key (e.g. "w", "dw", "f{char}")
	D string // English description
}

// Level is one stage. Two kinds:
//
//	"navigate" — move the cursor to collect keys (K) and remove bugs (*) with x, then reach exit ($).
//	             map glyphs: @ start · . floor · K key · * bug · $ exit · (space) word separator.
//	"edit"     — transform the buffer to match Target exactly (VimGolf style).
//	             Solution is the verified answer used by tests to confirm solvability (hidden in-game).
type Level struct {
	ID       string // "1-1", "3-4" 등 — Title 접두어와 동일. par 산출/저장소 키/월드 그룹핑에 쓰인다.
	Title    string
	Hint     string // the goal + which kind of command to use (NOT the literal answer)
	Cmds     []Cmd  // command palette for this level
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
		ID:    "1-1",
		Kind:  "navigate",
		Title: "1-1  First Steps",
		Hint:  "Grab the K (key), then reach the $ (exit). Use hjkl instead of the arrow keys to move.",
		Cmds: []Cmd{
			{"h", "← left one cell"},
			{"j", "↓ down one"},
			{"k", "↑ up one"},
			{"l", "→ right one"},
		},
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
		ID:    "1-2",
		Kind:  "navigate",
		Title: "1-2  Word Jumps",
		Hint:  "Don't crawl one cell at a time — jump word by word. Grab the key and head to $.",
		Cmds: []Cmd{
			{"w", "jump to start of next word"},
			{"b", "to start of previous word"},
			{"e", "to end of word"},
		},
		Map: []string{
			"@start  the  long  road  K  to  the  exit  $",
		},
		Solution: "wwwwwwwww",
	},
	{
		ID:    "1-3",
		Kind:  "navigate",
		Title: "1-3  Start & End of Line",
		Hint:  "The keys sit at both ends of the line. Use the keys that jump straight to the start/end.",
		Cmds: []Cmd{
			{"0", "to start of line"},
			{"$", "to end of line"},
			{"^", "to first non-blank char"},
			{"j k", "move down/up a line"},
		},
		Map: []string{
			"@...........K",
			".............",
			"K...........$",
		},
		Solution: "$0jj$",
	},
	{
		ID:    "1-4",
		Kind:  "navigate",
		Title: "1-4  Find a Character",
		Hint:  "Use a find command to leap directly to a far-off character (K, $).",
		Cmds: []Cmd{
			{"f{char}", "leap to that char on this line (e.g. fK)"},
			{";", "repeat the last find once more"},
		},
		Map: []string{
			"@....K..........$",
		},
		Solution: "fKf$",
	},
	{
		ID:    "1-5",
		Kind:  "navigate",
		Title: "1-5  Bug Hunt",
		Hint:  "Bugs (*) are killed by moving onto them and pressing the delete key. Dart up and down, clear them all, then grab the key and exit.",
		Cmds: []Cmd{
			{"gg", "warp to first line"},
			{"G", "warp to last line"},
			{"x", "delete char under cursor (kill bug)"},
			{"h j k l", "move one cell"},
		},
		Map: []string{
			"@..*......",
			".....K....",
			"..*.......",
			".........$",
		},
		Solution: "lllxjlljhhhxjlllllll",
	},

	// ───────────────────────── W2  Jump Canyon (fast motion) ─────────────────────────
	{
		ID:    "2-1",
		Kind:  "navigate",
		Title: "2-1  Repeat Jumps",
		Hint:  "There's a key that repeats your last find. Find once, then repeat to skip across the dots.",
		Cmds: []Cmd{
			{"f{char}", "jump to that char (e.g. f.)"},
			{";", "repeat find (forward)"},
			{",", "repeat find (backward)"},
		},
		Map: []string{
			"@a.b.c.d.K.e.f.g.$",
		},
		Solution: "fKf$",
	},
	{
		ID:    "2-2",
		Kind:  "navigate",
		Title: "2-2  Jump by Number",
		Hint:  "Jumping by line number makes vertical travel fast. Clear the bugs and reach the key/exit.",
		Cmds: []Cmd{
			{"{N}G", "jump to line N (e.g. 4G)"},
			{"gg", "to first line"},
			{"G", "to last line"},
			{"x", "delete bug"},
		},
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
		ID:    "2-3",
		Kind:  "navigate",
		Title: "2-3  Canyon Run",
		Hint:  "Use everything you've learned! Grab the keys at both ends and find the fastest path to the exit.",
		Cmds: []Cmd{
			{"w", "word jump"},
			{"f{char}", "jump to char"},
			{"0 $", "start / end of line"},
			{"gg G", "first / last line"},
		},
		Map: []string{
			"@one  two  three  K",
			"..................",
			"K  four  five  six  $",
		},
		Solution: "$0jj$",
	},

	// ───────────────────────── W3  The Editing Dungeon (operators + Insert) ─────────────────────────
	{
		ID:    "3-1",
		Kind:  "edit",
		Title: "3-1  Fix Typos with x",
		Hint:  "Too many letters (a typo). Move onto the extra letters and delete them to match the target on the right.",
		Cmds: []Cmd{
			{"f{char}", "jump to the typo"},
			{"x", "delete one char under cursor"},
			{"h l", "move left/right"},
		},
		Map:      []string{"hellllo world"},
		Target:   []string{"hello world"},
		Solution: "flxx",
	},
	{
		ID:    "3-2",
		Kind:  "edit",
		Title: "3-2  Delete Lines with dd",
		Hint:  "Some lines aren't needed. Move to them and delete whole lines to leave only the target.",
		Cmds: []Cmd{
			{"j", "down a line"},
			{"k", "up a line"},
			{"dd", "delete the whole current line"},
		},
		Map:      []string{"good line", "DELETE THIS", "another good", "DELETE THIS"},
		Target:   []string{"good line", "another good"},
		Solution: "jddjdd",
	},
	{
		ID:    "3-3",
		Kind:  "edit",
		Title: "3-3  Delete Words with dw",
		Hint:  "An unwanted word is wedged in the sentence. Use 'delete operator d + word motion w' to remove it whole.",
		Cmds: []Cmd{
			{"w", "move by word"},
			{"d", "delete operator (takes a motion)"},
			{"dw", "delete one word from cursor"},
		},
		Map:      []string{"hello cruel world"},
		Target:   []string{"hello world"},
		Solution: "wdw",
	},
	{
		ID:    "3-4",
		Kind:  "edit",
		Title: "3-4  Append Text with A",
		Hint:  "You need to append text to the end of the line. Enter insert mode, type, then press Esc to leave it.",
		Cmds: []Cmd{
			{"A", "start typing at end of line (Insert)"},
			{"(type)", "enter your text"},
			{"Esc", "leave insert → Normal"},
		},
		Map:      []string{"Hello"},
		Target:   []string{"Hello, World!"},
		Solution: "A, World!<esc>",
	},
	{
		ID:    "3-5",
		Kind:  "edit",
		Title: "3-5  Replace a Word with cw",
		Hint:  "Just one word needs swapping. The 'change operator c' deletes and drops you straight into insert.",
		Cmds: []Cmd{
			{"w", "move to the word"},
			{"cw", "delete the word and start typing"},
			{"Esc", "leave insert"},
		},
		Map:      []string{"fix THIS word"},
		Target:   []string{"fix that word"},
		Solution: "wcwthat<esc>",
	},
	{
		ID:    "3-6",
		Kind:  "edit",
		Title: "3-6  Repeat with .",
		Hint:  "All three words must become the same word. Use the key that repeats your last change to fly through it.",
		Cmds: []Cmd{
			{"cw", "delete word and type"},
			{"Esc", "leave insert"},
			{"w", "to next word"},
			{".", "repeat the last change"},
		},
		Map:      []string{"foo foo foo"},
		Target:   []string{"bar bar bar"},
		Solution: "cwbar<esc>w.w.",
	},

	// ───────────────────────── W4  Temple of Text Objects (advanced editing) ─────────────────────────
	{
		ID:    "4-1",
		Kind:  "edit",
		Title: "4-1  Delete a Whole Word with daw",
		Hint:  "Delete a single word, trailing space and all. Use the 'a word' text object with the delete operator.",
		Cmds: []Cmd{
			{"w", "move to the word"},
			{"daw", "delete a word incl. its space"},
		},
		Map:      []string{"one BAD two"},
		Target:   []string{"one two"},
		Solution: "wdaw",
	},
	{
		ID:    "4-2",
		Kind:  "edit",
		Title: "4-2  Change Inside ( ) with ci(",
		Hint:  "Only the contents inside the ( ) need changing. Use the 'inner' text object to target just inside the parentheses.",
		Cmds: []Cmd{
			{"f{char}", "move to ( (e.g. f( )"},
			{"ci(", "change inside the parentheses"},
			{"Esc", "leave insert"},
		},
		Map:      []string{"call(oldArg)"},
		Target:   []string{"call(newArg)"},
		Solution: "f(ci(newArg<esc>",
	},
	{
		ID:    "4-3",
		Kind:  "edit",
		Title: "4-3  Change Inside \" \" with ci\"",
		Hint:  "Only the text inside the \" \" needs changing. Use the text object that selects inside the quotes.",
		Cmds: []Cmd{
			{"f{char}", "move to \""},
			{"ci\"", "change inside the quotes"},
			{"Esc", "leave insert"},
		},
		Map:      []string{"title = \"old\""},
		Target:   []string{"title = \"new\""},
		Solution: "f\"ci\"new<esc>",
	},
	{
		ID:    "4-4",
		Kind:  "edit",
		Title: "4-4  Duplicate a Line with yy/p",
		Hint:  "You need one more copy of the line. Copy (yank) the line and paste it to duplicate.",
		Cmds: []Cmd{
			{"yy", "copy (yank) the line"},
			{"p", "paste the copied line below"},
		},
		Map:      []string{"duplicate me"},
		Target:   []string{"duplicate me", "duplicate me"},
		Solution: "yyp",
	},
	{
		ID:    "4-5",
		Kind:  "edit",
		Title: "4-5  Boss: ciw + . Combo",
		Hint:  "Turn both OLD into 42! Change one line, then use the 'repeat' key to do the next line the same way.",
		Cmds: []Cmd{
			{"$", "to end of line"},
			{"ciw", "change the word under cursor"},
			{"Esc", "leave insert"},
			{"j", "down a line"},
			{".", "repeat the last change"},
		},
		Map:      []string{"x = OLD", "y = OLD"},
		Target:   []string{"x = 42", "y = 42"},
		Solution: "$ciw42<esc>j$.",
	},

	// ───────────────────────── W5  Search Swamp (search) ─────────────────────────
	{
		ID:    "5-1",
		Kind:  "navigate",
		Title: "5-1  Search Swamp",
		Hint:  "The key is buried far down the swamp. Search for it instead of crawling — then search for the exit too.",
		Cmds: []Cmd{
			{"/{pattern}", "search forward for text"},
			{"<cr>", "confirm search"},
		},
		Map: []string{
			"@..........................K...................$",
		},
		Solution: "/K<cr>/$<cr>",
	},
	{
		ID:    "5-2",
		Kind:  "navigate",
		Title: "5-2  Twin Keys",
		Hint:  "Two keys share the swamp. Search once, then repeat the search to reach the second.",
		Cmds: []Cmd{
			{"/{pattern}", "search forward"},
			{"n", "repeat last search forward"},
		},
		Map: []string{
			"@.....K.......K.......$",
		},
		Solution: "/K<cr>n/$<cr>",
	},
	{
		ID:    "5-3",
		Kind:  "navigate",
		Title: "5-3  Backtrack",
		Hint:  "You'll overshoot the key if you rush to the exit. Search backward to retrieve what you missed, then forward again.",
		Cmds: []Cmd{
			{"/{pattern}", "search forward"},
			{"?{pattern}", "search backward"},
		},
		Map: []string{
			"@.........K.............$",
		},
		Solution: "/$<cr>?K<cr>/$<cr>",
	},
	{
		ID:    "5-4",
		Kind:  "navigate",
		Title: "5-4  Deep Search",
		Hint:  "Three keys are scattered through the swamp. Search once, then keep repeating to collect them all before heading to the exit.",
		Cmds: []Cmd{
			{"/{pattern}", "search forward"},
			{"n", "repeat last search forward"},
		},
		Map: []string{
			"@....K.........K.............K.......$",
		},
		Solution: "/K<cr>nn/$<cr>",
	},

	// ───────────────────────── W6  Precision Peaks (count, F/t, edit count) ─────────────────────────
	{
		ID:    "6-1",
		Kind:  "navigate",
		Title: "6-1  Triple Word Hop",
		Hint:  "Counting words one at a time is slow. Prefix w with a number to leap several words at once.",
		Cmds: []Cmd{
			{"{N}w", "jump N words forward (e.g. 3w)"},
		},
		Map: []string{
			"@one two three K four five six $",
		},
		Solution: "4w4w",
	},
	{
		ID:    "6-2",
		Kind:  "navigate",
		Title: "6-2  Backward Find",
		Hint:  "The key is behind you this time. Use the backward find to reach it, then head for the exit.",
		Cmds: []Cmd{
			{"F{char}", "leap backward to that char on this line"},
		},
		Map: []string{
			"K.......@..........$",
		},
		Solution: "FK$",
	},
	{
		ID:    "6-3",
		Kind:  "navigate",
		Title: "6-3  Till the Key",
		Hint:  "A till-motion stops one cell short of its target. A signpost (X) marks the spot just past the key — aim for it to land exactly on the key.",
		Cmds: []Cmd{
			{"t{char}", "move up to (not onto) that char"},
		},
		Map: []string{
			"@..........KX.................$",
		},
		Solution: "tX$",
	},
	{
		ID:    "6-4",
		Kind:  "edit",
		Title: "6-4  Delete Two Words with d2w",
		Hint:  "Two unwanted words sit in a row. One delete-word command with a count clears them both.",
		Cmds: []Cmd{
			{"w", "move by word"},
			{"d{N}w", "delete N words at once"},
		},
		Map:      []string{"keep BAD WORDS here"},
		Target:   []string{"keep here"},
		Solution: "wd2w",
	},
	{
		ID:    "6-5",
		Kind:  "navigate",
		Title: "6-5  Boss: Count & Find",
		Hint:  "Combine a counted word-jump with a direct find to cross the canyon in one clean run.",
		Cmds: []Cmd{
			{"{N}w", "jump N words forward"},
			{"f{char}", "leap to that char"},
		},
		Map: []string{
			"@one two three K x.y.z.w.$",
		},
		Solution: "4wf$",
	},

	// ───────────────────────── W7  Visual Valley (visual mode + text objects) ─────────────────────────
	{
		ID:    "7-1",
		Kind:  "edit",
		Title: "7-1  Visual Delete",
		Hint:  "Select the unwanted stretch with Visual mode, then delete the whole selection at once.",
		Cmds: []Cmd{
			{"v", "enter Visual (charwise)"},
			{"E", "to end of WORD"},
			{"d", "delete the selection"},
		},
		Map:      []string{"keep THIS OUT keep"},
		Target:   []string{"keep  keep"},
		Solution: "wvEEd",
	},
	{
		ID:    "7-2",
		Kind:  "edit",
		Title: "7-2  Yank a Word",
		Hint:  "Duplicate the word right next to itself — yank a whole word (with its space) and paste it back before it.",
		Cmds: []Cmd{
			{"yaw", "yank 'a word' (incl. trailing space)"},
			{"P", "paste before cursor"},
		},
		Map:      []string{"dup ME now"},
		Target:   []string{"dup ME ME now"},
		Solution: "wyawP",
	},
	{
		ID:    "7-3",
		Kind:  "edit",
		Title: "7-3  Change Inner Word",
		Hint:  "Select the word with Visual mode's text object, then change it in one motion.",
		Cmds: []Cmd{
			{"viw", "select inner word (Visual)"},
			{"c", "change the selection"},
		},
		Map:      []string{"fix THIS please"},
		Target:   []string{"fix OK please"},
		Solution: "wviwcOK<esc>",
	},
	{
		ID:    "7-4",
		Kind:  "edit",
		Title: "7-4  Visual + Text Object Combo",
		Hint:  "Clear the first intruder with a Visual text-object delete, then change the second one with a text-object change.",
		Cmds: []Cmd{
			{"v", "enter Visual"},
			{"aw", "'a word' text object (extends selection)"},
			{"d", "delete the selection"},
			{"ciw", "change inner word"},
		},
		Map:      []string{"cut BAD here fix OLD there"},
		Target:   []string{"cut here fix NEW there"},
		Solution: "wvawdwwciwNEW<esc>",
	},

	// ───────────────────────── W8  Yank & Undo Ruins (xp, ddp, p/P, u/Ctrl-r) ─────────────────────────
	{
		ID:    "8-1",
		Kind:  "edit",
		Title: "8-1  Swap Chars with xp",
		Hint:  "Two letters are swapped. A classic one-two: delete the wrong one, then paste it back one step over.",
		Cmds: []Cmd{
			{"x", "delete char under cursor"},
			{"p", "paste after cursor"},
		},
		Map:      []string{"abcd"},
		Target:   []string{"bacd"},
		Solution: "xp",
	},
	{
		ID:    "8-2",
		Kind:  "edit",
		Title: "8-2  Swap Lines with ddp",
		Hint:  "The lines are in the wrong order. Cut one and drop it back in on the other side.",
		Cmds: []Cmd{
			{"dd", "delete the whole line"},
			{"p", "paste after cursor line"},
		},
		Map:      []string{"first", "second"},
		Target:   []string{"second", "first"},
		Solution: "ddp",
	},
	{
		ID:    "8-3",
		Kind:  "edit",
		Title: "8-3  Paste Above with P",
		Hint:  "You need a copy placed above, not below. Use the paste command that goes the other way.",
		Cmds: []Cmd{
			{"yy", "copy the line"},
			{"P", "paste ABOVE the current line"},
		},
		Map:      []string{"one", "two"},
		Target:   []string{"one", "one", "two"},
		Solution: "yyjP",
	},
	{
		ID:    "8-4",
		Kind:  "edit",
		Title: "8-4  Undo a Mistake",
		Hint:  "You deleted one line too many. Undo just the last change to bring it back.",
		Cmds: []Cmd{
			{"dd", "delete the whole line"},
			{"u", "undo the last change"},
		},
		Map:      []string{"keep", "BAD1", "BAD2"},
		Target:   []string{"keep", "BAD2"},
		Solution: "jddddu",
	},
	{
		ID:    "8-5",
		Kind:  "edit",
		Title: "8-5  Line Shuffle",
		Hint:  "One line is out of place at the top. Cut it and drop it back in at the bottom.",
		Cmds: []Cmd{
			{"dd", "delete the whole line"},
			{"j", "down a line"},
			{"p", "paste after cursor line"},
		},
		Map:      []string{"third", "first", "second"},
		Target:   []string{"first", "second", "third"},
		Solution: "ddjp",
	},
	{
		ID:    "8-6",
		Kind:  "edit",
		Title: "8-6  Boss: Everything At Once",
		Hint:  "One last gauntlet — delete a stray word, change another word, and swap two letters, all in a row.",
		Cmds: []Cmd{
			{"daw", "delete a word incl. its space"},
			{"ciw", "change the word under cursor"},
			{"xp", "swap two chars"},
		},
		Map:      []string{"one BAD two", "x = OLD", "abcd"},
		Target:   []string{"one two", "x = 99", "bacd"},
		Solution: "wdawj$ciw99<esc>j0xp",
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
