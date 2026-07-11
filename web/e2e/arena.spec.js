// @ts-check
// Arena E2E — 탭 전환 → START → 5문제 완주 → 제출 → 리더보드까지, 브라우저
// keydown → glue.js → wasm → DOM 경로 전체를 확인한다. 네트워크는
// page.route 로 스텁해 CI 에 실제 리더보드 서버 없이 돈다(서버 자체의 검증은
// Go 쪽 test/arena 몫).
const { test, expect } = require('@playwright/test');

async function dismissIntro(page) {
  await page.locator('.btn.start').click();
}

async function waitForGame(page) {
  await page.waitForFunction(() => typeof window.vqState === 'function', null, {
    timeout: 15000,
  });
}

// ArenaLevels(internal/game/arena_levels.go)의 Solution 사본 — 'ESC' 는
// Escape 키. 레벨이 바뀌어 여기가 어긋나면 완주 단계에서 바로 실패하므로
// 드리프트는 테스트 실패로 드러난다.
const SOLUTIONS = [
  ['dwfox'],
  ['jddObeta', 'ESC'],
  ['ffcwbar', 'ESC', 'f9cw21', 'ESC', 'jA;', 'ESC'],
  ['d3wj.j.'],
  ['f"ci"release', 'ESC', 'jddf(ci(new', 'ESC'],
];

async function playSolution(page, tokens) {
  for (const t of tokens) {
    if (t === 'ESC') await page.keyboard.press('Escape');
    else await page.keyboard.type(t);
  }
}

// stubArenaApi 는 /api/arena/** 를 가로채 서버 없이 성공 응답을 돌려준다.
// 제출된 바디는 반환된 배열에 쌓인다. 응답 모양은 실제 서버와 같은 경쟁
// 컨텍스트(total/next_*/me)를 담는다 — 계약 자체의 검증은 Go 쪽 test/arena 몫.
async function stubArenaApi(page) {
  const submitted = [];
  await page.route('**/api/arena/**', async (route) => {
    const req = route.request();
    if (req.method() === 'POST') {
      const body = JSON.parse(req.postData() || '{}');
      submitted.push(body);
      await route.fulfill({
        json: {
          ok: true, best_ms: body.ms, rank: 2, total: 3,
          next_id: 'stub-champ', next_gap_ms: 1234,
        },
      });
      return;
    }
    // ?me= 가 붙으면 상위권 밖의 내 행을 함께 돌려준다(서버와 동일 계약).
    const json = { scores: [{ rank: 1, id: 'stub-champ', ms: 41000 }], total: 3 };
    if (req.url().includes('me=e2e-player')) {
      json.me = { rank: 2, id: 'e2e-player', ms: 42234 };
    }
    await route.fulfill({ json });
  });
  return submitted;
}

test('ARENA 탭 전환 — 패널이 바뀌고 게임 상태는 그대로', async ({ page }) => {
  await page.goto('/src/');
  await waitForGame(page);
  await dismissIntro(page);

  await expect(page.locator('#panel')).toBeVisible();
  await expect(page.locator('#arena-panel')).toBeHidden();

  await page.locator('#tab-arena').click();
  await expect(page.locator('#arena-panel')).toBeVisible();
  await expect(page.locator('#panel')).toBeHidden();
  // START 전엔 게임 상태를 건드리지 않는다.
  expect((await page.evaluate(() => window.vqState())).state).toBe('playing');

  await page.locator('#tab-tutorial').click();
  await expect(page.locator('#panel')).toBeVisible();
  await expect(page.locator('#arena-panel')).toBeHidden();
});

