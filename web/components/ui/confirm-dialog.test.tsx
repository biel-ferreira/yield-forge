import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

// SPEC-211 D2 — the one delete-confirm pattern reused for FII and fixed-income deletes.
describe("ConfirmDialog (SPEC-211 D2)", () => {
  it("requires an explicit confirm click — no accidental one-click destroy", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(
      <ConfirmDialog
        open
        title="Excluir FII?"
        description="Remover HGLG11 da sua carteira?"
        onConfirm={onConfirm}
        onCancel={() => {}}
      />,
    );
    expect(onConfirm).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Excluir" }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel from the Cancelar button", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(
      <ConfirmDialog
        open
        title="Excluir FII?"
        description="Remover HGLG11 da sua carteira?"
        onConfirm={() => {}}
        onCancel={onCancel}
      />,
    );
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("disables both actions while pending", () => {
    render(
      <ConfirmDialog
        open
        title="Excluir FII?"
        description="…"
        isPending
        onConfirm={() => {}}
        onCancel={() => {}}
      />,
    );
    expect(screen.getByRole("button", { name: "Cancelar" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Excluir" })).toBeDisabled();
  });

  it("surfaces a genuine failure inline (not the 404-as-success case)", () => {
    render(
      <ConfirmDialog
        open
        title="Excluir FII?"
        description="…"
        error="Falha ao excluir."
        onConfirm={() => {}}
        onCancel={() => {}}
      />,
    );
    expect(screen.getByText("Falha ao excluir.")).toBeInTheDocument();
  });
});
