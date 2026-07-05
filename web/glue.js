// web/glue.js — 입력 캡처 + 렌더러 구동. Go 익스포트(vqInput/vqState/vqTick)에
// 의존하는 부분은 vqInit() 안에 있고, wasm 로드가 끝난 뒤(go.run 이후) 호출된다.
'use strict';

// v1(JSON, encoding/json 시절)→v2(수제 코덱) 마이그레이션 — wasm 로드보다 먼저
// 실행돼야 한다. 이 스크립트가 <script src> 로 wasm 로더보다 앞서 로드되므로
// 문서 순서상 자연히 만족된다. Go 바이너리에 encoding/json 을 다시 들이지
// 않기 위해 브라우저 네이티브 JSON.parse 로 처리한다.
(function migrateProgress() {
  try {
    const v1 = localStorage.getItem('vimquest.v1');
    if (!v1 || localStorage.getItem('vimquest.v2')) return;
    const parsed = JSON.parse(v1);
    const parts = [];
    for (const id in parsed) {
      const p = parsed[id];
      parts.push(`${id}:${p.Unlocked ? 1 : 0},${p.BestStrokes || 0},${p.Stars || 0}`);
    }
    localStorage.setItem('vimquest.v2', parts.join(';'));
    localStorage.removeItem('vimquest.v1');
  } catch (e) {
    // best-effort — 실패해도 빈 진행으로 시작(게임 진행을 막지 않음)
  }
})();

let vqRenderer = null;
let vqTickRunning = false;

// effects/bell 이 살아있는 동안만 rAF 로 vqTick 을 돈다 — 상시 60fps 루프 없음.
function vqStartTick() {
  if (vqTickRunning) return;
  vqTickRunning = true;
  function loop() {
    const st = vqTick();
    vqRenderer.draw(st);
    if (st.effectsAlive) {
      requestAnimationFrame(loop);
    } else {
      vqTickRunning = false;
    }
  }
  requestAnimationFrame(loop);
}

function vqHandleKey(e) {
  if (e.isComposing || e.keyCode === 229) return; // 한글 IME 조합 중 — 배너가 안내
  let tok = null;
  if (e.key === 'Enter') tok = '<cr>';
  else if (e.key === 'Escape') tok = '<esc>';
  else if (e.key === 'Backspace') tok = '<bs>';
  else if (e.ctrlKey && e.key.toLowerCase() === 'r') tok = '<c-r>';
  else if (e.ctrlKey || e.altKey || e.metaKey) return; // 다른 조합키는 브라우저 기본 동작에 맡긴다
  else if (e.key.length === 1) tok = e.key;
  else return; // Tab/Shift 단독 등은 무시

  e.preventDefault();
  const st = vqInput(tok);
  vqRenderer.draw(st);
  if (st.effectsAlive) vqStartTick();
}

// vqInit 은 wasm 로드가 끝난 뒤(go.run 이후) 호출한다 — vqInput/vqState/vqTick
// 은 Go 쪽 main()(web_js.go)이 실행돼야 전역에 등록된다.
function vqInit() {
  const canvas = document.getElementById('game');
  vqRenderer = new Renderer(canvas);
  window.addEventListener('keydown', vqHandleKey);
  const st = vqState();
  vqRenderer.draw(st);
  if (st.effectsAlive) vqStartTick();
}
