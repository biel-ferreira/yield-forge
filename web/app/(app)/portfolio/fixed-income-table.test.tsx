import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { UseMutationResult } from "@tanstack/react-query";
import { FixedIncomeSection } from "@/app/(app)/portfolio/fixed-income-table";
import * as holdingsHooks from "@/lib/portfolio/holdings";
import * as marketHooks from "@/lib/portfolio/market";
import { ApiError } from "@/lib/api/error";
import type { FixedIncomeHolding, FixedIncomeInput } from "@/lib/portfolio/holdings";

type UpdateVars = { id: string; input: FixedIncomeInput };

vi.mock("@/lib/portfolio/holdings", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/portfolio/holdings")>();
  return {
    ...actual,
    useFixedIncomeHoldings: vi.fn(),
    useCreateFixedIncomeHolding: vi.fn(),
    useUpdateFixedIncomeHolding: vi.fn(),
    useDeleteFixedIncomeHolding: vi.fn(),
  };
});
vi.mock("@/lib/portfolio/market", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/portfolio/market")>();
  return { ...actual, useMarketIndicators: vi.fn() };
});

const useFixedIncomeHoldings = vi.mocked(holdingsHooks.useFixedIncomeHoldings);
const useCreateFixedIncomeHolding = vi.mocked(holdingsHooks.useCreateFixedIncomeHolding);
const useUpdateFixedIncomeHolding = vi.mocked(holdingsHooks.useUpdateFixedIncomeHolding);
const useDeleteFixedIncomeHolding = vi.mocked(holdingsHooks.useDeleteFixedIncomeHolding);
const useMarketIndicators = vi.mocked(marketHooks.useMarketIndicators);

type ListState = ReturnType<typeof holdingsHooks.useFixedIncomeHoldings>;

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

const cdiHolding: FixedIncomeHolding = {
  id: "fi1",
  name: "CDB Banco X",
  institution: "Banco X",
  invested_amount_centavos: 1_000_000,
  annual_rate_bps: 12_000,
  indexer_type: "cdi_percentual",
  effective_annual_rate_bps: 1_260,
  maturity_date: null,
  liquidity_type: "daily",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};
const prefixadoHolding: FixedIncomeHolding = {
  id: "fi2",
  name: "Tesouro Prefixado",
  institution: "Tesouro Direto",
  invested_amount_centavos: 500_000,
  annual_rate_bps: 1000,
  indexer_type: "prefixado",
  effective_annual_rate_bps: 1000,
  maturity_date: "2030-01-01",
  liquidity_type: "at_maturity",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  useCreateFixedIncomeHolding.mockReturnValue(mutationHook<FixedIncomeHolding, FixedIncomeInput>());
  useUpdateFixedIncomeHolding.mockReturnValue(mutationHook<FixedIncomeHolding, UpdateVars>());
  useDeleteFixedIncomeHolding.mockReturnValue(mutationHook<void, string>());
  useMarketIndicators.mockReturnValue({ indicators: [], isLoading: false, isError: false });
});

