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

// ── 언어 상태(EN/KO) ─────────────────────────────────────────────────
// 예전엔 매뉴얼(toggleLang)과 치트시트(toggleCheatLang)가 각자 독립된
// on/off 상태를 들고 있어서, 한쪽만 한국어로 바꾸고 다른 쪽은 영어로
// 남는 게 가능했다. 레벨 힌트 패널까지 한국어를 지원하게 되면서 셋을
// 하나의 전역 상태로 묶는다 — 토글 1회로 매뉴얼·치트시트·힌트가 함께 바뀐다.
let vqLang = 'en';

function vqApplyLang() {
  const showKo = vqLang === 'ko';

  const manualEn = document.getElementById('manual-en');
  const manualKo = document.getElementById('manual-ko');
  const langBtn = document.getElementById('lang-btn');
  const warn = document.getElementById('intro-warn');
  if (manualEn && manualKo) {
    manualEn.style.display = showKo ? 'none' : '';
    manualKo.style.display = showKo ? '' : 'none';
  }
  if (langBtn) langBtn.textContent = showKo ? 'English' : '한국어';
  if (warn) {
    warn.textContent = showKo
      ? '⚠ 키 입력은 영문 입력 상태에서만 동작합니다 (한/영 키 확인)'
      : '⚠ Keys work only in English input mode (check your 한/영 key)';
  }

  const cheatEn = document.getElementById('cheat-en');
  const cheatKo = document.getElementById('cheat-ko');
  const cheatBtn = document.getElementById('cheat-lang-btn');
  if (cheatEn && cheatKo) {
    cheatEn.style.display = showKo ? 'none' : '';
    cheatKo.style.display = showKo ? '' : 'none';
  }
  if (cheatBtn) cheatBtn.textContent = showKo ? 'English' : '한국어';

  // 레벨 힌트 패널도 즉시 반영 — vqUpdatePanel 이 vqLang 을 읽어 LEVEL_META
  // 또는 LEVEL_META_KO 를 고른다(마지막으로 그린 스냅샷 기준으로 재조회).
  if (vqLastState) vqUpdatePanel(vqLastState);
}

// vqToggleLang 은 매뉴얼의 한국어 버튼과 치트시트의 한국어 버튼 양쪽이
// 공유하는 단일 토글 진입점이다(index.html 의 onclick 이 호출).
function vqToggleLang() {
  vqLang = vqLang === 'en' ? 'ko' : 'en';
  vqApplyLang();
  vqRefreshOnboardingText();
}

// ── 신규 세션 온보딩 프롬프트 ─────────────────────────────────────────
// migrateProgress()(이 파일 맨 위)가 이미 쓰는 'vimquest.v1'/'vimquest.v2'
// 판별 로직을 그대로 재사용한다 — 새 API 나 game/snapshot.go 계약 변경
// 없이, 저장된 진행이 전혀 없으면 "신규 세션"으로 본다. 상태바 위에 한 줄
// 텍스트만 추가/제거한다(박스·화살표·애니메이션 없음 — 터미널 어법 유지).
let vqOnboardingKeysLeft = 0; // 0 이면 비활성(신규 세션이 아니거나 이미 소진)

function vqOnboardingText() {
  return vqLang === 'ko'
    ? '팁: j 를 눌러 아래로 이동해 보세요'
    : 'tip: press j to move down';
}

function vqShowOnboardingIfNewSession() {
  const seen = localStorage.getItem('vimquest.v1') || localStorage.getItem('vimquest.v2');
  if (seen) return;
  vqOnboardingKeysLeft = 3;
  const el = document.getElementById('onboarding-tip');
  if (el) {
    el.textContent = vqOnboardingText();
    el.style.display = '';
  }
}

// 언어 토글 중에도 문구가 즉시 반영되도록 vqToggleLang 이 호출한다.
function vqRefreshOnboardingText() {
  if (vqOnboardingKeysLeft <= 0) return;
  const el = document.getElementById('onboarding-tip');
  if (el) el.textContent = vqOnboardingText();
}

