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
// 제출된 바디는 반환된 배열에 쌓인다.
async function stubArenaApi(page) {
  const submitted = [];
  await page.route('**/api/arena/**', async (route) => {
    const req = route.request();
    if (req.method() === 'POST') {
      const body = JSON.parse(req.postData() || '{}');
      submitted.push(body);
      await route.fulfill({ json: { ok: true, best_ms: body.ms, rank: 1 } });
      return;
    }
    await route.fulfill({
      json: { scores: [{ rank: 1, id: 'stub-champ', ms: 41000 }] },
    });
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
  await page.locator('#arena-panel .btn.g').click(); // ▶ START

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

  // 완주 화면에서 키는 삼켜진다(편집 버퍼로 새지 않음).
  await page.keyboard.type('x');
  expect((await page.evaluate(() => window.vqState())).state).toBe('arenaDone');

  // ID 입력·제출 — input 타이핑은 게임으로 새지 않아야 한다.
  await page.locator('#arena-id').fill('e2e-player');
  await page.locator('#arena-done .btn.y').click(); // ⬆ SUBMIT

  await expect(page.locator('#arena-msg')).toContainText('saved', { timeout: 5000 });
  expect(submitted).toHaveLength(1);
  expect(submitted[0].id).toBe('e2e-player');
  expect(submitted[0].ms).toBeGreaterThan(0);

  // 리더보드 테이블이 스텁 응답으로 채워진다.
  await expect(page.locator('#arena-lb td', { hasText: 'stub-champ' })).toBeVisible();

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
  await page.locator('#arena-panel .btn.g').click(); // START

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
  await page.locator('#arena-panel .btn.g').click(); // 재 START
  await page.keyboard.type(':q');
  await page.keyboard.press('Enter');
  expect((await page.evaluate(() => window.vqState())).state).toBe('select');
  await expect(page.locator('#arena-problem')).toHaveText('READY');
  await expect(page.locator('#arena-timer')).toHaveText('0.0s');
});
