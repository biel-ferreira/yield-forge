import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import DashboardPage from "@/app/(app)/dashboard/page";
import * as dashboardHooks from "@/lib/dashboard/dashboard";
import type { Dashboard } from "@/lib/dashboard/dashboard";

vi.mock("@/lib/dashboard/dashboard", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/dashboard/dashboard")>();
  return { ...actual, useDashboard: vi.fn() };
});

const push = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));

const useDashboard = vi.mocked(dashboardHooks.useDashboard);

type DashboardState = ReturnType<typeof dashboardHooks.useDashboard>;

function state(over: Partial<DashboardState> = {}): DashboardState {
  return { dashboard: null, isLoading: false, isError: false, refetch: vi.fn(), ...over };
}

const populated: Dashboard = {
  summary: {
    total_invested_centavos: 2_075_000,
    current_value_centavos: 2_100_000,
    monthly_income_centavos: 11_000,
    growth_centavos: 25_000,
    growth_bps: 120,
  },
  allocation: [
    { asset_class: "fii", value_centavos: 1_600_000, share_bps: 7619 },
    { asset_class: "fixed_income", value_centavos: 500_000, share_bps: 2381 },
    { asset_class: "stocks", value_centavos: 0, share_bps: 0 },
    { asset_class: "etfs", value_centavos: 0, share_bps: 0 },
  ],
  fii_sectors: [{ sector: "logistics", value_centavos: 1_600_000, share_bps: 10000 }],
  stale_tickers: [],
};

const empty: Dashboard = {
  summary: {
    total_invested_centavos: 0,
    current_value_centavos: 0,
    monthly_income_centavos: 0,
    growth_centavos: 0,
    growth_bps: 0,
  },
  allocation: [
    { asset_class: "fii", value_centavos: 0, share_bps: 0 },
    { asset_class: "fixed_income", value_centavos: 0, share_bps: 0 },
    { asset_class: "stocks", value_centavos: 0, share_bps: 0 },
    { asset_class: "etfs", value_centavos: 0, share_bps: 0 },
  ],
  fii_sectors: [],
  stale_tickers: [],
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("DashboardPage — load states (SPEC-212 FR-2126/FR-2127)", () => {
  it("loading → skeleton, no data section rendered", () => {
    useDashboard.mockReturnValue(state({ isLoading: true }));
    render(<DashboardPage />);
    expect(screen.queryByText("Patrimônio total")).not.toBeInTheDocument();
    expect(screen.queryByText("Sua carteira está vazia")).not.toBeInTheDocument();
  });

  it("error → retry button calls refetch", async () => {
    const user = userEvent.setup();
    const refetch = vi.fn();
    useDashboard.mockReturnValue(state({ isError: true, refetch }));
    render(<DashboardPage />);
    await user.click(screen.getByRole("button", { name: "Tentar novamente" }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("empty portfolio → the dedicated empty state, not a zeroed dashboard", () => {
    useDashboard.mockReturnValue(state({ dashboard: empty }));
    render(<DashboardPage />);
    expect(screen.getByText("Sua carteira está vazia")).toBeInTheDocument();
    expect(screen.queryByText("Patrimônio total")).not.toBeInTheDocument();
  });

  it("the empty state's CTA navigates to /portfolio", async () => {
    const user = userEvent.setup();
    useDashboard.mockReturnValue(state({ dashboard: empty }));
    render(<DashboardPage />);
    await user.click(screen.getByRole("button", { name: "Ir para a Carteira" }));
    expect(push).toHaveBeenCalledWith("/portfolio");
  });

  it("populated → both the summary and allocation sections render", () => {
    useDashboard.mockReturnValue(state({ dashboard: populated }));
    render(<DashboardPage />);
    expect(screen.getByText("Patrimônio total")).toBeInTheDocument();
    expect(screen.getByText("Alocação por classe de ativo")).toBeInTheDocument();
    expect(screen.queryByText("Sua carteira está vazia")).not.toBeInTheDocument();
  });
});