// vqNoteOnboardingKey 는 키 입력마다(게임이 그 키를 받아들였는지와 무관하게)
// 독립적으로 카운트한다 — 3번째 키에서 프롬프트를 지운다.
function vqNoteOnboardingKey() {
  if (vqOnboardingKeysLeft <= 0) return;
  vqOnboardingKeysLeft--;
  if (vqOnboardingKeysLeft <= 0) {
    const el = document.getElementById('onboarding-tip');
    if (el) el.style.display = 'none';
  }
}

// vqDraw 는 렌더러에 그리는 동시에 vqLastState 갱신·사이드패널 동기화까지
// 하는 유일한 창구다 — COPY 버튼(vqCopySolution)이 마지막 클리어 화면
// 데이터를 읽고, 캔버스 밖 UI(#level-title/#hint/#status/#solve-cmds)가
// 항상 캔버스와 같은 스냅샷을 보게 한다.
function vqDraw(st) {
  vqLastState = st;
  vqRenderer.draw(st);
  vqUpdatePanel(st);
  vqArenaOnState(st);
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
    case 'drillSummary':
      vqSetText('level-title', 'DRILL SESSION SUMMARY');
      vqSetText('hint', 'Press any key to head back to level select.');
      vqSetText('status', `streak ${st.drillStreak}   ·   keys ${st.drillTotalKeys}/${st.drillTotalPar}`);
      vqSetCmds([]);
      return;
    case 'arenaDone':
      return; // 완주 UI 는 #arena-panel(vqArenaOnState)이 소유 — 튜토리얼 패널은 손대지 않는다
  }

  // playing / drill
  const metaTable = vqLang === 'ko'
    ? (typeof LEVEL_META_KO !== 'undefined' ? LEVEL_META_KO : null)
    : (typeof LEVEL_META !== 'undefined' ? LEVEL_META : null);
  const meta = (metaTable && metaTable[st.id]) || null;
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
  // Arena 의 ID <input> 등 폼 요소에 포커스가 있으면 게임으로 보내지 않는다 —
  // preventDefault 가 타이핑 자체를 먹어버린다.
  const tgt = e.target;
  if (tgt && (tgt.tagName === 'INPUT' || tgt.tagName === 'TEXTAREA')) return;
  let tok = null;
  if (e.key === 'Enter') tok = '<cr>';
  else if (e.key === 'Escape') tok = '<esc>';
  else if (e.key === 'Backspace') tok = '<bs>';
  else if (e.ctrlKey && e.key.toLowerCase() === 'r') tok = '<c-r>';
  else if (e.ctrlKey || e.altKey || e.metaKey) return;
  else if (e.key.length === 1) tok = e.key;
  else return;

  // 게임(vqInput)이 아직 준비되기 전이라도 "실제로 입력된 키"로 센다 —
  // 온보딩 카운트는 게임 로직/엔진 준비 상태에 기대지 않는 순수 JS 카운트다.
  vqNoteOnboardingKey();

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

// ── Arena(시간공격) ──────────────────────────────────────────────────
// 시간 측정은 여기(JS performance.now())가 전부다 — Go/엔진엔 wall-clock
// 개념이 없고, 서버는 클라이언트가 신고한 시간을 그대로 신뢰한다(사용자가
// 명시적으로 선택한 단순화). 네트워킹도 wasm 을 거치지 않는 순수 fetch.

// 프론트가 python http.server(다른 origin)로 뜨는 개발 배치가 기본이라
// API 베이스는 절대 URL — 배포 시엔 window.VQ_ARENA_API 로 덮어쓴다.
const vqArenaApi = window.VQ_ARENA_API || 'http://localhost:8080';
const vqArenaIdKey = 'vimquest.arena.id'; // 진행률 키(vimquest.v2)와 분리

