import { defineConfig, devices } from "@playwright/test";

// E2E smoke (SPEC-200 Phase 7). NOT wired into web-ci yet — E2E needs the backend as a CI
// service (deferred follow-up). To run locally:
//   1. `npx playwright install chromium`   (one-time browser download, ~150 MB)
//   2. bring the backend up: Go API on :8080 + Postgres (e.g. `task docker-up`)
//   3. `npm run test:e2e`
export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: false,
  reporter: "list",
  use: {
    baseURL: "http://localhost:3000",
    trace: "on-first-retry",
  },
  // Playwright starts (or reuses) the dev server, which proxies /api/* to the backend.
  webServer: {
    command: "npm run dev",
    url: "http://localhost:3000/login",
    reuseExistingServer: true,
    timeout: 60_000,
    env: { API_PROXY_TARGET: process.env.API_PROXY_TARGET ?? "http://localhost:8080" },
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
