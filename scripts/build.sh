#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# 웹 페이로드 gzip 합계 상한(bytes). vqInput/vqTick 추가로 125KB로 상향.
SIZE_BUDGET_BYTES=128000

if ! command -v tinygo >/dev/null 2>&1; then
  echo "✗ tinygo 를 찾을 수 없습니다. 설치: brew tap tinygo-org/tools && brew install tinygo-org/tools/tinygo" >&2
  exit 1
fi

mkdir -p web/dist

echo "▶ 레벨 메타(JS) 생성 중..."
# 커리큘럼 표시 데이터(제목·힌트·명령 팔레트)는 wasm 에 싣지 않는다 —
# levels_meta.go(!js)가 단일 진실이고, 여기서 JS 테이블로 변환해 웹이 읽는다.
go run ./tools/genmeta > web/src/levels_meta.js

echo "▶ TinyGo WASM 빌드 중..."
# -gc=leaking: GC 를 아예 안 돌리고 할당만 한다(gzip ~98KB→~71KB, 실측).
# 한 브라우저 탭의 짧은 플레이 세션 동안 메모리를 회수 안 해도 되는 트레이드오프 —
# :drill 이 문제를 계속 새로 만들어도 이 정도 규모(격자 100칸)로는 무해하다.
# -scheduler=none 은 시도하지 않는다: wasm_exec.js 의 콜백 재개(resume export)가
# 고루틴 스케줄러에 의존해서, 끄면 vqInput 반복 호출이 즉시 깨진다(실측 확인됨).
# -panic=trap: panic 시 메시지 출력 없이 즉시 trap(gzip ~116KB→~93KB, 실측) —
# 엔진이 임의 입력에 패닉하지 않는다는 걸 fuzz 하네스(FuzzEditorNeverPanics,
# CI 에서 상시 실행)가 기계적으로 보증하므로 무해한 트레이드오프로 판단.
tinygo build -target wasm -opt=z -gc=leaking -panic=trap -no-debug -o web/dist/game.wasm ./cmd/web

echo "▶ wasm_exec.js 동기화..."
cp "$(tinygo env TINYGOROOT)/targets/wasm_exec.js" web/dist/wasm_exec.js

echo "▶ 페이로드 크기 확인 중..."
total_gzip=$(cat web/dist/game.wasm web/dist/wasm_exec.js web/src/levels_meta.js web/src/levels_meta_ko.js web/src/renderer.js web/src/glue.js web/src/index.html | gzip -9 | wc -c | tr -d ' ')
wasm_bytes=$(wc -c < web/dist/game.wasm | tr -d ' ')

echo "  game.wasm      : $((wasm_bytes / 1024)) KB (raw)"
echo "  gzip 합계      : $((total_gzip / 1024)) KB / $((SIZE_BUDGET_BYTES / 1024)) KB 예산"

if [ "$total_gzip" -gt "$SIZE_BUDGET_BYTES" ]; then
  echo "✗ 크기 예산 초과: ${total_gzip} bytes > ${SIZE_BUDGET_BYTES} bytes" >&2
  echo "  (encoding/json 등 무거운 표준 라이브러리가 다시 들어왔는지 확인하세요 — SPEC_PHASE4.md §0.1)" >&2
  exit 1
fi

echo "✅ 빌드 완료 → web/dist/game.wasm (${wasm_bytes} bytes, gzip 합계 ${total_gzip} bytes)"
echo
echo "실행:  python3 -m http.server 8765 --directory web"
echo "접속:  http://localhost:8765/src/"
