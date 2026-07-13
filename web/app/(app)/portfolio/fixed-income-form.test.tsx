import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { UseMutationResult } from "@tanstack/react-query";
import { FixedIncomeForm } from "@/app/(app)/portfolio/fixed-income-form";
import * as holdingsHooks from "@/lib/portfolio/holdings";
import * as marketHooks from "@/lib/portfolio/market";
import { ApiError } from "@/lib/api/error";
import type { FixedIncomeHolding, FixedIncomeInput } from "@/lib/portfolio/holdings";

type UpdateVars = { id: string; input: FixedIncomeInput };

vi.mock("@/lib/portfolio/holdings", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/portfolio/holdings")>();
  return { ...actual, useCreateFixedIncomeHolding: vi.fn(), useUpdateFixedIncomeHolding: vi.fn() };
});
vi.mock("@/lib/portfolio/market", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/portfolio/market")>();
  return { ...actual, useMarketIndicators: vi.fn() };
});

const useCreateFixedIncomeHolding = vi.mocked(holdingsHooks.useCreateFixedIncomeHolding);
const useUpdateFixedIncomeHolding = vi.mocked(holdingsHooks.useUpdateFixedIncomeHolding);
const useMarketIndicators = vi.mocked(marketHooks.useMarketIndicators);

// Generic over the mutation's data/variables shape — create/update each mutate with different
// types (mirrors the project's no-mocking-library convention: hand fakes, not magic).
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

const cdbHolding: FixedIncomeHolding = {
  id: "fi1",
  name: "CDB Banco X",
  institution: "Banco X",
  invested_amount_centavos: 1_000_000,
  annual_rate_bps: 12_000,
  indexer_type: "cdi_percentual",
  effective_annual_rate_bps: 1_260,
  maturity_date: "2030-01-01",
  liquidity_type: "at_maturity",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  useCreateFixedIncomeHolding.mockReturnValue(mutationHook<FixedIncomeHolding, FixedIncomeInput>());
  useUpdateFixedIncomeHolding.mockReturnValue(mutationHook<FixedIncomeHolding, UpdateVars>());
  useMarketIndicators.mockReturnValue({ indicators: [], isLoading: false, isError: false });
});

describe("FixedIncomeForm — indexer picker & rate label (SPEC-211 FR-2116)", () => {
  it("defaults to prefixado, with the flat-rate label and no live reference", () => {
    render(<FixedIncomeForm open onClose={() => {}} initial={null} />);
    expect(screen.getByLabelText("Taxa anual (%)")).toBeInTheDocument();
    expect(screen.queryByText(/atual:|indisponível/)).not.toBeInTheDocument();
  });

  it("switching to % do CDI relabels the rate input and shows the live reference", async () => {
    const user = userEvent.setup();
    useMarketIndicators.mockReturnValue({
      indicators: [{ indicator: "cdi", value_bps: 1050, reference_date: "2026-07-01" }],
      isLoading: false,
      isError: false,
    });
    render(<FixedIncomeForm open onClose={() => {}} initial={null} />);
    await user.click(screen.getByRole("radio", { name: "% do CDI" }));
    expect(screen.getByLabelText("% do CDI")).toBeInTheDocument();
    expect(screen.getByText("CDI atual: 10,50% a.a. (ref. 01/07/2026)")).toBeInTheDocument();
  });

  it("degrades to 'indisponível' without blocking the save path when the indicator is absent", async () => {
    const user = userEvent.setup();
    useMarketIndicators.mockReturnValue({ indicators: [], isLoading: false, isError: false });
    render(<FixedIncomeForm open onClose={() => {}} initial={null} />);
    await user.click(screen.getByRole("radio", { name: "IPCA +" }));
    expect(screen.getByText("IPCA indisponível no momento.")).toBeInTheDocument();
    // the save path itself is unaffected — filling the rest still enables Salvar
    await user.type(screen.getByLabelText("Nome"), "Tesouro IPCA+");
    await user.type(screen.getByLabelText("Instituição"), "Tesouro Direto");
    await user.type(screen.getByLabelText("Valor investido (R$)"), "1.000,00");
    await user.type(screen.getByLabelText("Spread sobre o IPCA (%)"), "5,80");
    expect(screen.getByRole("button", { name: "Salvar" })).toBeEnabled();
  });
});

