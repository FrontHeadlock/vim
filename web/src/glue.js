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
let vqLastState = null; // COPY 버튼이 읽는, 마지막으로 그린 스냅샷

// vqDraw 는 렌더러에 그리는 동시에 vqLastState 갱신·사이드패널 동기화까지
// 하는 유일한 창구다 — COPY 버튼(vqCopySolution)이 마지막 클리어 화면
// 데이터를 읽고, 캔버스 밖 UI(#level-title/#hint/#status/#solve-cmds)가
// 항상 캔버스와 같은 스냅샷을 보게 한다.
function vqDraw(st) {
  vqLastState = st;
  vqRenderer.draw(st);
  vqUpdatePanel(st);
}

// ── 사이드패널 동기화 ─────────────────────────────────────────────────
// 예전엔 Go(dom.go + platform.SetText/SetHTML)가 담당했지만, 커리큘럼 레벨의
// 표시 데이터(제목·힌트·명령 팔레트)를 wasm 에 싣지 않기 위해 JS 로 옮겼다.
// 커리큘럼은 LEVEL_META(생성 파일 levels_meta.js, 원본 levels_meta.go)를
// id 로 조회하고, :drill 등 런타임 생성 레벨은 스냅샷의 title/hint 를 쓴다.

// vqPanelCache 는 같은 값이면 DOM 을 다시 건드리지 않기 위한 dedupe 캐시 —
// #hint 의 타자기 효과(MutationObserver)가 값이 같은데도 재설정될 때마다
// 처음부터 다시 시작하는 것을 막는다(예전 platform.SetText 의 lastValues 와
// 같은 역할).
const vqPanelCache = {};

function vqSetText(id, value) {
  if (vqPanelCache[id] === value) return;
  vqPanelCache[id] = value;
  const el = document.getElementById(id);
  if (el) el.textContent = value;
}

function vqSetCmds(cmds) {
  const key = JSON.stringify(cmds);
  if (vqPanelCache['solve-cmds'] === key) return;
  vqPanelCache['solve-cmds'] = key;
  const el = document.getElementById('solve-cmds');
  if (!el) return;
  el.textContent = '';
  for (const c of cmds) {
    const row = document.createElement('div');
    row.className = 'cmd';
    const k = document.createElement('span');
    k.className = 'k';
    k.textContent = c.k;
    const d = document.createElement('span');
    d.className = 'd';
    d.textContent = c.d;
    row.append(k, d);
    el.append(row);
  }
}

function vqUpdatePanel(st) {
  switch (st.state) {
    case 'allclear':
      vqSetText('level-title', '🎉 ALL CLEAR!');
      vqSetText('hint', `Congratulations! You cleared all ${st.levelCount} levels across W1-W${st.worldCount}. Now go practice in real Vim!`);
      vqSetText('status', '');
      vqSetCmds([]);
      return;
    case 'clear':
      vqSetText('level-title', `LEVEL ${st.id} CLEAR!`);
      vqSetText('hint', 'Press Enter for the next level, or r to retry this one.');
      vqSetText('status', `keys ${st.clearStrokes} / par ${st.clearPar}`);
      vqSetCmds([]);
      return;
    case 'select':
      vqSetText('level-title', 'SELECT LEVEL');
      vqSetText('hint', 'h/l move between worlds, j/k move within a world, Enter to play, Esc to go back.');
      vqSetText('status', '');
      vqSetCmds([]);
      return;
  }

  // playing / drill
  const meta = (typeof LEVEL_META !== 'undefined' && LEVEL_META[st.id]) || null;
  vqSetText('level-title', meta ? meta.title : (st.title || ''));
  vqSetText('hint', meta ? meta.hint : (st.hint || ''));
  vqSetCmds(meta ? meta.cmds : []);

  let parInfo = `   ·   keys ${st.strokes}/par ${st.par}`;
  if (st.drill) {
    parInfo += `   ·   streak ${st.drill.streak}   ·   total ${st.drill.totalKeys}/${st.drill.totalPar}`;
  }
  if (st.kind === 'navigate') {
    let s = `keys ${st.keys}/${st.keysNeed}`;
    if (st.bugs > 0) {
      s += `   ·   ${st.bugs} bug(s) left`;
    } else if (st.keys === st.keysNeed) {
      s += '   ·   now head to $ (exit)!';
    }
    vqSetText('status', s + parInfo);
  } else {
    vqSetText('status', 'Make CURRENT match TARGET — a line turns green when it matches!' + parInfo);
  }
}