let vqArenaT0 = 0;         // START 시각(performance.now) — 0 이면 계측 중 아님
let vqArenaFinalMs = null; // arenaDone 프레임에 딱 한 번 얼린 최종 기록
let vqArenaTimerOn = false; // rAF 루프 중복 기동 방지(vqTickRunning 과 같은 관례)

function vqArenaFmt(ms) {
  return (ms / 1000).toFixed(1) + 's';
}

// 계측 중에만 rAF 로 타이머 표시를 갱신한다(vqStartTick 과 같은 원칙 —
// 상시 루프 없음, 한 번에 한 루프만).
function vqArenaTimerLoop() {
  if (!vqArenaT0) {
    vqArenaTimerOn = false;
    return;
  }
  const el = document.getElementById('arena-timer');
  if (el) el.textContent = vqArenaFmt(performance.now() - vqArenaT0);
  requestAnimationFrame(vqArenaTimerLoop);
}

function vqArenaTimerStart() {
  if (vqArenaTimerOn) return;
  vqArenaTimerOn = true;
  requestAnimationFrame(vqArenaTimerLoop);
}

function vqArenaStartClick() {
  if (typeof vqArenaStart !== 'function') return; // wasm 로드 전
  vqArenaT0 = 0; // 진행 중이던 계측 폐기 — 아래 관찰 훅이 새 런의 시계를 개시한다
  vqCallAndDraw(window.vqArenaStart);
  const c = document.getElementById('game');
  if (c) c.focus();
}

// vqArenaOnState 는 vqDraw 를 지나는 모든 스냅샷을 관찰한다 — 계측의 개시/
// 폐기/동결을 전부 게임 상태 전이에서만 결정하는 단일 지점이다. 버튼별로
// 시계를 만지면 START 밖의 진입 경로(완주 화면 RESET = Go 쪽 EnterArena)가
// 생길 때마다 시계가 새는 버그가 재발한다.
function vqArenaOnState(st) {
  if (st.state === 'arena' && st.arena) {
    if (!vqArenaT0) {
      // 새 런 개시 — 완주 잔해(동결 기록·제출 폼·메시지)를 치우고 계측 시작.
      vqArenaT0 = performance.now();
      vqArenaFinalMs = null;
      const done = document.getElementById('arena-done');
      if (done) done.style.display = 'none';
      const msg = document.getElementById('arena-msg');
      if (msg) msg.textContent = '';
      vqArenaTimerStart();
    }
    const el = document.getElementById('arena-problem');
    if (el) el.textContent = `PROBLEM ${st.arena.num} / ${st.arena.count}`;
    return;
  }
  // 아레나 계열 밖으로 나갔는데 계측이 살아있으면(예: 도중 :q 로 레벨 선택
  // 이탈) 폐기한다 — 안 그러면 rAF 타이머가 세션 내내 돌고, 다음 완주가
  // 이탈 이전 시각 기준으로 얼려져 엉뚱한 기록이 제출된다.
  if (st.state !== 'arenaDone' && vqArenaT0) {
    vqArenaT0 = 0;
    const p = document.getElementById('arena-problem');
    const tm = document.getElementById('arena-timer');
    if (p) p.textContent = 'READY';
    if (tm) tm.textContent = '0.0s';
    return;
  }
  if (st.state === 'arenaDone' && vqArenaT0) {
    vqArenaFinalMs = Math.round(performance.now() - vqArenaT0);
    vqArenaT0 = 0; // rAF 루프 자연 종료
    const set = (id, v) => { const el = document.getElementById(id); if (el) el.textContent = v; };
    set('arena-problem', 'FINISHED!');
    set('arena-timer', vqArenaFmt(vqArenaFinalMs));
    set('arena-final', `YOUR TIME: ${vqArenaFmt(vqArenaFinalMs)}`);
    const done = document.getElementById('arena-done');
    if (done) done.style.display = '';
    const idEl = document.getElementById('arena-id');
    const saved = localStorage.getItem(vqArenaIdKey);
    if (idEl && saved) idEl.value = saved;
    vqArenaFetchBoard();
  }
}

