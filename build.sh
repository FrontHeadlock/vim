#!/usr/bin/env bash
# VimQuest 빌드 스크립트 — Go 코드를 WebAssembly 로 컴파일하고 web/ 에 배치한다.
set -euo pipefail

cd "$(dirname "$0")"

echo "▶ WASM 빌드 중..."
GOOS=js GOARCH=wasm go build -o web/game.wasm .

echo "▶ wasm_exec.js 동기화..."
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js

echo "✅ 빌드 완료 → web/game.wasm ($(wc -c < web/game.wasm | tr -d ' ') bytes)"
echo
echo "실행:  cd web && python3 -m http.server 8765"
echo "접속:  http://localhost:8765/"
