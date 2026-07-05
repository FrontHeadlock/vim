//go:build !js

package game

// levels_meta.go — 커리큘럼 레벨의 표시 전용 데이터(제목·힌트·명령 팔레트).
// 게임 규칙(par 산출·승리 판정)에는 전혀 쓰이지 않으므로 wasm 에 싣지 않는다:
// 웹은 tools/genmeta 가 이 데이터로 생성한 web/src/levels_meta.js 를 읽고,
// 이 파일 자체는 js 빌드 태그로 TinyGo 웹 빌드에서 제외된다. 데스크톱·테스트·
// 생성 도구(호스트 Go)만 이 데이터를 본다. 항목 누락은 TestLevelMetaComplete 가 잡는다.
var levelMeta = map[string]LevelMeta{
	"1-1": {
		Title: "1-1  First Steps",
		Hint:  "Grab the K (key), then reach the $ (exit). Use hjkl instead of the arrow keys to move.",
		Cmds: []Cmd{
			{"h", "← left one cell"},
			{"j", "↓ down one"},
			{"k", "↑ up one"},
			{"l", "→ right one"},
		},
	},
	"1-2": {
		Title: "1-2  Word Jumps",
		Hint:  "Don't crawl one cell at a time — jump word by word. Grab the key and head to $.",
		Cmds: []Cmd{
			{"w", "jump to start of next word"},
			{"b", "to start of previous word"},
			{"e", "to end of word"},
		},
	},
	"1-3": {
		Title: "1-3  Start & End of Line",
		Hint:  "The keys sit at both ends of the line. Use the keys that jump straight to the start/end.",
		Cmds: []Cmd{
			{"0", "to start of line"},
			{"$", "to end of line"},
			{"^", "to first non-blank char"},
			{"j k", "move down/up a line"},
		},
	},
	"1-4": {
		Title: "1-4  Find a Character",
		Hint:  "Use a find command to leap directly to a far-off character (K, $).",
		Cmds: []Cmd{
			{"f{char}", "leap to that char on this line (e.g. fK)"},
			{";", "repeat the last find once more"},
		},
	},
	"1-5": {
		Title: "1-5  Bug Hunt",
		Hint:  "Bugs (*) are killed by moving onto them and pressing the delete key. Dart up and down, clear them all, then grab the key and exit.",
		Cmds: []Cmd{
			{"gg", "warp to first line"},
			{"G", "warp to last line"},
			{"x", "delete char under cursor (kill bug)"},
			{"h j k l", "move one cell"},
		},
	},
	"1-6": {
		Title: "1-6  Bonus: No Counting",
		Hint:  "This line is long and one key sits behind you. Counting cells with hjkl is slow — F/f leap straight to a landmark, in either direction.",
		Cmds: []Cmd{
			{"f{char}", "leap forward to that char"},
			{"F{char}", "leap backward to that char"},
		},
	},
	"2-1": {
		Title: "2-1  Repeat Jumps",
		Hint:  "There's a key that repeats your last find. Find once, then repeat to skip across the dots.",
		Cmds: []Cmd{
			{"f{char}", "jump to that char (e.g. f.)"},
			{";", "repeat find (forward)"},
			{",", "repeat find (backward)"},
		},
	},
	"2-2": {
		Title: "2-2  Jump by Number",
		Hint:  "Jumping by line number makes vertical travel fast. Clear the bugs and reach the key/exit.",
		Cmds: []Cmd{
			{"{N}G", "jump to line N (e.g. 4G)"},
			{"gg", "to first line"},
			{"G", "to last line"},
			{"x", "delete bug"},
		},
	},
	"2-3": {
		Title: "2-3  Canyon Run",
		Hint:  "Use everything you've learned! Grab the keys at both ends and find the fastest path to the exit.",
		Cmds: []Cmd{
			{"w", "word jump"},
			{"f{char}", "jump to char"},
			{"0 $", "start / end of line"},
			{"gg G", "first / last line"},
		},
	},
	"3-1": {
		Title: "3-1  Fix Typos with x",
		Hint:  "Too many letters (a typo). Move onto the extra letters and delete them to match the target on the right.",
		Cmds: []Cmd{
			{"f{char}", "jump to the typo"},
			{"x", "delete one char under cursor"},
			{"h l", "move left/right"},
		},
	},
	"3-2": {
		Title: "3-2  Delete Lines with dd",
		Hint:  "Some lines aren't needed. Move to them and delete whole lines to leave only the target.",
		Cmds: []Cmd{
			{"j", "down a line"},
			{"k", "up a line"},
			{"dd", "delete the whole current line"},
		},
	},
	"3-3": {
		Title: "3-3  Delete Words with dw",
		Hint:  "An unwanted word is wedged in the sentence. Use 'delete operator d + word motion w' to remove it whole.",
		Cmds: []Cmd{
			{"w", "move by word"},
			{"d", "delete operator (takes a motion)"},
			{"dw", "delete one word from cursor"},
		},
	},
	"3-4": {
		Title: "3-4  Append Text with A",
		Hint:  "You need to append text to the end of the line. Enter insert mode, type, then press Esc to leave it.",
		Cmds: []Cmd{
			{"A", "start typing at end of line (Insert)"},
			{"(type)", "enter your text"},
			{"Esc", "leave insert → Normal"},
		},
	},
	"3-5": {
		Title: "3-5  Replace a Word with cw",
		Hint:  "Just one word needs swapping. The 'change operator c' deletes and drops you straight into insert.",
		Cmds: []Cmd{
			{"w", "move to the word"},
			{"cw", "delete the word and start typing"},
			{"Esc", "leave insert"},
		},
	},
	"3-6": {
		Title: "3-6  Repeat with .",
		Hint:  "All three words must become the same word. Use the key that repeats your last change to fly through it.",
		Cmds: []Cmd{
			{"cw", "delete word and type"},
			{"Esc", "leave insert"},
			{"w", "to next word"},
			{".", "repeat the last change"},
		},
	},
	"4-1": {
		Title: "4-1  Delete a Whole Word with daw",
		Hint:  "Delete a single word, trailing space and all. Use the 'a word' text object with the delete operator.",
		Cmds: []Cmd{
			{"w", "move to the word"},
			{"daw", "delete a word incl. its space"},
		},
	},
	"4-2": {
		Title: "4-2  Change Inside ( ) with ci(",
		Hint:  "Only the contents inside the ( ) need changing. Use the 'inner' text object to target just inside the parentheses.",
		Cmds: []Cmd{
			{"f{char}", "move to ( (e.g. f( )"},
			{"ci(", "change inside the parentheses"},
			{"Esc", "leave insert"},
		},
	},
	"4-3": {
		Title: "4-3  Change Inside \" \" with ci\"",
		Hint:  "Only the text inside the \" \" needs changing. Use the text object that selects inside the quotes.",
		Cmds: []Cmd{
			{"f{char}", "move to \""},
			{"ci\"", "change inside the quotes"},
			{"Esc", "leave insert"},
		},
	},
	"4-4": {
		Title: "4-4  Duplicate a Line with yy/p",
		Hint:  "You need one more copy of the line. Copy (yank) the line and paste it to duplicate.",
		Cmds: []Cmd{
			{"yy", "copy (yank) the line"},
			{"p", "paste the copied line below"},
		},
	},
	"4-5": {
		Title: "4-5  Boss: ciw + . Combo",
		Hint:  "Turn both OLD into 42! Change one line, then use the 'repeat' key to do the next line the same way.",
		Cmds: []Cmd{
			{"$", "to end of line"},
			{"ciw", "change the word under cursor"},
			{"Esc", "leave insert"},
			{"j", "down a line"},
			{".", "repeat the last change"},
		},
	},
	"5-1": {
		Title: "5-1  Search Swamp",
		Hint:  "The key is buried far down the swamp. Search for it instead of crawling — then search for the exit too.",
		Cmds: []Cmd{
			{"/{pattern}", "search forward for text"},
			{"<cr>", "confirm search"},
		},
	},
	"5-2": {
		Title: "5-2  Twin Keys",
		Hint:  "Two keys share the swamp. Search once, then repeat the search to reach the second.",
		Cmds: []Cmd{
			{"/{pattern}", "search forward"},
			{"n", "repeat last search forward"},
		},
	},
	"5-3": {
		Title: "5-3  Backtrack",
		Hint:  "You'll overshoot the key if you rush to the exit. Search backward to retrieve what you missed, then forward again.",
		Cmds: []Cmd{
			{"/{pattern}", "search forward"},
			{"?{pattern}", "search backward"},
		},
	},
	"5-4": {
		Title: "5-4  Deep Search",
		Hint:  "Three keys are scattered through the swamp. Search once, then keep repeating to collect them all before heading to the exit.",
		Cmds: []Cmd{
			{"/{pattern}", "search forward"},
			{"n", "repeat last search forward"},
		},
	},
	"6-1": {
		Title: "6-1  Triple Word Hop",
		Hint:  "Counting words one at a time is slow. Prefix w with a number to leap several words at once.",
		Cmds: []Cmd{
			{"{N}w", "jump N words forward (e.g. 3w)"},
		},
	},
	"6-2": {
		Title: "6-2  Backward Find",
		Hint:  "The key is behind you this time. Use the backward find to reach it, then head for the exit.",
		Cmds: []Cmd{
			{"F{char}", "leap backward to that char on this line"},
		},
	},
	"6-3": {
		Title: "6-3  Till the Key",
		Hint:  "A till-motion stops one cell short of its target. A signpost (X) marks the spot just past the key — aim for it to land exactly on the key.",
		Cmds: []Cmd{
			{"t{char}", "move up to (not onto) that char"},
		},
	},
	"6-4": {
		Title: "6-4  Delete Two Words with d2w",
		Hint:  "Two unwanted words sit in a row. One delete-word command with a count clears them both.",
		Cmds: []Cmd{
			{"w", "move by word"},
			{"d{N}w", "delete N words at once"},
		},
	},
	"6-5": {
		Title: "6-5  Boss: Count & Find",
		Hint:  "Combine a counted word-jump with a direct find to cross the canyon in one clean run.",
		Cmds: []Cmd{
			{"{N}w", "jump N words forward"},
			{"f{char}", "leap to that char"},
		},
	},
	"7-1": {
		Title: "7-1  Visual Delete",
		Hint:  "Select the unwanted stretch with Visual mode, then delete the whole selection at once.",
		Cmds: []Cmd{
			{"v", "enter Visual (charwise)"},
			{"E", "to end of WORD"},
			{"d", "delete the selection"},
		},
	},
	"7-2": {
		Title: "7-2  Yank a Word",
		Hint:  "Duplicate the word right next to itself — yank a whole word (with its space) and paste it back before it.",
		Cmds: []Cmd{
			{"yaw", "yank 'a word' (incl. trailing space)"},
			{"P", "paste before cursor"},
		},
	},
	"7-3": {
		Title: "7-3  Change Inner Word",
		Hint:  "Select the word with Visual mode's text object, then change it in one motion.",
		Cmds: []Cmd{
			{"viw", "select inner word (Visual)"},
			{"c", "change the selection"},
		},
	},
	"7-4": {
		Title: "7-4  Visual + Text Object Combo",
		Hint:  "Clear the first intruder with a Visual text-object delete, then change the second one with a text-object change.",
		Cmds: []Cmd{
			{"v", "enter Visual"},
			{"aw", "'a word' text object (extends selection)"},
			{"d", "delete the selection"},
			{"ciw", "change inner word"},
		},
	},
	"8-1": {
		Title: "8-1  Swap Chars with xp",
		Hint:  "Two letters are swapped. A classic one-two: delete the wrong one, then paste it back one step over.",
		Cmds: []Cmd{
			{"x", "delete char under cursor"},
			{"p", "paste after cursor"},
		},
	},
	"8-2": {
		Title: "8-2  Swap Lines with ddp",
		Hint:  "The lines are in the wrong order. Cut one and drop it back in on the other side.",
		Cmds: []Cmd{
			{"dd", "delete the whole line"},
			{"p", "paste after cursor line"},
		},
	},
	"8-3": {
		Title: "8-3  Paste Above with P",
		Hint:  "You need a copy placed above, not below. Use the paste command that goes the other way.",
		Cmds: []Cmd{
			{"yy", "copy the line"},
			{"P", "paste ABOVE the current line"},
		},
	},
	"8-4": {
		Title: "8-4  Undo a Mistake",
		Hint:  "You deleted one line too many. Undo just the last change to bring it back.",
		Cmds: []Cmd{
			{"dd", "delete the whole line"},
			{"u", "undo the last change"},
		},
	},
	"8-5": {
		Title: "8-5  Line Shuffle",
		Hint:  "One line is out of place at the top. Cut it and drop it back in at the bottom.",
		Cmds: []Cmd{
			{"dd", "delete the whole line"},
			{"j", "down a line"},
			{"p", "paste after cursor line"},
		},
	},
	"8-6": {
		Title: "8-6  Boss: Everything At Once",
		Hint:  "One last gauntlet — delete a stray word, change another word, and swap two letters, all in a row.",
		Cmds: []Cmd{
			{"daw", "delete a word incl. its space"},
			{"ciw", "change the word under cursor"},
			{"xp", "swap two chars"},
		},
	},
}

// MetaFor 는 레벨 ID 의 표시 데이터를 돌려준다(없으면 zero value).
func MetaFor(id string) LevelMeta { return levelMeta[id] }
