import { FiiSection } from "@/app/(app)/portfolio/fii-table";
import { FixedIncomeSection } from "@/app/(app)/portfolio/fixed-income-table";

// The Carteira screen (SPEC-211). Both sections wired (Phases 3/4, live-verified against the
// running backend); Phase 5 still owns the final polish (page heading, layout).
export default function PortfolioPage() {
  return (
    <div className="space-y-8">
      <FiiSection />
      <FixedIncomeSection />
    </div>
  );
}
