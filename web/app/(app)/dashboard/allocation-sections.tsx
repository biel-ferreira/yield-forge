import { Card } from "@/components/ui/card";
import { AllocationBar, type AllocationSegment } from "@/components/allocation-bar";
import { sectorLabel } from "@/lib/dashboard/labels";
import type { Dashboard } from "@/lib/dashboard/dashboard";

// Positional color assignment (mirrors the styleguide's own AllocationBar usage) — the backend
// already emits both `allocation` and `fii_sectors` in a fixed, stable order (SPEC-103
// compute.go: a `sectorOrder` slice, not map iteration), so index-based colors stay stable
// across fetches without this screen imposing its own sort (BR-2121: read, don't reorder).
const AURORA_COLORS = [
  "var(--aurora-1)",
  "var(--aurora-2)",
  "var(--aurora-3)",
  "var(--aurora-4)",
  "var(--aurora-5)",
];

function auroraColor(index: number): string {
  return AURORA_COLORS[index % AURORA_COLORS.length];
}

const ASSET_CLASS_LABELS: Record<Dashboard["allocation"][number]["asset_class"], string> = {
  fii: "FIIs",
  fixed_income: "Renda fixa",
  stocks: "Ações",
  etfs: "ETFs",
};

// Asset-class allocation (FR-2123) + FII sector exposure (FR-2124), SPEC-212 D4: both reuse the
// existing AllocationBar (SPEC-200) rather than a new chart component.
export function AllocationSections({ dashboard }: { dashboard: Dashboard }) {
  // A zero-share class (Stocks/ETFs, always 0 in the MVP) is never shown as a confusing
  // zero-width segment (D7) — still real data, just not worth a legend row.
  const classSegments: AllocationSegment[] = dashboard.allocation
    .filter((entry) => entry.share_bps > 0)
    .map((entry, index) => ({
      label: ASSET_CLASS_LABELS[entry.asset_class],
      bps: entry.share_bps,
      color: auroraColor(index),
    }));

  const sectorSegments: AllocationSegment[] = dashboard.fii_sectors.map((entry, index) => ({
    label: sectorLabel(entry.sector),
    bps: entry.share_bps,
    color: auroraColor(index),
  }));

  // FII sector exposure only makes sense when the investor actually holds FIIs (FR-2124) — a
  // fixed-income-only portfolio omits the section entirely rather than showing it empty.
  const hasFiiExposure = dashboard.allocation.some(
    (entry) => entry.asset_class === "fii" && entry.value_centavos > 0,
  );

  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Card className="p-6">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-muted">
          Alocação por classe de ativo
        </h3>
        <AllocationBar segments={classSegments} className="mt-4" />
      </Card>

      {hasFiiExposure && (
        <Card className="p-6">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-muted">
            Exposição por setor (FIIs)
          </h3>
          <AllocationBar segments={sectorSegments} className="mt-4" />
        </Card>
      )}
    </div>
  );
}