function vqArenaSubmit() {
  const idEl = document.getElementById('arena-id');
  const msg = document.getElementById('arena-msg');
  const id = idEl ? idEl.value.trim() : '';
  if (vqArenaFinalMs == null) return;
  if (!id) {
    if (msg) msg.textContent = 'enter an id first';
    return;
  }
  localStorage.setItem(vqArenaIdKey, id);
  if (msg) msg.textContent = 'submitting…';
  fetch(vqArenaApi + '/api/arena/score', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id, ms: vqArenaFinalMs }),
  })
    .then((r) => r.json())
    .then((res) => {
      if (res && res.ok) {
        if (msg) msg.textContent = `saved — best ${vqArenaFmt(res.best_ms)} · rank #${res.rank}`;
        return vqArenaFetchBoard();
      }
      if (msg) msg.textContent = 'rejected: ' + ((res && res.error) || 'unknown error');
    })
    .catch(() => {
      if (msg) msg.textContent = 'server unreachable — start it with: go run ./cmd/server';
    });
}

function vqArenaFetchBoard() {
  fetch(vqArenaApi + '/api/arena/leaderboard?limit=10')
    .then((r) => r.json())
    .then((res) => vqArenaRenderBoard((res && res.scores) || []))
    .catch(() => {
      const msg = document.getElementById('arena-msg');
      if (msg) msg.textContent = 'server unreachable — start it with: go run ./cmd/server';
    });
}

function vqArenaRenderBoard(scores) {
  const tbl = document.getElementById('arena-lb');
  if (!tbl) return;
  tbl.textContent = '';
  if (!scores.length) return;
  const head = tbl.insertRow();
  for (const h of ['RANK', 'ID', 'TIME']) {
    const th = document.createElement('th');
    th.textContent = h;
    head.append(th);
  }
  for (const s of scores) {
    const row = tbl.insertRow();
    row.insertCell().textContent = '#' + s.rank;
    row.insertCell().textContent = s.id;
    const ms = row.insertCell();
    ms.textContent = vqArenaFmt(s.ms);
    ms.className = 'ms';
  }
}

// vqSwitchTab 은 TUTORIAL/ARENA 패널 표시를 전환한다. 캔버스는 공유 —
// ARENA 탭은 START 전까지 게임 상태를 건드리지 않고, 아레나 도중 TUTORIAL
// 로 돌아가면 진행 중 계측을 폐기하고 레벨 선택으로 복귀시킨다(아레나
// 상태의 캔버스가 튜토리얼 패널 뒤에 남아 헷갈리지 않게).
function vqSwitchTab(tab) {
  const arena = tab === 'arena';
  document.body.classList.toggle('tab-arena', arena);
  const tt = document.getElementById('tab-tutorial');
  const ta = document.getElementById('tab-arena');
  if (tt) tt.classList.toggle('active', !arena);
  if (ta) ta.classList.toggle('active', arena);
  if (!arena && vqLastState && (vqLastState.state === 'arena' || vqLastState.state === 'arenaDone')) {
    vqArenaT0 = 0;
    vqCallAndDraw(window.vimquestLevelSelect);
  }
  const c = document.getElementById('game');
  if (c) c.focus();
}

// vqInit 은 wasm 로드가 끝난 뒤(go.run 이후) 호출한다 — vqInput/vqState/vqTick
// 은 Go 쪽 main()(web_js.go)이 실행돼야 전역에 등록된다.
function vqInit() {
  const canvas = document.getElementById('game');
  vqRenderer = new Renderer(canvas);
  document.addEventListener('keydown', vqHandleKey);
  canvas.focus();
  vqShowOnboardingIfNewSession();
  const st = vqState();
  vqDraw(st);
  if (st.effectsAlive) vqStartTick();
}
