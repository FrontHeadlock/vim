// 생성 파일 — 직접 편집 금지. 원본: internal/game/levels_meta.go
// 재생성: go run ./tools/genmeta > web/src/levels_meta.js (build.sh 가 자동 실행)
'use strict';
const LEVEL_META = {
  "1-1": {
    "title": "1-1  First Steps",
    "hint": "Grab the K (key), then reach the $ (exit). Use hjkl instead of the arrow keys to move.",
    "cmds": [
      {
        "k": "h",
        "d": "← left one cell"
      },
      {
        "k": "j",
        "d": "↓ down one"
      },
      {
        "k": "k",
        "d": "↑ up one"
      },
      {
        "k": "l",
        "d": "→ right one"
      }
    ]
  },
  "1-2": {
    "title": "1-2  Word Jumps",
    "hint": "Don't crawl one cell at a time — jump word by word. Grab the key and head to $.",
    "cmds": [
      {
        "k": "w",
        "d": "jump to start of next word"
      },
      {
        "k": "b",
        "d": "to start of previous word"
      },
      {
        "k": "e",
        "d": "to end of word"
      }
    ]
  },
  "1-3": {
    "title": "1-3  Start \u0026 End of Line",
    "hint": "The keys sit at both ends of the line. Use the keys that jump straight to the start/end.",
    "cmds": [
      {
        "k": "0",
        "d": "to start of line"
      },
      {
        "k": "$",
        "d": "to end of line"
      },
      {
        "k": "^",
        "d": "to first non-blank char"
      },
      {
        "k": "j k",
        "d": "move down/up a line"
      }
    ]
  },
  "1-4": {
    "title": "1-4  Find a Character",
    "hint": "Use a find command to leap directly to a far-off character (K, $).",
    "cmds": [
      {
        "k": "f{char}",
        "d": "leap to that char on this line (e.g. fK)"
      },
      {
        "k": ";",
        "d": "repeat the last find once more"
      }
    ]
  },
  "1-5": {
    "title": "1-5  Bug Hunt",
    "hint": "Bugs (*) are killed by moving onto them and pressing the delete key. Dart up and down, clear them all, then grab the key and exit.",
    "cmds": [
      {
        "k": "gg",
        "d": "warp to first line"
      },
      {
        "k": "G",
        "d": "warp to last line"
      },
      {
        "k": "x",
        "d": "delete char under cursor (kill bug)"
      },
      {
        "k": "h j k l",
        "d": "move one cell"
      }
    ]
  },
  "1-6": {
    "title": "1-6  Bonus: No Counting",
    "hint": "This line is long and one key sits behind you. Counting cells with hjkl is slow — F/f leap straight to a landmark, in either direction.",
    "cmds": [
      {
        "k": "f{char}",
        "d": "leap forward to that char"
      },
      {
        "k": "F{char}",
        "d": "leap backward to that char"
      }
    ]
  },
  "2-1": {
    "title": "2-1  Repeat Jumps",
    "hint": "There's a key that repeats your last find. Find once, then repeat to skip across the dots.",
    "cmds": [
      {
        "k": "f{char}",
        "d": "jump to that char (e.g. f.)"
      },
      {
        "k": ";",
        "d": "repeat find (forward)"
      },
      {
        "k": ",",
        "d": "repeat find (backward)"
      }
    ]
  },
  "2-2": {
    "title": "2-2  Jump by Number",
    "hint": "Jumping by line number makes vertical travel fast. Clear the bugs and reach the key/exit.",
    "cmds": [
      {
        "k": "{N}G",
        "d": "jump to line N (e.g. 4G)"
      },
      {
        "k": "gg",
        "d": "to first line"
      },
      {
        "k": "G",
        "d": "to last line"
      },
      {
        "k": "x",
        "d": "delete bug"
      }
    ]
  },
  "2-3": {
    "title": "2-3  Canyon Run",
    "hint": "Use everything you've learned! Grab the keys at both ends and find the fastest path to the exit.",
    "cmds": [
      {
        "k": "w",
        "d": "word jump"
      },
      {
        "k": "f{char}",
        "d": "jump to char"
      },
      {
        "k": "0 $",
        "d": "start / end of line"
      },
      {
        "k": "gg G",
        "d": "first / last line"
      }
    ]
  },
  "3-1": {
    "title": "3-1  Fix Typos with x",
    "hint": "Too many letters (a typo). Move onto the extra letters and delete them to match the target on the right.",
    "cmds": [
      {
        "k": "f{char}",
        "d": "jump to the typo"
      },
      {
        "k": "x",
        "d": "delete one char under cursor"
      },
      {
        "k": "h l",
        "d": "move left/right"
      }
    ]
  },
  "3-2": {
    "title": "3-2  Delete Lines with dd",
    "hint": "Some lines aren't needed. Move to them and delete whole lines to leave only the target.",
    "cmds": [
      {
        "k": "j",
        "d": "down a line"
      },
      {
        "k": "k",
        "d": "up a line"
      },
      {
        "k": "dd",
        "d": "delete the whole current line"
      }
    ]
  },
  "3-3": {
    "title": "3-3  Delete Words with dw",
    "hint": "An unwanted word is wedged in the sentence. Use 'delete operator d + word motion w' to remove it whole.",
    "cmds": [
      {
        "k": "w",
        "d": "move by word"
      },
      {
        "k": "d",
        "d": "delete operator (takes a motion)"
      },
      {
        "k": "dw",
        "d": "delete one word from cursor"
      }
    ]
  },
  "3-4": {
    "title": "3-4  Append Text with A",
    "hint": "You need to append text to the end of the line. Enter insert mode, type, then press Esc to leave it.",
    "cmds": [
      {
        "k": "A",
        "d": "start typing at end of line (Insert)"
      },
      {
        "k": "(type)",
        "d": "enter your text"
      },
      {
        "k": "Esc",
        "d": "leave insert → Normal"
      }
    ]
  },
  "3-5": {
    "title": "3-5  Replace a Word with cw",
    "hint": "Just one word needs swapping. The 'change operator c' deletes and drops you straight into insert.",
    "cmds": [
      {
        "k": "w",
        "d": "move to the word"
      },
      {
        "k": "cw",
        "d": "delete the word and start typing"
      },
      {
        "k": "Esc",
        "d": "leave insert"
      }
    ]
  },
  "3-6": {
    "title": "3-6  Repeat with .",
    "hint": "All three words must become the same word. Use the key that repeats your last change to fly through it.",
    "cmds": [
      {
        "k": "cw",
        "d": "delete word and type"
      },
      {
        "k": "Esc",
        "d": "leave insert"
      },
      {
        "k": "w",
        "d": "to next word"
      },
      {
        "k": ".",
        "d": "repeat the last change"
      }
    ]
  },
  "4-1": {
    "title": "4-1  Delete a Whole Word with daw",
    "hint": "Delete a single word, trailing space and all. Use the 'a word' text object with the delete operator.",
    "cmds": [
      {
        "k": "w",
        "d": "move to the word"
      },
      {
        "k": "daw",
        "d": "delete a word incl. its space"
      }
    ]
  },
  "4-2": {
    "title": "4-2  Change Inside ( ) with ci(",
    "hint": "Only the contents inside the ( ) need changing. Use the 'inner' text object to target just inside the parentheses.",
    "cmds": [
      {
        "k": "f{char}",
        "d": "move to ( (e.g. f( )"
      },
      {
        "k": "ci(",
        "d": "change inside the parentheses"
      },
      {
        "k": "Esc",
        "d": "leave insert"
      }
    ]
  },
  "4-3": {
    "title": "4-3  Change Inside \" \" with ci\"",
    "hint": "Only the text inside the \" \" needs changing. Use the text object that selects inside the quotes.",
    "cmds": [
      {
        "k": "f{char}",
        "d": "move to \""
      },
      {
        "k": "ci\"",
        "d": "change inside the quotes"
      },
      {
        "k": "Esc",
        "d": "leave insert"
      }
    ]
  },
  "4-4": {
    "title": "4-4  Duplicate a Line with yy/p",
    "hint": "You need one more copy of the line. Copy (yank) the line and paste it to duplicate.",
    "cmds": [
      {
        "k": "yy",
        "d": "copy (yank) the line"
      },
      {
        "k": "p",
        "d": "paste the copied line below"
      }
    ]
  },
  "4-5": {
    "title": "4-5  Boss: ciw + . Combo",
    "hint": "Turn both OLD into 42! Change one line, then use the 'repeat' key to do the next line the same way.",
    "cmds": [
      {
        "k": "$",
        "d": "to end of line"
      },
      {
        "k": "ciw",
        "d": "change the word under cursor"
      },
      {
        "k": "Esc",
        "d": "leave insert"
      },
      {
        "k": "j",
        "d": "down a line"
      },
      {
        "k": ".",
        "d": "repeat the last change"
      }
    ]
  },
  "5-1": {
    "title": "5-1  Search Swamp",
    "hint": "The key is buried far down the swamp. Search for it instead of crawling — then search for the exit too.",
    "cmds": [
      {
        "k": "/{pattern}",
        "d": "search forward for text"
      },
      {
        "k": "\u003ccr\u003e",
        "d": "confirm search"
      }
    ]
  },
  "5-2": {
    "title": "5-2  Twin Keys",
    "hint": "Two keys share the swamp. Search once, then repeat the search to reach the second.",
    "cmds": [
      {
        "k": "/{pattern}",
        "d": "search forward"
      },
      {
        "k": "n",
        "d": "repeat last search forward"
      }
    ]
  },
  "5-3": {
    "title": "5-3  Backtrack",
    "hint": "You'll overshoot the key if you rush to the exit. Search backward to retrieve what you missed, then forward again.",
    "cmds": [
      {
        "k": "/{pattern}",
        "d": "search forward"
      },
      {
        "k": "?{pattern}",
        "d": "search backward"
      }
    ]
  },
  "5-4": {
    "title": "5-4  Deep Search",
    "hint": "Three keys are scattered through the swamp. Search once, then keep repeating to collect them all before heading to the exit.",
    "cmds": [
      {
        "k": "/{pattern}",
        "d": "search forward"
      },
      {
        "k": "n",
        "d": "repeat last search forward"
      }
    ]
  },
  "6-1": {
    "title": "6-1  Triple Word Hop",
    "hint": "Counting words one at a time is slow. Prefix w with a number to leap several words at once.",
    "cmds": [
      {
        "k": "{N}w",
        "d": "jump N words forward (e.g. 3w)"
      }
    ]
  },
  "6-2": {
    "title": "6-2  Backward Find",
    "hint": "The key is behind you this time. Use the backward find to reach it, then head for the exit.",
    "cmds": [
      {
        "k": "F{char}",
        "d": "leap backward to that char on this line"
      }
    ]
  },
  "6-3": {
    "title": "6-3  Till the Key",
    "hint": "A till-motion stops one cell short of its target. A signpost (X) marks the spot just past the key — aim for it to land exactly on the key.",
    "cmds": [
      {
        "k": "t{char}",
        "d": "move up to (not onto) that char"
      }
    ]
  },
  "6-4": {
    "title": "6-4  Delete Two Words with d2w",
    "hint": "Two unwanted words sit in a row. One delete-word command with a count clears them both.",
    "cmds": [
      {
        "k": "w",
        "d": "move by word"
      },
      {
        "k": "d{N}w",
        "d": "delete N words at once"
      }
    ]
  },
  "6-5": {
    "title": "6-5  Boss: Count \u0026 Find",
    "hint": "Combine a counted word-jump with a direct find to cross the canyon in one clean run.",
    "cmds": [
      {
        "k": "{N}w",
        "d": "jump N words forward"
      },
      {
        "k": "f{char}",
        "d": "leap to that char"
      }
    ]
  },
  "7-1": {
    "title": "7-1  Visual Delete",
    "hint": "Select the unwanted stretch with Visual mode, then delete the whole selection at once.",
    "cmds": [
      {
        "k": "v",
        "d": "enter Visual (charwise)"
      },
      {
        "k": "E",
        "d": "to end of WORD"
      },
      {
        "k": "d",
        "d": "delete the selection"
      }
    ]
  },
  "7-2": {
    "title": "7-2  Yank a Word",
    "hint": "Duplicate the word right next to itself — yank a whole word (with its space) and paste it back before it.",
    "cmds": [
      {
        "k": "yaw",
        "d": "yank 'a word' (incl. trailing space)"
      },
      {
        "k": "P",
        "d": "paste before cursor"
      }
    ]
  },
  "7-3": {
    "title": "7-3  Change Inner Word",
    "hint": "Select the word with Visual mode's text object, then change it in one motion.",
    "cmds": [
      {
        "k": "viw",
        "d": "select inner word (Visual)"
      },
      {
        "k": "c",
        "d": "change the selection"
      }
    ]
  },
  "7-4": {
    "title": "7-4  Visual + Text Object Combo",
    "hint": "Clear the first intruder with a Visual text-object delete, then change the second one with a text-object change.",
    "cmds": [
      {
        "k": "v",
        "d": "enter Visual"
      },
      {
        "k": "aw",
        "d": "'a word' text object (extends selection)"
      },
      {
        "k": "d",
        "d": "delete the selection"
      },
      {
        "k": "ciw",
        "d": "change inner word"
      }
    ]
  },
  "8-1": {
    "title": "8-1  Swap Chars with xp",
    "hint": "Two letters are swapped. A classic one-two: delete the wrong one, then paste it back one step over.",
    "cmds": [
      {
        "k": "x",
        "d": "delete char under cursor"
      },
      {
        "k": "p",
        "d": "paste after cursor"
      }
    ]
  },
  "8-2": {
    "title": "8-2  Swap Lines with ddp",
    "hint": "The lines are in the wrong order. Cut one and drop it back in on the other side.",
    "cmds": [
      {
        "k": "dd",
        "d": "delete the whole line"
      },
      {
        "k": "p",
        "d": "paste after cursor line"
      }
    ]
  },
  "8-3": {
    "title": "8-3  Paste Above with P",
    "hint": "You need a copy placed above, not below. Use the paste command that goes the other way.",
    "cmds": [
      {
        "k": "yy",
        "d": "copy the line"
      },
      {
        "k": "P",
        "d": "paste ABOVE the current line"
      }
    ]
  },
  "8-4": {
    "title": "8-4  Undo a Mistake",
    "hint": "You deleted one line too many. Undo just the last change to bring it back.",
    "cmds": [
      {
        "k": "dd",
        "d": "delete the whole line"
      },
      {
        "k": "u",
        "d": "undo the last change"
      }
    ]
  },
  "8-5": {
    "title": "8-5  Line Shuffle",
    "hint": "One line is out of place at the top. Cut it and drop it back in at the bottom.",
    "cmds": [
      {
        "k": "dd",
        "d": "delete the whole line"
      },
      {
        "k": "j",
        "d": "down a line"
      },
      {
        "k": "p",
        "d": "paste after cursor line"
      }
    ]
  },
  "8-6": {
    "title": "8-6  Boss: Everything At Once",
    "hint": "One last gauntlet — delete a stray word, change another word, and swap two letters, all in a row.",
    "cmds": [
      {
        "k": "daw",
        "d": "delete a word incl. its space"
      },
      {
        "k": "ciw",
        "d": "change the word under cursor"
      },
      {
        "k": "xp",
        "d": "swap two chars"
      }
    ]
  }
};
