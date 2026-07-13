import { FiiSection } from "@/app/(app)/portfolio/fii-table";

// The Carteira screen (SPEC-211). FII section wired (Phase 3, live-verified against the running
// backend); the fixed-income section (Phase 4) still needs to be added alongside it (Phase 5).
export default function PortfolioPage() {
  return (
    <div className="space-y-8">
      <FiiSection />
    </div>
  );
}
