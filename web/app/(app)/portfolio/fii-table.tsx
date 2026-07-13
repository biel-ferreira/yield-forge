"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { EmptyState } from "@/components/ui/empty-state";
import { formatCentavos } from "@/lib/money";
import { useDeleteFIIHolding, useFIIHoldings, type FIIHolding } from "@/lib/portfolio/holdings";
import { FiiForm } from "@/app/(app)/portfolio/fii-form";

// The FII vertical (SPEC-211 FR-2111…FR-2114): list, add/edit modal, delete-confirm — all owned
// here so app/(app)/portfolio/page.tsx (Phase 5) just composes this section + the fixed-income
// one, mirroring how app/(app)/profile/page.tsx owns its own load/form state end to end.
export function FiiSection() {
  const { holdings, isLoading, isError, refetch } = useFIIHoldings();
  const del = useDeleteFIIHolding();

  const [formTarget, setFormTarget] = useState<"add" | FIIHolding | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<FIIHolding | null>(null);

  const sorted = [...holdings].sort((a, b) => a.ticker.localeCompare(b.ticker));

  return (
    <section>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="font-serif text-xl font-semibold text-on-dark">FIIs</h2>
        <Button size="sm" onClick={() => setFormTarget("add")}>
          Adicionar FII
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
          <p className="text-sm text-loss">Não foi possível carregar seus FIIs.</p>
          <Button variant="secondary" size="sm" onClick={() => refetch()}>
            Tentar novamente
          </Button>
        </div>
      )}

      {!isLoading && !isError && sorted.length === 0 && (
        <EmptyState
          title="Nenhum FII cadastrado"
          description="Adicione sua primeira posição em FIIs."
        >
          <Button size="sm" onClick={() => setFormTarget("add")}>
            Adicionar FII
          </Button>
        </EmptyState>
      )}

      {!isLoading && !isError && sorted.length > 0 && (
        <Card className="overflow-hidden p-0">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-hairline text-left text-xs uppercase tracking-wide text-muted">
                <th className="px-4 py-3 font-semibold">Ticker</th>
                <th className="px-4 py-3 font-semibold">Quantidade</th>
                <th className="px-4 py-3 font-semibold">Preço médio</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {sorted.map((holding) => (
                <tr key={holding.id} className="border-b border-hairline last:border-0">
                  <td className="px-4 py-3 font-semibold text-on-dark">{holding.ticker}</td>
                  <td className="tabular px-4 py-3 text-body">{holding.quantity}</td>
                  <td className="tabular px-4 py-3 text-body">
                    {formatCentavos(holding.average_price_centavos)}
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

      <FiiForm
        open={formTarget !== null}
        onClose={() => setFormTarget(null)}
        initial={formTarget === "add" ? null : formTarget}
      />

      <ConfirmDialog
        open={deleteTarget !== null}
        title="Excluir FII?"
        description={
          deleteTarget
            ? `Remover ${deleteTarget.ticker} da sua carteira? Esta ação não pode ser desfeita.`
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
