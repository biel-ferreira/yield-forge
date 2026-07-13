"use client";

import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Segmented } from "@/components/ui/segmented";
import { ApiError } from "@/lib/api/error";
import { formatDateBR, todayISO } from "@/lib/date";
import { formatBps, parseBps, parseCentavos } from "@/lib/money";
import {
  useCreateFixedIncomeHolding,
  useUpdateFixedIncomeHolding,
  type FixedIncomeHolding,
  type FixedIncomeInput,
} from "@/lib/portfolio/holdings";
import { findIndicator, useMarketIndicators } from "@/lib/portfolio/market";
import {
  INDEXER_LABELS,
  INDEXERS,
  INDICATOR_LABELS,
  LIQUIDITY_LABELS,
  LIQUIDITY_TYPES,
  referenceIndicator,
  type Indexer,
  type LiquidityType,
} from "@/lib/portfolio/labels";

// The rate-value input's label per indexer (FR-2116) — same wire field (annual_rate_bps),
// different meaning depending on indexer_type (SPEC-109).
const RATE_LABELS: Record<Indexer, string> = {
  prefixado: "Taxa anual (%)",
  cdi_percentual: "% do CDI",
  ipca_spread: "Spread sobre o IPCA (%)",
};

// Add/edit a fixed-income holding (SPEC-211 FR-2116/FR-2117) — one form, both modes, mirroring
// fii-form.tsx. Adds the SPEC-109 indexer picker + live reference display (FR-2120) on top of
// the FII form's shape.
export function FixedIncomeForm({
  open,
  onClose,
  initial,
}: {
  open: boolean;
  onClose: () => void;
  initial?: FixedIncomeHolding | null;
}) {
  const isEdit = initial != null;
  const create = useCreateFixedIncomeHolding();
  const update = useUpdateFixedIncomeHolding();
  const mutation = isEdit ? update : create;
  const { indicators } = useMarketIndicators();

  const [name, setName] = useState(initial?.name ?? "");
  const [institution, setInstitution] = useState(initial?.institution ?? "");
  const [amount, setAmount] = useState(
    initial ? (initial.invested_amount_centavos / 100).toFixed(2).replace(".", ",") : "",
  );
  const [indexer, setIndexer] = useState<Indexer>(initial?.indexer_type ?? "prefixado");
  const [rate, setRate] = useState(
    initial ? (initial.annual_rate_bps / 100).toFixed(2).replace(".", ",") : "",
  );
  const [liquidity, setLiquidity] = useState<LiquidityType>(initial?.liquidity_type ?? "daily");
  const [maturity, setMaturity] = useState(initial?.maturity_date ?? "");

  const amountCentavos = parseCentavos(amount);
  const rateBps = parseBps(rate);
  const requiresMaturity = liquidity === "at_maturity";
  // The create-time past-date rule (SPEC-102) only applies to a NEW at-maturity holding — an
  // existing one may have legitimately matured since it was recorded (FR-2116).
  const maturityInPast = !isEdit && requiresMaturity && maturity !== "" && maturity < todayISO();

  const valid =
    name.trim().length > 0 &&
    institution.trim().length > 0 &&
    amountCentavos !== null &&
    amountCentavos >= 1 &&
    rateBps !== null &&
    rateBps >= 0 &&
    (!requiresMaturity || (maturity !== "" && !maturityInPast));

  function onLiquidityChange(value: LiquidityType) {
    setLiquidity(value);
    if (value === "daily") setMaturity("");
  }

  function onSubmit(event: FormEvent) {
    event.preventDefault();
    if (!valid || amountCentavos === null || rateBps === null) return;
    const input: FixedIncomeInput = {
      name: name.trim(),
      institution: institution.trim(),
      invested_amount_centavos: amountCentavos,
      annual_rate_bps: rateBps,
      indexer_type: indexer,
      liquidity_type: liquidity,
      maturity_date: requiresMaturity ? maturity : null,
    };
    if (isEdit && initial) {
      update.mutate(
        { id: initial.id, input },
        {
          onSuccess: onClose,
          onError: (error) => {
            if (error instanceof ApiError && error.status === 404) onClose();
          },
        },
      );
    } else {
      create.mutate(input, { onSuccess: onClose });
    }
  }

  const refIndicator = referenceIndicator(indexer);
  const liveReference = refIndicator ? findIndicator(indicators, refIndicator) : undefined;

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={isEdit ? "Editar renda fixa" : "Adicionar renda fixa"}
      className="max-w-lg"
    >
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label htmlFor="fi-name" className="mb-1.5 block text-xs font-medium text-muted">
            Nome
          </label>
          <Input
            id="fi-name"
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="CDB Banco X"
            autoFocus
          />
        </div>
        <div>
          <label htmlFor="fi-institution" className="mb-1.5 block text-xs font-medium text-muted">
            Instituição
          </label>
          <Input
            id="fi-institution"
            value={institution}
            onChange={(event) => setInstitution(event.target.value)}
            placeholder="Banco X"
          />
        </div>
        <div>
          <label htmlFor="fi-amount" className="mb-1.5 block text-xs font-medium text-muted">
            Valor investido (R$)
          </label>
          <Input
            id="fi-amount"
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
            placeholder="1.000,00"
            inputMode="decimal"
          />
        </div>

        <div>
          <label className="mb-1.5 block text-xs font-medium text-muted">Indexador</label>
          <Segmented
            ariaLabel="Indexador"
            value={indexer}
            onChange={setIndexer}
            options={INDEXERS.map((v) => ({ value: v, label: INDEXER_LABELS[v] }))}
          />
          {refIndicator && (
            <p className="mt-2 text-xs text-muted-strong">
              {liveReference
                ? `${INDICATOR_LABELS[refIndicator]} atual: ${formatBps(liveReference.value_bps)} a.a. (ref. ${formatDateBR(liveReference.reference_date)})`
                : `${INDICATOR_LABELS[refIndicator]} indisponível no momento.`}
            </p>
          )}
        </div>

        <div>
          <label htmlFor="fi-rate" className="mb-1.5 block text-xs font-medium text-muted">
            {RATE_LABELS[indexer]}
          </label>
          <Input
            id="fi-rate"
            value={rate}
            onChange={(event) => setRate(event.target.value)}
            placeholder="10,50"
            inputMode="decimal"
          />
        </div>

        <div>
          <label className="mb-1.5 block text-xs font-medium text-muted">Liquidez</label>
          <Segmented
            ariaLabel="Liquidez"
            value={liquidity}
            onChange={onLiquidityChange}
            options={LIQUIDITY_TYPES.map((v) => ({ value: v, label: LIQUIDITY_LABELS[v] }))}
          />
        </div>

        {requiresMaturity && (
          <div>
            <label htmlFor="fi-maturity" className="mb-1.5 block text-xs font-medium text-muted">
              Vencimento
            </label>
            <Input
              id="fi-maturity"
              type="date"
              value={maturity}
              onChange={(event) => setMaturity(event.target.value)}
            />
            {maturityInPast && (
              <p className="mt-1.5 text-xs text-loss">
                A data de vencimento não pode estar no passado.
              </p>
            )}
          </div>
        )}

        <div className="flex items-center gap-4 pt-2">
          <Button type="submit" disabled={!valid || mutation.isPending}>
            {mutation.isPending ? "Salvando…" : "Salvar"}
          </Button>
          {mutation.isError && <span className="text-xs text-loss">{mutation.error.message}</span>}
        </div>
      </form>
    </Dialog>
  );
}
