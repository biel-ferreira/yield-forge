import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import {
  useCreateFIIHolding,
  useDeleteFIIHolding,
  useCreateFixedIncomeHolding,
} from "@/lib/portfolio/holdings";
import { DASHBOARD_KEY } from "@/lib/dashboard/dashboard";
import { api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  api: { GET: vi.fn(), POST: vi.fn(), PUT: vi.fn(), DELETE: vi.fn() },
}));

function makeWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  }
  return { qc, wrapper };
}

beforeEach(() => {
  vi.clearAllMocks();
});

// SPEC-212 review finding: the Dashboard is a computed view *over* holdings, but none of the
// holdings mutations invalidated its cache — with a 30s global staleTime, a Carteira→Painel SPA
// navigation could silently serve pre-mutation figures. Fixed via invalidateHoldingsAndDashboard;
// these tests exercise the REAL hooks (unlike fii-table.test.tsx etc., which mock this module
// wholesale and so never actually ran this code path) to prove the fix, not just assert it.
describe("holdings mutations invalidate the dashboard cache alongside their own list", () => {
  it("useCreateFIIHolding (onSuccess) invalidates both FII_KEY and DASHBOARD_KEY", async () => {
    vi.mocked(api.POST).mockResolvedValue({
      data: { id: "h1", ticker: "HGLG11", quantity: 1, average_price_centavos: 100 },
      error: undefined,
      response: { status: 201 },
    } as never);
    const { qc, wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => useCreateFIIHolding(), { wrapper });

    result.current.mutate({ ticker: "HGLG11", quantity: 1, average_price_centavos: 100 });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const invalidatedKeys = invalidateSpy.mock.calls.map((call) => call[0]?.queryKey);
    expect(invalidatedKeys).toContainEqual(["holdings", "fii"]);
    expect(invalidatedKeys).toContainEqual(DASHBOARD_KEY);
  });

  it("useDeleteFIIHolding (onSettled, incl. the 404-as-success path) invalidates both keys", async () => {
    vi.mocked(api.DELETE).mockResolvedValue({
      error: undefined,
      response: { status: 204 },
    } as never);
    const { qc, wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => useDeleteFIIHolding(), { wrapper });

    result.current.mutate("h1");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const invalidatedKeys = invalidateSpy.mock.calls.map((call) => call[0]?.queryKey);
    expect(invalidatedKeys).toContainEqual(["holdings", "fii"]);
    expect(invalidatedKeys).toContainEqual(DASHBOARD_KEY);
  });

  it("useCreateFixedIncomeHolding (onSuccess) invalidates both FIXED_INCOME_KEY and DASHBOARD_KEY", async () => {
    vi.mocked(api.POST).mockResolvedValue({
      data: { id: "fi1", name: "CDB", institution: "Banco X" },
      error: undefined,
      response: { status: 201 },
    } as never);
    const { qc, wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => useCreateFixedIncomeHolding(), { wrapper });

    result.current.mutate({
      name: "CDB",
      institution: "Banco X",
      invested_amount_centavos: 100_000,
      annual_rate_bps: 1000,
      indexer_type: "prefixado",
      liquidity_type: "daily",
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const invalidatedKeys = invalidateSpy.mock.calls.map((call) => call[0]?.queryKey);
    expect(invalidatedKeys).toContainEqual(["holdings", "fixed-income"]);
    expect(invalidatedKeys).toContainEqual(DASHBOARD_KEY);
  });
});
