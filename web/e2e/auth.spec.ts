import { test, expect } from "@playwright/test";

// Smoke (SPEC-200 FR-2003/FR-2006): unauthenticated redirect → register (auto-login) →
// authenticated shell → logout. Requires the backend running (Go API :8080 + Postgres).
// Uses a unique email per run so it's idempotent.

test("unauthenticated visitor is redirected to /login", async ({ page }) => {
  await page.goto("/dashboard");
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "Entrar" })).toBeVisible();
});

test("register → land in the authenticated shell → logout", async ({ page }) => {
  const email = `e2e${Date.now()}@example.com`;

  await page.goto("/register");
  await page.getByLabel("E-mail").fill(email);
  await page.getByLabel(/Senha/).fill("password123");
  await page.getByRole("button", { name: /Criar conta/ }).click();

  await expect(page).toHaveURL(/\/dashboard$/);
  // the global floating copilot launcher is present on every authenticated screen
  await expect(page.getByRole("button", { name: "Abrir copiloto" })).toBeVisible();

  await page.getByRole("button", { name: "Sair" }).click();
  await expect(page).toHaveURL(/\/login$/);
});
