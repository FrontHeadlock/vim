#!/usr/bin/env bash
# VimQuest 웹 빌드 스크립트 — TinyGo 로 WebAssembly 로 컴파일하고 web/ 에 배치한다.
# Phase 4 L2부터 Ebiten 대신 TinyGo + canvas 2D 렌더러(web/renderer.js)를 쓴다.
# 데스크톱(Ebiten) 빌드는 build_desktop.sh 를 쓸 것.
set -euo pipefail

cd "$(dirname "$0")"

# 웹 페이로드 gzip 합계 상한(bytes). PLAN_PHASE4.md L2 DoD — 100KB.
SIZE_BUDGET_BYTES=102400

if ! command -v tinygo >/dev/null 2>&1; then
  echo "✗ tinygo 를 찾을 수 없습니다. 설치: brew tap tinygo-org/tools && brew install tinygo-org/tools/tinygo" >&2
  exit 1
fi

echo "▶ TinyGo WASM 빌드 중..."
# -gc=leaking: GC 를 아예 안 돌리고 할당만 한다(gzip ~98KB→~71KB, 실측).
# 한 브라우저 탭의 짧은 플레이 세션 동안 메모리를 회수 안 해도 되는 트레이드오프 —
# :drill 이 문제를 계속 새로 만들어도 이 정도 규모(격자 100칸)로는 무해하다.
# -scheduler=none 은 시도하지 않는다: wasm_exec.js 의 콜백 재개(resume export)가
# 고루틴 스케줄러에 의존해서, 끄면 vqInput 반복 호출이 즉시 깨진다(실측 확인됨).
tinygo build -target wasm -opt=z -gc=leaking -no-debug -o web/game.wasm .

echo "▶ wasm_exec.js 동기화..."
cp "$(tinygo env TINYGOROOT)/targets/wasm_exec.js" web/wasm_exec.js

echo "▶ 페이로드 크기 확인 중..."
total_gzip=$(cat web/game.wasm web/wasm_exec.js web/renderer.js web/glue.js web/index.html | gzip -9 | wc -c | tr -d ' ')
wasm_bytes=$(wc -c < web/game.wasm | tr -d ' ')

echo "  game.wasm      : $((wasm_bytes / 1024)) KB (raw)"
echo "  gzip 합계      : $((total_gzip / 1024)) KB / $((SIZE_BUDGET_BYTES / 1024)) KB 예산"

if [ "$total_gzip" -gt "$SIZE_BUDGET_BYTES" ]; then
  echo "✗ 크기 예산 초과: ${total_gzip} bytes > ${SIZE_BUDGET_BYTES} bytes" >&2
  echo "  (encoding/json 등 무거운 표준 라이브러리가 다시 들어왔는지 확인하세요 — SPEC_PHASE4.md §0.1)" >&2
  exit 1
fi

echo "✅ 빌드 완료 → web/game.wasm (${wasm_bytes} bytes, gzip 합계 ${total_gzip} bytes)"
echo
echo "실행:  cd web && python3 -m http.server 8765"
echo "접속:  http://localhost:8765/"
