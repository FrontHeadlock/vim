# VimQuest

Learn Vim by playing — a free, retro browser game. Collect keys, explore the text world, and pick up **real Vim commands** (motions, operators, text objects) across 19 levels in 4 worlds.

![VimQuest](docs/screenshot.png)

## Play

```bash
./build.sh                              # compile Go → WebAssembly
cd web && python3 -m http.server 8765   # then open http://localhost:8765/
```

> Keys only work in English input mode.

## Built with

Go + [Ebitengine](https://ebitengine.org/) compiled to WebAssembly. The gameplay is a small hand-written Vim engine (`editor.go`); the UI is plain HTML/CSS. See [`GAME_DESIGN.md`](GAME_DESIGN.md) for the design.
