"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { useDashboard } from "@/lib/dashboard/dashboard";
import { SummaryHero } from "@/app/(app)/dashboard/summary-hero";
import { AllocationSections } from "@/app/(app)/dashboard/allocation-sections";

// The Painel screen (SPEC-212) — the frontend face of SPEC-103. Read-only: every figure comes
// straight from GET /dashboard (BR-2121). Health Score and AI Insights are SPEC-213's territory,
// not this screen's (SPEC-212 §2 Scope).
export default function DashboardPage() {
  const router = useRouter();
  const { dashboard, isLoading, isError, refetch } = useDashboard();

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-40 animate-pulse rounded-xl bg-elevated" />
        <div className="grid gap-4 sm:grid-cols-3">
          <div className="h-24 animate-pulse rounded-xl bg-elevated" />
          <div className="h-24 animate-pulse rounded-xl bg-elevated" />
          <div className="h-24 animate-pulse rounded-xl bg-elevated" />
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex flex-col items-center gap-4 py-16 text-center">
        <p className="text-sm text-loss">Não foi possível carregar seu painel.</p>
        <Button variant="secondary" onClick={() => refetch()}>
          Tentar novamente
        </Button>
      </div>
    );
  }

  if (!dashboard) return null; // unreachable once isLoading/isError are both false

  const { summary } = dashboard;
  const isEmpty = summary.total_invested_centavos === 0 && summary.current_value_centavos === 0;

  if (isEmpty) {
    return (
      <EmptyState
        title="Sua carteira está vazia"
        description="Adicione seus FIIs e posições de renda fixa na Carteira para ver seu painel."
      >
        <Button size="sm" onClick={() => router.push("/portfolio")}>
          Ir para a Carteira
        </Button>
      </EmptyState>
    );
  }

  return (
    <div className="space-y-6">
      <SummaryHero dashboard={dashboard} />
      <AllocationSections dashboard={dashboard} />
    </div>
  );
}
