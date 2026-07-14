"use client";

import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api/error";
import { centavosToInputString, parseCentavos } from "@/lib/money";
import {
  useCreateFIIHolding,
  useUpdateFIIHolding,
  type FIIHolding,
  type FIIHoldingInput,
} from "@/lib/portfolio/holdings";

// Add/edit an FII holding (SPEC-211 FR-2112/FR-2113) — one form, both modes (mirrors
// app/(app)/profile/page.tsx's ProfileForm pattern): `initial` present = edit (prefilled,
// PUT); absent = add (blank, POST). Ticker is free text, uppercase-normalized client-side
// (D3) — the backend does not validate it against a known universe at creation.
export function FiiForm({
  open,
  onClose,
  initial,
}: {
  open: boolean;
  onClose: () => void;
  initial?: FIIHolding | null;
}) {
  const isEdit = initial != null;
  const create = useCreateFIIHolding();
  const update = useUpdateFIIHolding();
  const mutation = isEdit ? update : create;

  const [ticker, setTicker] = useState(initial?.ticker ?? "");
  const [quantity, setQuantity] = useState(initial ? String(initial.quantity) : "");
  const [price, setPrice] = useState(
    initial ? centavosToInputString(initial.average_price_centavos) : "",
  );

  const quantityValue = Number(quantity);
  const priceCentavos = parseCentavos(price);
  const valid =
    ticker.trim().length > 0 &&
    Number.isInteger(quantityValue) &&
    quantityValue >= 1 &&
    priceCentavos !== null &&
    priceCentavos >= 0;

  function onSubmit(event: FormEvent) {
    event.preventDefault();
    if (!valid || priceCentavos === null) return;
    const input: FIIHoldingInput = {
      ticker: ticker.trim().toUpperCase(),
      quantity: quantityValue,
      average_price_centavos: priceCentavos,
    };
    if (isEdit && initial) {
      update.mutate(
        { id: initial.id, input },
        {
          onSuccess: onClose,
          onError: (error) => {
            // A 404 means "already deleted / not owned elsewhere" (BR-2111) — the list already
            // refreshes via the hook's onSettled; close quietly instead of alarming the user.
            if (error instanceof ApiError && error.status === 404) onClose();
          },
        },
      );
    } else {
      create.mutate(input, { onSuccess: onClose });
    }
  }

  return (
    <Dialog open={open} onClose={onClose} title={isEdit ? "Editar FII" : "Adicionar FII"}>
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label htmlFor="fii-ticker" className="mb-1.5 block text-xs font-medium text-muted">
            Ticker
          </label>
          <Input
            id="fii-ticker"
            value={ticker}
            onChange={(event) => setTicker(event.target.value.toUpperCase())}
            placeholder="HGLG11"
            maxLength={12}
            autoFocus
          />
        </div>
        <div>
          <label htmlFor="fii-quantity" className="mb-1.5 block text-xs font-medium text-muted">
            Quantidade (cotas)
          </label>
          <Input
            id="fii-quantity"
            type="number"
            min={1}
            step={1}
            value={quantity}
            onChange={(event) => setQuantity(event.target.value)}
          />
        </div>
        <div>
          <label htmlFor="fii-price" className="mb-1.5 block text-xs font-medium text-muted">
            Preço médio (R$)
          </label>
          <Input
            id="fii-price"
            value={price}
            onChange={(event) => setPrice(event.target.value)}
            placeholder="157,50"
            inputMode="decimal"
          />
        </div>

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
