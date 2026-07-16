import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AllocationSections } from "@/app/(app)/dashboard/allocation-sections";
import type { Dashboard } from "@/lib/dashboard/dashboard";

function dashboard(over: Partial<Dashboard> = {}): Dashboard {
  return {
    summary: {
      total_invested_centavos: 0,
      current_value_centavos: 0,
      monthly_income_centavos: 0,
      growth_centavos: 0,
      growth_bps: 0,
    },
    allocation: [
      { asset_class: "fii", value_centavos: 1_600_000, share_bps: 7619 },
      { asset_class: "fixed_income", value_centavos: 500_000, share_bps: 2381 },
      { asset_class: "stocks", value_centavos: 0, share_bps: 0 },
      { asset_class: "etfs", value_centavos: 0, share_bps: 0 },
    ],
    fii_sectors: [{ sector: "logistics", value_centavos: 1_600_000, share_bps: 10000 }],
    stale_tickers: [],
    ...over,
  };
}

describe("AllocationSections — asset-class allocation (SPEC-212 FR-2123)", () => {
  it("renders every non-zero class with its pt-BR label and share", () => {
    render(<AllocationSections dashboard={dashboard()} />);
    expect(screen.getByText("FIIs")).toBeInTheDocument();
    expect(screen.getByText("Renda fixa")).toBeInTheDocument();
  });

  it("omits a zero-share class from the legend (D7)", () => {
    render(<AllocationSections dashboard={dashboard()} />);
    expect(screen.queryByText("Ações")).not.toBeInTheDocument();
    expect(screen.queryByText("ETFs")).not.toBeInTheDocument();
  });

  it("a single-class portfolio renders a full-width segment without error", () => {
    const onlyFii = dashboard({
      allocation: [
        { asset_class: "fii", value_centavos: 1_000_000, share_bps: 10000 },
        { asset_class: "fixed_income", value_centavos: 0, share_bps: 0 },
        { asset_class: "stocks", value_centavos: 0, share_bps: 0 },
        { asset_class: "etfs", value_centavos: 0, share_bps: 0 },
      ],
    });
    render(<AllocationSections dashboard={onlyFii} />);
    expect(screen.getByText("FIIs")).toBeInTheDocument();
    // The fixture's fii_sectors (also 100% Logística) means "100%" legitimately appears twice —
    // asserting presence, not uniqueness, is the actual intent here (a single segment renders
    // cleanly, no divide-by-zero/NaN-width crash).
    expect(screen.getAllByText("100%").length).toBeGreaterThan(0);
  });
});

describe("AllocationSections — FII sector exposure (SPEC-212 FR-2124)", () => {
  it("renders sectors with their pt-BR labels when the investor holds FIIs", () => {
    render(<AllocationSections dashboard={dashboard()} />);
    expect(screen.getByText("Exposição por setor (FIIs)")).toBeInTheDocument();
    expect(screen.getByText("Logística")).toBeInTheDocument();
  });

  it("omits the whole section for a fixed-income-only portfolio, not shown empty", () => {
    const fixedIncomeOnly = dashboard({
      allocation: [
        { asset_class: "fii", value_centavos: 0, share_bps: 0 },
        { asset_class: "fixed_income", value_centavos: 500_000, share_bps: 10000 },
        { asset_class: "stocks", value_centavos: 0, share_bps: 0 },
        { asset_class: "etfs", value_centavos: 0, share_bps: 0 },
      ],
      fii_sectors: [],
    });
    render(<AllocationSections dashboard={fixedIncomeOnly} />);
    expect(screen.queryByText("Exposição por setor (FIIs)")).not.toBeInTheDocument();
  });

  it("labels the 'other' (unknown/no-quote) sector distinctly", () => {
    const withOther = dashboard({
      fii_sectors: [{ sector: "other", value_centavos: 1_600_000, share_bps: 10000 }],
    });
    render(<AllocationSections dashboard={withOther} />);
    expect(screen.getByText("Outros / sem cotação")).toBeInTheDocument();
  });
});
