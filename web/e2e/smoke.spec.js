// @ts-check
// F4: 브라우저 스모크 — 넓고 얕게. 게임 규칙 검증은 Go 테스트(internal/game)의
// 몫이고, 여기서는 "브라우저 keydown이 실제로 wasm 까지 도달해 상태를
// 바꾸는가"만 vqState() 스냅샷으로 확인한다. 픽셀 비교는 하지 않는다
// (docs/ARCHITECTURE.md D2 논의 — 골든 이미지는 비용 대비 과함).
const { test, expect } = require('@playwright/test');

// dismissIntro 는 최초 진입 시 뜨는 안내 오버레이(#intro)를 닫는다 —
// 안 닫으면 캔버스 클릭/키 입력이 오버레이에 가로채인다.
async function dismissIntro(page) {
  await page.locator('.btn.start').click();
}

test('hjkl 이동, :q 로 레벨 선택 진입', async ({ page }) => {
  await page.goto('/src/');

  // wasm 로드가 끝나 vqInit()이 vqState 를 전역에 등록할 때까지 대기.
  await page.waitForFunction(() => typeof window.vqState === 'function', null, {
    timeout: 15000,
  });
  await dismissIntro(page);

  const before = await page.evaluate(() => window.vqState());
  expect(before.state).toBe('playing');
  const startRow = before.row;
  const startCol = before.col;

  await page.locator('#game').click();
  await page.keyboard.press('l');
  await page.keyboard.press('l');
  await page.keyboard.press('j');

  const afterMove = await page.evaluate(() => window.vqState());
  expect(afterMove.col).toBeGreaterThan(startCol);
  expect(afterMove.row).toBeGreaterThan(startRow);

  // :q<cr> — 레벨 선택 화면으로.
  await page.keyboard.type(':q');
  await page.keyboard.press('Enter');

  const afterQuit = await page.evaluate(() => window.vqState());
  expect(afterQuit.state).toBe('select');
});

test('사이드패널이 LEVEL_META(생성 파일)로 채워진다', async ({ page }) => {
  await page.goto('/src/');
  await page.waitForFunction(() => typeof window.vqState === 'function', null, {
    timeout: 15000,
  });
  await dismissIntro(page);

  // 커리큘럼 표시 데이터가 wasm 이 아니라 levels_meta.js 에서 온다 —
  // 제목/힌트/명령 팔레트가 실제로 패널에 채워졌는지 확인한다.
  await expect(page.locator('#level-title')).toHaveText(/1-1/);
  await expect(page.locator('#hint')).toContainText('hjkl', { timeout: 5000 });
  await expect(page.locator('#solve-cmds .cmd').first()).toBeVisible();

  // 레벨 선택으로 나가면 패널도 상태에 맞게 바뀐다.
  await page.locator('#game').click();
  await page.keyboard.type(':q');
  await page.keyboard.press('Enter');
  await expect(page.locator('#level-title')).toHaveText('SELECT LEVEL');
});

test('치트시트 열림 중 키는 게임으로 새지 않는다', async ({ page }) => {
  await page.goto('/src/');
  await page.waitForFunction(() => typeof window.vqState === 'function', null, {
    timeout: 15000,
  });
  await dismissIntro(page);

  const before = await page.evaluate(() => window.vqState());
  expect(before.state).toBe('playing');

  await page.locator('button', { hasText: 'CHEATSHEET' }).click();
  await expect(page.locator('#cheatsheet')).toBeVisible();

  // 이동 키·모드 전환 키 모두 오버레이가 삼킨다. 특히 닫는 Esc 는
  // document(vqHandleKey)가 window(닫기 핸들러)보다 먼저 받으므로,
  // 가드가 없으면 치트시트를 닫는 순간 게임 모드까지 바뀌는 회귀가 있었다.
  await page.keyboard.press('l');
  await page.keyboard.press('i');
  await page.keyboard.press('Escape');
  await expect(page.locator('#cheatsheet')).toBeHidden();

  const after = await page.evaluate(() => window.vqState());
  expect(after.col).toBe(before.col);
  expect(after.mode).toBe(before.mode);
  expect(after.strokes).toBe(before.strokes);

  // 닫힌 뒤에는 키가 다시 게임에 도달한다.
  await page.keyboard.press('l');
  expect((await page.evaluate(() => window.vqState())).col).toBe(before.col + 1);
});

test('Esc 로 레벨 선택에서 복귀', async ({ page }) => {
  await page.goto('/src/');
  await page.waitForFunction(() => typeof window.vqState === 'function', null, {
    timeout: 15000,
  });
  await dismissIntro(page);

  await page.locator('#game').click();
  await page.keyboard.type(':q');
  await page.keyboard.press('Enter');
  expect((await page.evaluate(() => window.vqState())).state).toBe('select');

  await page.keyboard.press('Escape');
  const back = await page.evaluate(() => window.vqState());
  expect(back.state).toBe('playing');
});
