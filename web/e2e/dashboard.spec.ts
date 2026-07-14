import { test, expect } from "@playwright/test";

// Painel smoke (SPEC-212) — doubles as the integration proof against a real backend (this repo
// has no separate integration-test tier for the frontend; the same real-network Playwright run
// mirrors e2e/portfolio.spec.ts's precedent): register → the fresh dashboard shows the empty
// state → add a holding via Carteira → the dashboard reflects a non-zero patrimony, with the
// hero and the metric row's growth card agreeing exactly. Requires the backend running (Go API
// :8080 + Postgres).
test("empty dashboard, then reflects a holding added via Carteira", async ({ page }) => {
  const email = `e2e-dashboard-${Date.now()}@example.com`;

  await page.goto("/register");
  await page.getByLabel("E-mail").fill(email);
  await page.getByLabel(/Senha/).fill("password123");
  await page.getByRole("button", { name: /Criar conta/ }).click();
  await expect(page).toHaveURL(/\/dashboard$/);

  // Fresh account: the empty state, not a zeroed dashboard.
  await expect(page.getByText("Sua carteira está vazia")).toBeVisible();

  // Its CTA navigates to Carteira.
  await page.getByRole("button", { name: "Ir para a Carteira" }).click();
  await expect(page).toHaveURL(/\/portfolio$/);

  // Add a fixed-income holding (prefixado — deterministic, no market-data dependency).
  await page.getByRole("button", { name: "Adicionar renda fixa" }).click();
  const fiDialog = page.getByRole("dialog", { name: "Adicionar renda fixa" });
  await fiDialog.getByLabel("Nome").fill("CDB Banco X");
  await fiDialog.getByLabel("Instituição").fill("Banco X");
  await fiDialog.getByLabel("Valor investido (R$)").fill("5.000,00");
  await fiDialog.getByLabel("Taxa anual (%)").fill("10");
  await fiDialog.getByRole("button", { name: "Salvar" }).click();
  await expect(page.getByText("CDB Banco X")).toBeVisible();

  // Back on the Painel: a non-zero patrimony, matching what was just invested (no market-value
  // drift possible for a same-day prefixado holding — the figures must match exactly).
  await page.goto("/dashboard");
  await expect(page.getByText("Patrimônio total")).toBeVisible();
  // Both the hero AND the "Total investido" metric card legitimately show R$ 5.000,00 — a
  // same-day prefixado holding has zero elapsed days of accrual, so current value === cost basis.
  await expect(page.getByText("R$ 5.000,00")).toHaveCount(2);
  await expect(page.getByText("Alocação por classe de ativo")).toBeVisible();
  // A fixed-income-only portfolio has no FII sector exposure — the section is omitted, not empty.
  await expect(page.getByText("Exposição por setor (FIIs)")).not.toBeVisible();
});
