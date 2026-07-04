package main

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
}
