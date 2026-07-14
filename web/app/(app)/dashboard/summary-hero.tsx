import { Card } from "@/components/ui/card";
import { formatBps, formatCentavos } from "@/lib/money";
import type { Dashboard } from "@/lib/dashboard/dashboard";

// The growth figure (FR-2121's hero badge and FR-2122's metric-row card both call this — SPEC-212
// D1: one shared formatting path, never two independent reads that could visually drift apart).
// Growth is vs. cost basis (current_value − total_invested), NOT a monthly figure — the backend
// tracks no history/time series, so this is never labelled as "no mês" (a design-mockup
// inaccuracy this spec deliberately does not repeat, see SPEC-212 FR-2121).
function GrowthFigure({ centavos, bps }: { centavos: number; bps: number }) {
  if (centavos === 0) {
    return (
      <span className="tabular font-mono text-sm text-muted-strong">
        {formatCentavos(0)} · {formatBps(0)}
      </span>
    );
  }
  const gain = centavos > 0;
  const sign = gain ? "+" : "";
  return (
    <span className={`tabular font-mono text-sm ${gain ? "text-gain" : "text-loss"}`}>
      {gain ? "▲" : "▼"} {sign}
      {formatCentavos(centavos)} · {sign}
      {formatBps(bps)}
    </span>
  );
}

// The hero patrimony card + key-metrics row + stale-ticker notice (SPEC-212 FR-2121/2122/2125) —
// everything derived from `summary` and `stale_tickers`, nothing computed client-side (BR-2121).
export function SummaryHero({ dashboard }: { dashboard: Dashboard }) {
  const { summary, stale_tickers } = dashboard;

  return (
    <div className="space-y-4">
      <Card className="p-6">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted">Patrimônio total</p>
        <p className="tabular mt-1 font-mono text-4xl font-semibold text-on-dark">
          {formatCentavos(summary.current_value_centavos)}
        </p>
        <div className="mt-2">
          <GrowthFigure centavos={summary.growth_centavos} bps={summary.growth_bps} />
          <span className="ml-1.5 text-xs text-muted">vs. custo de aquisição</span>
        </div>
      </Card>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card className="p-5">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted">
            Total investido
          </p>
          <p className="tabular mt-2 font-mono text-xl font-semibold text-on-dark">
            {formatCentavos(summary.total_invested_centavos)}
          </p>
        </Card>
        <Card className="p-5">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted">
            Renda passiva / mês
          </p>
          <p className="tabular mt-2 font-mono text-xl font-semibold text-on-dark">
            {formatCentavos(summary.monthly_income_centavos)}
          </p>
        </Card>
        <Card className="p-5">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted">Valorização</p>
          <p className="mt-2">
            <GrowthFigure centavos={summary.growth_centavos} bps={summary.growth_bps} />
          </p>
        </Card>
      </div>

      {stale_tickers.length > 0 && (
        <div className="rounded-lg border border-hairline bg-elevated px-4 py-3 text-xs text-muted-strong">
          Avaliado(s) pelo custo — cotação indisponível: {stale_tickers.join(", ")}
        </div>
      )}
    </div>
  );
}