describe("FixedIncomeForm — liquidity ↔ maturity (SPEC-211 FR-2116)", () => {
  it("Diária (default) shows no maturity field", () => {
    render(<FixedIncomeForm open onClose={() => {}} initial={null} />);
    expect(screen.queryByLabelText("Vencimento")).not.toBeInTheDocument();
  });

  it("No vencimento requires a maturity date before Salvar enables", async () => {
    const user = userEvent.setup();
    render(<FixedIncomeForm open onClose={() => {}} initial={null} />);
    await user.type(screen.getByLabelText("Nome"), "CDB");
    await user.type(screen.getByLabelText("Instituição"), "Banco X");
    await user.type(screen.getByLabelText("Valor investido (R$)"), "1.000,00");
    await user.type(screen.getByLabelText("Taxa anual (%)"), "10");
    await user.click(screen.getByRole("radio", { name: "No vencimento" }));
    expect(screen.getByLabelText("Vencimento")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  });

  it("switching back to Diária clears the maturity field", async () => {
    const user = userEvent.setup();
    render(<FixedIncomeForm open onClose={() => {}} initial={null} />);
    await user.click(screen.getByRole("radio", { name: "No vencimento" }));
    await user.type(screen.getByLabelText("Vencimento"), "2030-01-01");
    await user.click(screen.getByRole("radio", { name: "Diária" }));
    expect(screen.queryByLabelText("Vencimento")).not.toBeInTheDocument();
  });

  it("rejects a past maturity date on a NEW at-maturity holding, at the edge", async () => {
    // "2020-01-01" is unambiguously in the past regardless of when this test runs — no need to
    // fake the system clock (userEvent + fake timers is a known deadlock hazard).
    const user = userEvent.setup();
    render(<FixedIncomeForm open onClose={() => {}} initial={null} />);
    await user.type(screen.getByLabelText("Nome"), "CDB");
    await user.type(screen.getByLabelText("Instituição"), "Banco X");
    await user.type(screen.getByLabelText("Valor investido (R$)"), "1.000,00");
    await user.type(screen.getByLabelText("Taxa anual (%)"), "10");
    await user.click(screen.getByRole("radio", { name: "No vencimento" }));
    await user.type(screen.getByLabelText("Vencimento"), "2020-01-01");
    expect(screen.getByText("A data de vencimento não pode estar no passado.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  });
});

describe("FixedIncomeForm — edit mode (SPEC-211 FR-2117)", () => {
  it("prefills every field, including the correct indexer selection", () => {
    render(<FixedIncomeForm open onClose={() => {}} initial={cdbHolding} />);
    expect(screen.getByRole("heading", { name: "Editar renda fixa" })).toBeInTheDocument();
    expect(screen.getByLabelText("Nome")).toHaveValue("CDB Banco X");
    expect(screen.getByRole("radio", { name: "% do CDI" })).toBeChecked();
    expect(screen.getByLabelText("% do CDI")).toHaveValue("120,00");
    expect(screen.getByRole("radio", { name: "No vencimento" })).toBeChecked();
    expect(screen.getByLabelText("Vencimento")).toHaveValue("2030-01-01");
  });

  it("does not reject a past maturity date on an existing holding (edit-mode exempt)", () => {
    const pastMaturity: FixedIncomeHolding = { ...cdbHolding, maturity_date: "2020-01-01" };
    render(<FixedIncomeForm open onClose={() => {}} initial={pastMaturity} />);
    expect(
      screen.queryByText("A data de vencimento não pode estar no passado."),
    ).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Salvar" })).toBeEnabled();
  });

  it("submits {id, input} with maturity_date: null for a Diária holding", async () => {
    const user = userEvent.setup();
    const update = mutationHook<FixedIncomeHolding, UpdateVars>({
      onCall: (_v, opts) => opts?.onSuccess?.(cdbHolding),
    });
    useUpdateFixedIncomeHolding.mockReturnValue(update);
    const daily: FixedIncomeHolding = {
      ...cdbHolding,
      liquidity_type: "daily",
      maturity_date: null,
      indexer_type: "prefixado",
      annual_rate_bps: 1000,
    };
    render(<FixedIncomeForm open onClose={() => {}} initial={daily} />);
    await user.click(screen.getByRole("button", { name: "Salvar" }));
    expect(update.mutate).toHaveBeenCalledWith(
      {
        id: "fi1",
        input: {
          name: "CDB Banco X",
          institution: "Banco X",
          invested_amount_centavos: 1_000_000,
          annual_rate_bps: 1000,
          indexer_type: "prefixado",
          liquidity_type: "daily",
          maturity_date: null,
        },
      },
      expect.anything(),
    );
  });

  it("a 404 on update closes quietly — no scary error (BR-2111)", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    useUpdateFixedIncomeHolding.mockReturnValue(
      mutationHook<FixedIncomeHolding, UpdateVars>({
        onCall: (_v, opts) => opts?.onError?.(new ApiError(404, "not found")),
      }),
    );
    render(<FixedIncomeForm open onClose={onClose} initial={cdbHolding} />);
    await user.click(screen.getByRole("button", { name: "Salvar" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