// effects/bell 이 살아있는 동안만 rAF 로 vqTick 을 돈다 — 상시 60fps 루프 없음.
function vqStartTick() {
  if (vqTickRunning) return;
  vqTickRunning = true;
  function loop() {
    const st = vqTick();
    vqDraw(st);
    if (st.effectsAlive) {
      requestAnimationFrame(loop);
    } else {
      vqTickRunning = false;
    }
  }
  requestAnimationFrame(loop);
}

// vqCallAndDraw 는 RESET/RESTART/LEVELS 같은 버튼 훅을 감싼다. web_js.go 의
// vimquestReset/Restart/LevelSelect 는 vqInput 과 동일하게 스냅샷을 돌려주므로,
// 여기서 바로 그려야 클릭 직후 화면이 갱신된다(버튼이 keydown 이벤트를 안 거쳐
// 그냥 두면 다음 키 입력 전까지 캔버스가 이전 화면에 멈춰 있는다).
function vqCallAndDraw(fn) {
  if (!fn) return;
  const st = fn();
  if (st && vqRenderer) {
    vqDraw(st);
    if (st.effectsAlive) vqStartTick();
  }
}

// vqCopySolution 은 클리어 화면의 "yours" 를 VimGolf 식 한 줄로 클립보드에
// 복사한다("VimQuest 3-3 · 5 keys (par 3): wdwj."). 클리어 화면이 아니거나
// yours 가 비어 있으면 조용히 무시(터미널 어법 — 에러 팝업 없음).
function vqCopySolution() {
  const st = vqLastState;
  if (!st || st.state !== 'clear' || !st.clearYours) return;
  const text = `VimQuest ${st.id} · ${st.clearStrokes} keys (par ${st.clearPar}): ${st.clearYours}`;
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).catch(() => {});
  }
}

function vqHandleKey(e) {
  if (e.isComposing || e.keyCode === 229) return;
  let tok = null;
  if (e.key === 'Enter') tok = '<cr>';
  else if (e.key === 'Escape') tok = '<esc>';
  else if (e.key === 'Backspace') tok = '<bs>';
  else if (e.ctrlKey && e.key.toLowerCase() === 'r') tok = '<c-r>';
  else if (e.ctrlKey || e.altKey || e.metaKey) return;
  else if (e.key.length === 1) tok = e.key;
  else return;

  if (!vqRenderer) {
    console.error('vqRenderer not initialized');
    return;
  }
  if (typeof vqInput !== 'function') {
    console.error('vqInput not available:', typeof vqInput);
    return;
  }

  e.preventDefault();
  const st = vqInput(tok);
  if (!st) {
    console.error('vqInput returned null for token:', tok);
    return;
  }
  vqDraw(st); // vqDraw 경유 — vqLastState(COPY 버튼)와 사이드패널이 함께 갱신된다
  if (st.effectsAlive) vqStartTick();
}

// vqInit 은 wasm 로드가 끝난 뒤(go.run 이후) 호출한다 — vqInput/vqState/vqTick
// 은 Go 쪽 main()(web_js.go)이 실행돼야 전역에 등록된다.
function vqInit() {
  const canvas = document.getElementById('game');
  vqRenderer = new Renderer(canvas);
  document.addEventListener('keydown', vqHandleKey);
  canvas.focus();
  const st = vqState();
  vqDraw(st);
  if (st.effectsAlive) vqStartTick();
}
