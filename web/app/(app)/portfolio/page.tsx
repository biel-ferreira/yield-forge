import { FiiSection } from "@/app/(app)/portfolio/fii-table";
import { FixedIncomeSection } from "@/app/(app)/portfolio/fixed-income-table";

// The Carteira screen (SPEC-211) — the frontend face of SPEC-102 (holdings) + SPEC-109 (the
// fixed-income rate indexer). The shell's TopBar already renders the page-level "Carteira"
// heading (SPEC-200, route-derived from lib/shell/nav.ts); each section owns its own h2 + CTA,
// so there's no redundant top-level heading here. Not width-constrained like the narrower
// Perfil form (SPEC-210) — the fixed-income table has 7 columns and needs the room.
export default function PortfolioPage() {
  return (
    <div className="space-y-8">
      <FiiSection />
      <FixedIncomeSection />
    </div>
  );
}