describe("FixedIncomeSection — list states (SPEC-211 FR-2115)", () => {
  it("empty → empty state with a CTA, not an error", () => {
    useFixedIncomeHoldings.mockReturnValue(listState({ holdings: [] }));
    render(<FixedIncomeSection />);
    expect(screen.getByText("Nenhuma renda fixa cadastrada")).toBeInTheDocument();
  });

  it("error → retry calls refetch", async () => {
    const user = userEvent.setup();
    const refetch = vi.fn();
    useFixedIncomeHoldings.mockReturnValue(listState({ isError: true, refetch }));
    render(<FixedIncomeSection />);
    await user.click(screen.getByRole("button", { name: "Tentar novamente" }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("shows liquidity label and '—' for a Diária holding with no maturity", () => {
    useFixedIncomeHoldings.mockReturnValue(listState({ holdings: [cdiHolding] }));
    render(<FixedIncomeSection />);
    const row = screen.getAllByRole("row")[1];
    expect(within(row).getByText("Diária")).toBeInTheDocument();
    expect(within(row).getByText("—")).toBeInTheDocument();
  });

  it("shows a pt-BR maturity date for an at-maturity holding", () => {
    useFixedIncomeHoldings.mockReturnValue(listState({ holdings: [prefixadoHolding] }));
    render(<FixedIncomeSection />);
    expect(screen.getByText("01/01/2030")).toBeInTheDocument();
  });
});

describe("FixedIncomeSection — effective rate + reference date (SPEC-211 FR-2120/D7)", () => {
  it("shows the reference date when the indicator is present", () => {
    useFixedIncomeHoldings.mockReturnValue(listState({ holdings: [cdiHolding] }));
    useMarketIndicators.mockReturnValue({
      indicators: [{ indicator: "cdi", value_bps: 1050, reference_date: "2026-07-01" }],
      isLoading: false,
      isError: false,
    });
    render(<FixedIncomeSection />);
    expect(screen.getByText("% do CDI · 12,60%")).toBeInTheDocument();
    expect(screen.getByText("(ref. 01/07/2026)")).toBeInTheDocument();
  });

  it("shows 'sem referência disponível' when the indicator is entirely absent", () => {
    useFixedIncomeHoldings.mockReturnValue(listState({ holdings: [cdiHolding] }));
    useMarketIndicators.mockReturnValue({ indicators: [], isLoading: false, isError: false });
    render(<FixedIncomeSection />);
    expect(screen.getByText("(sem referência disponível)")).toBeInTheDocument();
  });

  it("prefixado never shows a reference — nothing to resolve against", () => {
    useFixedIncomeHoldings.mockReturnValue(listState({ holdings: [prefixadoHolding] }));
    useMarketIndicators.mockReturnValue({
      indicators: [{ indicator: "cdi", value_bps: 1050, reference_date: "2026-07-01" }],
      isLoading: false,
      isError: false,
    });
    render(<FixedIncomeSection />);
    expect(screen.getByText("10,00%")).toBeInTheDocument();
    expect(screen.queryByText(/ref\.|sem referência/)).not.toBeInTheDocument();
  });
});

describe("FixedIncomeSection — edit/delete wiring (SPEC-211 FR-2117/FR-2118, BR-2111)", () => {
  it("Editar opens the prefilled form", async () => {
    const user = userEvent.setup();
    useFixedIncomeHoldings.mockReturnValue(listState({ holdings: [cdiHolding] }));
    render(<FixedIncomeSection />);
    await user.click(screen.getByRole("button", { name: "Editar" }));
    expect(screen.getByRole("heading", { name: "Editar renda fixa" })).toBeInTheDocument();
    expect(screen.getByLabelText("Nome")).toHaveValue("CDB Banco X");
  });

  it("Excluir → confirm → deletes", async () => {
    const user = userEvent.setup();
    useFixedIncomeHoldings.mockReturnValue(listState({ holdings: [cdiHolding] }));
    const del = mutationHook<void, string>({ onCall: (_v, opts) => opts?.onSuccess?.(undefined) });
    useDeleteFixedIncomeHolding.mockReturnValue(del);
    render(<FixedIncomeSection />);
    await user.click(screen.getByRole("button", { name: "Excluir" }));
    const dialog = screen.getByRole("dialog", { name: "Excluir renda fixa?" });
    expect(within(dialog).getByText(/Remover "CDB Banco X"/)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Excluir" }));
    expect(del.mutate).toHaveBeenCalledWith("fi1", expect.anything());
  });

  it("a genuine delete failure surfaces inline", async () => {
    const user = userEvent.setup();
    useFixedIncomeHoldings.mockReturnValue(listState({ holdings: [cdiHolding] }));
    useDeleteFixedIncomeHolding.mockReturnValue(
      mutationHook<void, string>({ isError: true, error: new ApiError(500, "falha no servidor") }),
    );
    render(<FixedIncomeSection />);
    await user.click(screen.getByRole("button", { name: "Excluir" }));
    expect(screen.getByText("falha no servidor")).toBeInTheDocument();
  });
});
