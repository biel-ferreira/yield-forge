import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { SummaryHero } from "@/app/(app)/dashboard/summary-hero";
import type { Dashboard } from "@/lib/dashboard/dashboard";

function dashboard(
  over: Partial<Dashboard["summary"]> = {},
  staleTickers: string[] = [],
): Dashboard {
  return {
    summary: {
      total_invested_centavos: 2_075_000,
      current_value_centavos: 2_100_000,
      monthly_income_centavos: 11_000,
      growth_centavos: 25_000,
      growth_bps: 120,
      ...over,
    },
    allocation: [],
    fii_sectors: [],
    stale_tickers: staleTickers,
  };
}

describe("SummaryHero — hero + metrics (SPEC-212 FR-2121/FR-2122)", () => {
  it("shows the current value as the headline figure", () => {
    render(<SummaryHero dashboard={dashboard()} />);
    expect(screen.getByText("R$ 21.000,00")).toBeInTheDocument();
  });

  it("a gain renders green with an up arrow", () => {
    render(<SummaryHero dashboard={dashboard({ growth_centavos: 25_000, growth_bps: 120 })} />);
    const badges = screen.getAllByText(/▲/);
    expect(badges.length).toBeGreaterThan(0);
    for (const b of badges) expect(b.closest("span")).toHaveClass("text-gain");
  });

  it("a loss renders red with a down arrow", () => {
    render(<SummaryHero dashboard={dashboard({ growth_centavos: -12_044, growth_bps: -250 })} />);
    const badges = screen.getAllByText(/▼/);
    expect(badges.length).toBeGreaterThan(0);
    for (const b of badges) expect(b.closest("span")).toHaveClass("text-loss");
  });

  it("zero growth renders neutral with no arrow", () => {
    render(<SummaryHero dashboard={dashboard({ growth_centavos: 0, growth_bps: 0 })} />);
    expect(screen.queryByText(/▲/)).not.toBeInTheDocument();
    expect(screen.queryByText(/▼/)).not.toBeInTheDocument();
  });

  it("the hero badge and the metric-row card show the identical growth figure (D1)", () => {
    render(<SummaryHero dashboard={dashboard({ growth_centavos: 25_000, growth_bps: 120 })} />);
    // Both the hero's inline badge and "Valorização" card render via the same GrowthFigure
    // component — the formatted centavos string must appear exactly twice, not diverge.
    expect(screen.getAllByText(/\+R\$ 250,00/)).toHaveLength(2);
    expect(screen.getAllByText(/\+1,20%/)).toHaveLength(2);
  });

  it("shows total invested and monthly income as their own metric cards", () => {
    render(<SummaryHero dashboard={dashboard()} />);
    expect(screen.getByText("R$ 20.750,00")).toBeInTheDocument();
    expect(screen.getByText("R$ 110,00")).toBeInTheDocument();
  });

  it("labels growth vs. cost basis, never as a monthly figure", () => {
    render(<SummaryHero dashboard={dashboard()} />);
    expect(screen.getByText("vs. custo de aquisição")).toBeInTheDocument();
    expect(screen.queryByText(/no mês/)).not.toBeInTheDocument();
  });
});

describe("SummaryHero — stale-ticker notice (SPEC-212 FR-2125)", () => {
  it("is absent when stale_tickers is empty", () => {
    render(<SummaryHero dashboard={dashboard({}, [])} />);
    expect(screen.queryByText(/cotação indisponível/)).not.toBeInTheDocument();
  });

  it("is present and lists the affected tickers when not empty", () => {
    render(<SummaryHero dashboard={dashboard({}, ["HGLG11", "KNRI11"])} />);
    expect(screen.getByText(/HGLG11, KNRI11/)).toBeInTheDocument();
  });
});
