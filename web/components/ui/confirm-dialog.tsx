import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";

// A delete confirmation (SPEC-211 D2), built on Dialog (D1) — the one confirm pattern reused
// for both FII and fixed-income deletes (FR-2114/FR-2118). Requires an explicit confirmation
// click; there is no accidental one-click destroy.
export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel = "Excluir",
  cancelLabel = "Cancelar",
  isPending = false,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  isPending?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <Dialog open={open} onClose={onCancel} title={title}>
      <p className="text-sm text-muted-strong">{description}</p>
      <div className="mt-6 flex justify-end gap-3">
        <Button variant="secondary" size="sm" onClick={onCancel} disabled={isPending}>
          {cancelLabel}
        </Button>
        <Button variant="destructive" size="sm" onClick={onConfirm} disabled={isPending}>
          {confirmLabel}
        </Button>
      </div>
    </Dialog>
  );
}
