import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { UseMutationResult } from "@tanstack/react-query";
import { FiiForm } from "@/app/(app)/portfolio/fii-form";
import * as holdingsHooks from "@/lib/portfolio/holdings";
import { ApiError } from "@/lib/api/error";
import type { FIIHolding, FIIHoldingInput } from "@/lib/portfolio/holdings";

type UpdateVars = { id: string; input: FIIHoldingInput };

vi.mock("@/lib/portfolio/holdings", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/portfolio/holdings")>();
  return { ...actual, useCreateFIIHolding: vi.fn(), useUpdateFIIHolding: vi.fn() };
});

const useCreateFIIHolding = vi.mocked(holdingsHooks.useCreateFIIHolding);
const useUpdateFIIHolding = vi.mocked(holdingsHooks.useUpdateFIIHolding);

// A hand-written mutation-hook fake, generic over the mutation's data/variables shape — create,
// update, and delete each mutate with different variable/return types, so this can't be pinned
// to one of them (mirrors the project's no-mocking-library convention: hand fakes, not magic).
type MutateOpts<TData> = { onSuccess?: (data: TData) => void; onError?: (error: Error) => void };
function mutationHook<TData, TVariables>(
  over: Partial<UseMutationResult<TData, Error, TVariables>> & {
    onCall?: (vars: TVariables, opts?: MutateOpts<TData>) => void;
  } = {},
): UseMutationResult<TData, Error, TVariables> {
  const { onCall, ...rest } = over;
  return {
    mutate: vi.fn((vars: TVariables, opts?: MutateOpts<TData>) => onCall?.(vars, opts)),
    isPending: false,
    isError: false,
    error: null,
    ...rest,
  } as unknown as UseMutationResult<TData, Error, TVariables>;
}

const holding: FIIHolding = {
  id: "h1",
  ticker: "HGLG11",
  quantity: 100,
  average_price_centavos: 15750,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  useCreateFIIHolding.mockReturnValue(mutationHook<FIIHolding, FIIHoldingInput>());
  useUpdateFIIHolding.mockReturnValue(mutationHook<FIIHolding, UpdateVars>());
});

describe("FiiForm — add mode (SPEC-211 FR-2112)", () => {
  it("starts blank with Salvar disabled", () => {
    render(<FiiForm open onClose={() => {}} initial={null} />);
    expect(screen.getByRole("heading", { name: "Adicionar FII" })).toBeInTheDocument();
    expect(screen.getByLabelText("Ticker")).toHaveValue("");
    expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  });

  it("uppercases the ticker as typed (D3, free text)", async () => {
    const user = userEvent.setup();
    render(<FiiForm open onClose={() => {}} initial={null} />);
    await user.type(screen.getByLabelText("Ticker"), "hglg11");
    expect(screen.getByLabelText("Ticker")).toHaveValue("HGLG11");
  });

  it("gates Salvar on quantity ≥1 and a parseable price", async () => {
    const user = userEvent.setup();
    render(<FiiForm open onClose={() => {}} initial={null} />);
    const salvar = screen.getByRole("button", { name: "Salvar" });
    await user.type(screen.getByLabelText("Ticker"), "hglg11");
    expect(salvar).toBeDisabled(); // no quantity/price yet
    await user.type(screen.getByLabelText("Quantidade (cotas)"), "100");
    expect(salvar).toBeDisabled(); // no price yet
    await user.type(screen.getByLabelText("Preço médio (R$)"), "157,50");
    expect(salvar).toBeEnabled();
  });

  it("submits the exact FIIHoldingRequest (no user_id) and closes on success", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const create = mutationHook<FIIHolding, FIIHoldingInput>({
      onCall: (_v, opts) => opts?.onSuccess?.(holding),
    });
    useCreateFIIHolding.mockReturnValue(create);
    render(<FiiForm open onClose={onClose} initial={null} />);
    await user.type(screen.getByLabelText("Ticker"), "hglg11");
    await user.type(screen.getByLabelText("Quantidade (cotas)"), "100");
    await user.type(screen.getByLabelText("Preço médio (R$)"), "157,50");
    await user.click(screen.getByRole("button", { name: "Salvar" }));
    expect(create.mutate).toHaveBeenCalledWith(
      { ticker: "HGLG11", quantity: 100, average_price_centavos: 15750 },
      expect.anything(),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});

describe("FiiForm — edit mode (SPEC-211 FR-2113)", () => {
  it("prefills from the holding", () => {
    render(<FiiForm open onClose={() => {}} initial={holding} />);
    expect(screen.getByRole("heading", { name: "Editar FII" })).toBeInTheDocument();
    expect(screen.getByLabelText("Ticker")).toHaveValue("HGLG11");
    expect(screen.getByLabelText("Quantidade (cotas)")).toHaveValue(100);
    expect(screen.getByLabelText("Preço médio (R$)")).toHaveValue("157,50");
  });

  it("submits {id, input} to update and closes on success", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const update = mutationHook<FIIHolding, UpdateVars>({
      onCall: (_v, opts) => opts?.onSuccess?.(holding),
    });
    useUpdateFIIHolding.mockReturnValue(update);
    render(<FiiForm open onClose={onClose} initial={holding} />);
    await user.clear(screen.getByLabelText("Quantidade (cotas)"));
    await user.type(screen.getByLabelText("Quantidade (cotas)"), "150");
    await user.click(screen.getByRole("button", { name: "Salvar" }));
    expect(update.mutate).toHaveBeenCalledWith(
      { id: "h1", input: { ticker: "HGLG11", quantity: 150, average_price_centavos: 15750 } },
      expect.anything(),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("a 404 on update closes quietly — no scary error (BR-2111)", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const update = mutationHook<FIIHolding, UpdateVars>({
      onCall: (_v, opts) => opts?.onError?.(new ApiError(404, "not found")),
    });
    useUpdateFIIHolding.mockReturnValue(update);
    render(<FiiForm open onClose={onClose} initial={holding} />);
    await user.click(screen.getByRole("button", { name: "Salvar" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("a genuine 400 on update surfaces inline and does NOT close", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const update = mutationHook<FIIHolding, UpdateVars>({
      isError: true,
      error: new ApiError(400, "ticker inválido"),
      onCall: (_v, opts) => opts?.onError?.(new ApiError(400, "ticker inválido")),
    });
    useUpdateFIIHolding.mockReturnValue(update);
    render(<FiiForm open onClose={onClose} initial={holding} />);
    await user.click(screen.getByRole("button", { name: "Salvar" }));
    expect(screen.getByText("ticker inválido")).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });
});
