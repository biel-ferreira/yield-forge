"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { EmptyState } from "@/components/ui/empty-state";
import { formatDateBR } from "@/lib/date";
import { formatBps, formatCentavos } from "@/lib/money";
import {
  useDeleteFixedIncomeHolding,
  useFixedIncomeHoldings,
  type FixedIncomeHolding,
} from "@/lib/portfolio/holdings";
import {
  INDEXER_LABELS,
  LIQUIDITY_LABELS,
  referenceIndicator,
  type Indexer,
} from "@/lib/portfolio/labels";
import { findIndicator, useMarketIndicators } from "@/lib/portfolio/market";
import { FixedIncomeForm } from "@/app/(app)/portfolio/fixed-income-form";

// The fixed-income vertical (SPEC-211 FR-2115…FR-2118/FR-2120): list, add/edit modal with the
// indexer picker + live reference, delete-confirm — mirrors fii-table.tsx's FiiSection shape.
export function FixedIncomeSection() {
  const { holdings, isLoading, isError, refetch } = useFixedIncomeHoldings();
  const { indicators } = useMarketIndicators();
  const del = useDeleteFixedIncomeHolding();

  const [formTarget, setFormTarget] = useState<"add" | FixedIncomeHolding | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<FixedIncomeHolding | null>(null);

  const sorted = [...holdings].sort((a, b) => a.name.localeCompare(b.name));

  return (
    <section>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="font-serif text-xl font-semibold text-on-dark">Renda fixa</h2>
        <Button size="sm" onClick={() => setFormTarget("add")}>
          Adicionar renda fixa
        </Button>
      </div>

      {isLoading && (
        <div className="space-y-2">
          <div className="h-12 animate-pulse rounded-lg bg-elevated" />
          <div className="h-12 animate-pulse rounded-lg bg-elevated" />
        </div>
      )}

      {isError && (
        <div className="flex flex-col items-center gap-3 py-10 text-center">
          <p className="text-sm text-loss">Não foi possível carregar sua renda fixa.</p>
          <Button variant="secondary" size="sm" onClick={() => refetch()}>
            Tentar novamente
          </Button>
        </div>
      )}

      {!isLoading && !isError && sorted.length === 0 && (
        <EmptyState
          title="Nenhuma renda fixa cadastrada"
          description="Adicione sua primeira posição em renda fixa."
        >
          <Button size="sm" onClick={() => setFormTarget("add")}>
            Adicionar renda fixa
          </Button>
        </EmptyState>
      )}

      {!isLoading && !isError && sorted.length > 0 && (
        <Card className="overflow-x-auto p-0">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-hairline text-left text-xs uppercase tracking-wide text-muted">
                <th className="px-4 py-3 font-semibold">Nome</th>
                <th className="px-4 py-3 font-semibold">Instituição</th>
                <th className="px-4 py-3 font-semibold">Valor investido</th>
                <th className="px-4 py-3 font-semibold">Taxa efetiva</th>
                <th className="px-4 py-3 font-semibold">Liquidez</th>
                <th className="px-4 py-3 font-semibold">Vencimento</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {sorted.map((holding) => (
                <tr key={holding.id} className="border-b border-hairline last:border-0">
                  <td className="px-4 py-3 font-semibold text-on-dark">{holding.name}</td>
                  <td className="px-4 py-3 text-body">{holding.institution}</td>
                  <td className="tabular px-4 py-3 text-body">
                    {formatCentavos(holding.invested_amount_centavos)}
                  </td>
                  <td className="tabular px-4 py-3 text-body">
                    <EffectiveRate holding={holding} indicators={indicators} />
                  </td>
                  <td className="px-4 py-3 text-body">
                    {LIQUIDITY_LABELS[holding.liquidity_type]}
                  </td>
                  <td className="px-4 py-3 text-body">
                    {holding.maturity_date ? formatDateBR(holding.maturity_date) : "—"}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex justify-end gap-2">
                      <Button variant="ghost" size="sm" onClick={() => setFormTarget(holding)}>
                        Editar
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(holding)}>
                        Excluir
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}

      <FixedIncomeForm
        // See fii-table.tsx's FiiForm key for why this is needed — without it, switching edit
        // targets (or reopening "add") would leak the previous session's stale form state.
        key={formTarget === null ? "closed" : formTarget === "add" ? "add" : formTarget.id}
        open={formTarget !== null}
        onClose={() => setFormTarget(null)}
        initial={formTarget === "add" ? null : formTarget}
      />

      <ConfirmDialog
        open={deleteTarget !== null}
        title="Excluir renda fixa?"
        description={
          deleteTarget
            ? `Remover "${deleteTarget.name}" da sua carteira? Esta ação não pode ser desfeita.`
            : ""
        }
        isPending={del.isPending}
        error={del.isError ? del.error.message : undefined}
        onConfirm={() => {
          if (!deleteTarget) return;
          del.mutate(deleteTarget.id, { onSuccess: () => setDeleteTarget(null) });
        }}
        onCancel={() => {
          setDeleteTarget(null);
          del.reset();
        }}
      />
    </section>
  );
}

// FR-2120: the resolved effective rate, with its reference date when the indicator is present —
// or "sem referência disponível" when it's entirely absent (never ingested / a transient fetch
// error). Deliberately no computed staleness threshold (D7) — the date alone lets the investor
// judge freshness.
function EffectiveRate({
  holding,
  indicators,
}: {
  holding: FixedIncomeHolding;
  indicators: ReturnType<typeof useMarketIndicators>["indicators"];
}) {
  const indexer: Indexer = holding.indexer_type ?? "prefixado";
  const effectiveBps = holding.effective_annual_rate_bps ?? holding.annual_rate_bps;
  const rateLabel =
    indexer === "prefixado"
      ? formatBps(effectiveBps)
      : `${INDEXER_LABELS[indexer]} · ${formatBps(effectiveBps)}`;

  const indicator = referenceIndicator(indexer);
  if (indicator === null) {
    return <span>{rateLabel}</span>;
  }

  const reference = findIndicator(indicators, indicator);
  return (
    <span>
      {rateLabel}
      <span className="ml-1.5 text-xs text-muted">
        {reference
          ? `(ref. ${formatDateBR(reference.reference_date)})`
          : "(sem referência disponível)"}
      </span>
    </span>
  );
}
