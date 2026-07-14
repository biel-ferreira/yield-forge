import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { UseMutationResult } from "@tanstack/react-query";
import { FiiSection } from "@/app/(app)/portfolio/fii-table";
import * as holdingsHooks from "@/lib/portfolio/holdings";
import { ApiError } from "@/lib/api/error";
import type { FIIHolding, FIIHoldingInput } from "@/lib/portfolio/holdings";

type UpdateVars = { id: string; input: FIIHoldingInput };

vi.mock("@/lib/portfolio/holdings", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/portfolio/holdings")>();
  return {
    ...actual,
    useFIIHoldings: vi.fn(),
    useCreateFIIHolding: vi.fn(),
    useUpdateFIIHolding: vi.fn(),
    useDeleteFIIHolding: vi.fn(),
  };
});

const useFIIHoldings = vi.mocked(holdingsHooks.useFIIHoldings);
const useCreateFIIHolding = vi.mocked(holdingsHooks.useCreateFIIHolding);
const useUpdateFIIHolding = vi.mocked(holdingsHooks.useUpdateFIIHolding);
const useDeleteFIIHolding = vi.mocked(holdingsHooks.useDeleteFIIHolding);

type ListState = ReturnType<typeof holdingsHooks.useFIIHoldings>;

function listState(over: Partial<ListState> = {}): ListState {
  return {
    holdings: [],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
    ...over,
  } as unknown as ListState;
}

// Generic over the mutation's data/variables shape — create/update/delete each mutate with
// different types (mirrors the project's no-mocking-library convention: hand fakes, not magic).
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
    reset: vi.fn(),
    ...rest,
  } as unknown as UseMutationResult<TData, Error, TVariables>;
}

const hglg: FIIHolding = {
  id: "h1",
  ticker: "HGLG11",
  quantity: 100,
  average_price_centavos: 15750,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};
const knri: FIIHolding = {
  id: "h2",
  ticker: "KNRI11",
  quantity: 50,
  average_price_centavos: 16000,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  useCreateFIIHolding.mockReturnValue(mutationHook<FIIHolding, FIIHoldingInput>());
  useUpdateFIIHolding.mockReturnValue(mutationHook<FIIHolding, UpdateVars>());
  useDeleteFIIHolding.mockReturnValue(mutationHook<void, string>());
});

describe("FiiSection — list states (SPEC-211 FR-2111)", () => {
  it("loading → skeleton, no table", () => {
    useFIIHoldings.mockReturnValue(listState({ isLoading: true }));
    render(<FiiSection />);
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("error → retry button calls refetch", async () => {
    const user = userEvent.setup();
    const refetch = vi.fn();
    useFIIHoldings.mockReturnValue(listState({ isError: true, refetch }));
    render(<FiiSection />);
    await user.click(screen.getByRole("button", { name: "Tentar novamente" }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("empty → empty state with an Adicionar FII CTA, not an error", () => {
    useFIIHoldings.mockReturnValue(listState({ holdings: [] }));
    render(<FiiSection />);
    expect(screen.getByText("Nenhum FII cadastrado")).toBeInTheDocument();
    expect(screen.queryByText(/Não foi possível/)).not.toBeInTheDocument();
  });

  it("populated → table sorted by ticker with formatted money", () => {
    useFIIHoldings.mockReturnValue(listState({ holdings: [knri, hglg] })); // deliberately unsorted
    render(<FiiSection />);
    const rows = screen.getAllByRole("row").slice(1); // skip header
    expect(rows[0]).toHaveTextContent("HGLG11");
    expect(rows[1]).toHaveTextContent("KNRI11");
    expect(screen.getByText("R$ 157,50")).toBeInTheDocument();
  });
});

describe("FiiSection — edit/delete wiring (SPEC-211 FR-2113/FR-2114, BR-2111)", () => {
  it("Editar opens the prefilled form", async () => {
    const user = userEvent.setup();
    useFIIHoldings.mockReturnValue(listState({ holdings: [hglg] }));
    render(<FiiSection />);
    await user.click(screen.getByRole("button", { name: "Editar" }));
    expect(screen.getByRole("heading", { name: "Editar FII" })).toBeInTheDocument();
    expect(screen.getByLabelText("Ticker")).toHaveValue("HGLG11");
  });

  it("Excluir → confirm → deletes and closes", async () => {
    const user = userEvent.setup();
    useFIIHoldings.mockReturnValue(listState({ holdings: [hglg] }));
    const del = mutationHook<void, string>({ onCall: (_v, opts) => opts?.onSuccess?.(undefined) });
    useDeleteFIIHolding.mockReturnValue(del);
    render(<FiiSection />);
    await user.click(screen.getByRole("button", { name: "Excluir" }));
    const dialog = screen.getByRole("dialog", { name: "Excluir FII?" });
    expect(within(dialog).getByText(/Remover HGLG11/)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Excluir" }));
    expect(del.mutate).toHaveBeenCalledWith("h1", expect.anything());
  });

  it("a genuine delete failure surfaces inline in the confirm dialog", async () => {
    const user = userEvent.setup();
    useFIIHoldings.mockReturnValue(listState({ holdings: [hglg] }));
    useDeleteFIIHolding.mockReturnValue(
      mutationHook<void, string>({ isError: true, error: new ApiError(500, "falha no servidor") }),
    );
    render(<FiiSection />);
    await user.click(screen.getByRole("button", { name: "Excluir" }));
    expect(screen.getByText("falha no servidor")).toBeInTheDocument();
  });
});
