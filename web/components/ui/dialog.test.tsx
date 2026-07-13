import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Dialog } from "@/components/ui/dialog";

// SPEC-211 D1. jsdom polyfills <dialog>'s showModal()/close() (vitest.setup.ts) so this
// component's OWN control logic is testable — driving the native API correctly, wiring
// backdrop-click and the close button to onClose, labelling via aria-labelledby. Real native
// semantics (Escape-to-close, focus trap, real backdrop hit-testing) are the browser's own
// guarantee for showModal()-shown dialogs, verified live in a real browser this session — not
// re-provable in jsdom, which doesn't implement them at all.
describe("Dialog (SPEC-211 D1)", () => {
  it("calls showModal when open, close when not", () => {
    const { rerender } = render(
      <Dialog open onClose={() => {}} title="Título">
        conteúdo
      </Dialog>,
    );
    const dialog = document.querySelector("dialog")!;
    expect(dialog.open).toBe(true);

    rerender(
      <Dialog open={false} onClose={() => {}} title="Título">
        conteúdo
      </Dialog>,
    );
    expect(dialog.open).toBe(false);
  });

  it("renders the title, linked via aria-labelledby, and the children", () => {
    render(
      <Dialog open onClose={() => {}} title="Adicionar FII">
        <p>conteúdo do formulário</p>
      </Dialog>,
    );
    const dialog = screen.getByRole("dialog", { name: "Adicionar FII" });
    expect(dialog).toBeInTheDocument();
    expect(screen.getByText("conteúdo do formulário")).toBeInTheDocument();
  });

  it("calls onClose when the close (X) button is clicked", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <Dialog open onClose={onClose} title="Título">
        conteúdo
      </Dialog>,
    );
    await user.click(screen.getByRole("button", { name: "Fechar" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("calls onClose on a backdrop click (click target is the dialog itself, not its content)", () => {
    const onClose = vi.fn();
    render(
      <Dialog open onClose={onClose} title="Título">
        conteúdo
      </Dialog>,
    );
    const dialog = document.querySelector("dialog")!;
    fireEvent.click(dialog); // target === the dialog element itself, per the backdrop-click contract
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("does NOT call onClose when clicking inside the visible content", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <Dialog open onClose={onClose} title="Título">
        <p>conteúdo</p>
      </Dialog>,
    );
    await user.click(screen.getByText("conteúdo"));
    expect(onClose).not.toHaveBeenCalled();
  });
});
