import { test, expect } from "@playwright/test";

// Profile smoke (SPEC-210): register → set the profile → save → reload persists.
// Requires the backend running (Go API :8080 + Postgres). Unique email per run.
test("set the investor profile → save → persists on reload", async ({ page }) => {
  const email = `e2e-profile-${Date.now()}@example.com`;

  await page.goto("/register");
  await page.getByLabel("E-mail").fill(email);
  await page.getByLabel(/Senha/).fill("password123");
  await page.getByRole("button", { name: /Criar conta/ }).click();
  await expect(page).toHaveURL(/\/dashboard$/);

  // First run: the Perfil screen shows the empty "Defina seu perfil" state.
  await page.goto("/profile");
  await expect(page.getByRole("heading", { name: "Defina seu perfil" })).toBeVisible();

  await page.getByRole("radio", { name: "Agressivo" }).click();
  await page.getByRole("checkbox", { name: "Aposentadoria" }).click();
  await page.getByRole("checkbox", { name: "Crescimento de longo prazo" }).click();
  await page.getByRole("button", { name: "Salvar" }).click();
  await expect(page.getByText("Perfil salvo")).toBeVisible();

  // Reload → prefilled ("Seu perfil"), the selections persisted.
  await page.reload();
  await expect(page.getByRole("heading", { name: "Seu perfil" })).toBeVisible();
  await expect(page.getByRole("radio", { name: "Agressivo" })).toBeChecked();
  await expect(page.getByRole("checkbox", { name: "Aposentadoria" })).toBeChecked();
});