test('START → 5문제 완주 → 제출 → 리더보드 렌더', async ({ page }) => {
  const submitted = await stubArenaApi(page);
  await page.goto('/src/');
  await waitForGame(page);
  await dismissIntro(page);

  await page.locator('#tab-arena').click();
  await page.locator('#arena-start-btn').click(); // ▶ START

  let st = await page.evaluate(() => window.vqState());
  expect(st.state).toBe('arena');
  expect(st.arena.num).toBe(1);
  await expect(page.locator('#arena-problem')).toHaveText('PROBLEM 1 / 5');

  for (let i = 0; i < SOLUTIONS.length; i++) {
    await playSolution(page, SOLUTIONS[i]);
    st = await page.evaluate(() => window.vqState());
    if (i < SOLUTIONS.length - 1) {
      expect(st.state).toBe('arena');
      expect(st.arena.num).toBe(i + 2);
    }
  }
  expect(st.state).toBe('arenaDone');

  // 완주 UI — 시간이 얼려져 표시되고 제출 폼이 드러난다.
  await expect(page.locator('#arena-problem')).toHaveText('FINISHED!');
  await expect(page.locator('#arena-done')).toBeVisible();
  await expect(page.locator('#arena-final')).toContainText('YOUR TIME');

  // 스피드런 요소 — 문제 5개 전부의 누적 스플릿이 찍히고, 첫 완주는 곧
  // 개인 기록이라 NEW PB 배너와 상태줄 PB 가 함께 나타난다.
  await expect(page.locator('#arena-splits .split')).toHaveCount(5);
  await expect(page.locator('#arena-gap')).toContainText('PERSONAL BEST');
  await expect(page.locator('#arena-pb')).toContainText('PB ');
  // 이번 기록(수 초)이 스텁 보드 1위(41초)보다 빠르므로 추격 안내 대신
  // "보드 정상" 문구가 뜬다.
  await expect(page.locator('#arena-leader')).toContainText('tops the current board');

  // 완주 화면에서 키는 삼켜진다(편집 버퍼로 새지 않음).
  await page.keyboard.type('x');
  expect((await page.evaluate(() => window.vqState())).state).toBe('arenaDone');

  // ID 입력·제출 — input 타이핑은 게임으로 새지 않아야 한다.
  await page.locator('#arena-id').fill('e2e-player');
  await page.locator('#arena-submit-btn').click(); // ⬆ SUBMIT

  await expect(page.locator('#arena-msg')).toContainText('saved', { timeout: 5000 });
  expect(submitted).toHaveLength(1);
  expect(submitted[0].id).toBe('e2e-player');
  expect(submitted[0].ms).toBeGreaterThan(0);

  // 제출 응답의 경쟁 컨텍스트가 한 줄로 표시된다 — 분모 있는 순위와 추격 대상.
  await expect(page.locator('#arena-msg')).toContainText('rank #2/3');
  await expect(page.locator('#arena-msg')).toContainText('next: stub-champ');

  // 리더보드 — 상위권 + '⋯' 아래 내 행이 YOU 로 하이라이트되고 참가자 수가 붙는다.
  await expect(page.locator('#arena-lb td', { hasText: 'stub-champ' })).toBeVisible();
  await expect(page.locator('#arena-lb tr.me td', { hasText: 'e2e-player ◀ YOU' })).toBeVisible();
  await expect(page.locator('#arena-lb tr.total td')).toContainText('3 players');

  // ID 는 localStorage 에 저장돼 다음 방문 시 프리필된다.
  const savedId = await page.evaluate(() => localStorage.getItem('vimquest.arena.id'));
  expect(savedId).toBe('e2e-player');
});

test('아레나 도중 RESET 은 같은 문제 유지, 튜토리얼 복귀는 레벨 선택으로', async ({ page }) => {
  await stubArenaApi(page);
  await page.goto('/src/');
  await waitForGame(page);
  await dismissIntro(page);

  await page.locator('#tab-arena').click();
  await page.locator('#arena-start-btn').click(); // START

  await page.keyboard.type('dw'); // 문제 1 을 반쯤 풀다가
  await page.locator('#arena-panel .btn', { hasText: 'RESET' }).click();
  const st = await page.evaluate(() => window.vqState());
  expect(st.state).toBe('arena');
  expect(st.arena.num).toBe(1);
  expect(st.strokes).toBe(0);

  await page.locator('#tab-tutorial').click(); // 아레나 상태로 튜토리얼 복귀
  expect((await page.evaluate(() => window.vqState())).state).toBe('select');

  // 아레나 도중 :q 로 이탈하면 계측이 폐기되고 패널이 READY 로 돌아간다 —
  // 살아있는 타이머가 다음 완주 기록을 오염시키는 회귀를 막는다.
  await page.locator('#tab-arena').click();
  await page.locator('#arena-start-btn').click(); // 재 START
  await page.keyboard.type(':q');
  await page.keyboard.press('Enter');
  expect((await page.evaluate(() => window.vqState())).state).toBe('select');
  await expect(page.locator('#arena-problem')).toHaveText('READY');
  await expect(page.locator('#arena-timer')).toHaveText('0.0s');
});
