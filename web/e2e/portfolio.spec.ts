import { test, expect } from "@playwright/test";

// Carteira smoke (SPEC-211) — doubles as the integration proof against a real backend (this repo
// has no separate integration-test tier for the frontend; the same real-network Playwright run
// mirrors profile.spec.ts's precedent): register → add an FII holding and a fixed-income holding,
// see both appear → edit one, list reflects it → delete the other, confirm it's gone. Requires
// the backend running (Go API :8080 + Postgres).
test("create → edit → delete a holding, for both FII and fixed-income", async ({ page }) => {
  const email = `e2e-portfolio-${Date.now()}@example.com`;

  await page.goto("/register");
  await page.getByLabel("E-mail").fill(email);
  await page.getByLabel(/Senha/).fill("password123");
  await page.getByRole("button", { name: /Criar conta/ }).click();
  await expect(page).toHaveURL(/\/dashboard$/);

  await page.goto("/portfolio");
  await expect(page.getByText("Nenhum FII cadastrado")).toBeVisible();
  await expect(page.getByText("Nenhuma renda fixa cadastrada")).toBeVisible();

  // Add an FII holding.
  await page.getByRole("button", { name: "Adicionar FII" }).click();
  const fiiDialog = page.getByRole("dialog", { name: "Adicionar FII" });
  await fiiDialog.getByLabel("Ticker").fill("hglg11");
  await fiiDialog.getByLabel("Quantidade (cotas)").fill("100");
  await fiiDialog.getByLabel("Preço médio (R$)").fill("157,50");
  await fiiDialog.getByRole("button", { name: "Salvar" }).click();
  await expect(page.getByText("HGLG11")).toBeVisible();

  // Add a fixed-income holding (prefixado — no live-reference dependency for a deterministic smoke test).
  await page.getByRole("button", { name: "Adicionar renda fixa" }).click();
  const fiDialog = page.getByRole("dialog", { name: "Adicionar renda fixa" });
  await fiDialog.getByLabel("Nome").fill("CDB Banco X");
  await fiDialog.getByLabel("Instituição").fill("Banco X");
  await fiDialog.getByLabel("Valor investido (R$)").fill("1.000,00");
  await fiDialog.getByLabel("Taxa anual (%)").fill("10");
  await fiDialog.getByRole("button", { name: "Salvar" }).click();
  await expect(page.getByText("CDB Banco X")).toBeVisible();

  // Edit the FII holding — scoped to its row, since the fixed-income row also has an "Editar"
  // button — the list reflects the change.
  const fiiRow = page.getByRole("row", { name: /HGLG11/ });
  await fiiRow.getByRole("button", { name: "Editar" }).click();
  const editFiiDialog = page.getByRole("dialog", { name: "Editar FII" });
  await expect(editFiiDialog.getByLabel("Ticker")).toHaveValue("HGLG11");
  await editFiiDialog.getByLabel("Quantidade (cotas)").fill("150");
  await editFiiDialog.getByRole("button", { name: "Salvar" }).click();
  await expect(
    page.getByRole("row", { name: /HGLG11/ }).getByRole("cell", { name: "150" }),
  ).toBeVisible();

  // Delete the FII holding — confirm, then it's gone (the fixed-income one remains).
  await page
    .getByRole("row", { name: /HGLG11/ })
    .getByRole("button", { name: "Excluir" })
    .click();
  const confirmDialog = page.getByRole("dialog", { name: "Excluir FII?" });
  await confirmDialog.getByRole("button", { name: "Excluir" }).click();
  await expect(page.getByText("Nenhum FII cadastrado")).toBeVisible();
  await expect(page.getByText("CDB Banco X")).toBeVisible();
});
