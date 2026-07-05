// @ts-check
// F4: 브라우저→wasm 브리지 회귀를 잡는 유일한 계층(E2E). Go 쪽 테스트는 전부
// 프로세스 내부라 여기(keydown → glue.js → vqInput → wasm)엔 구조적으로
// 닿을 수 없다 — 실제로 hjkl 입력 회귀가 두 번 났던 계층이 정확히 여기다.
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: '.',
  timeout: 30000,
  fullyParallel: false,
  retries: process.env.CI ? 1 : 0,
  webServer: {
    command: 'python3 -m http.server 8765',
    cwd: '..', // web/e2e -> web (index.html 은 web/src/, wasm 은 web/dist/)
    url: 'http://localhost:8765/src/',
    reuseExistingServer: !process.env.CI,
    timeout: 15000,
  },
  use: {
    baseURL: 'http://localhost:8765',
  },
});
