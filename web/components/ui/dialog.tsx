"use client";

import { useEffect, useId, useRef, type ReactNode } from "react";
import { IconClose } from "@/components/shell/icons";
import { cn } from "@/lib/cn";

// A modal dialog built on the native <dialog> element (SPEC-211 D1) — showModal()/close() give
// focus-trap, Escape-to-close, and top-layer backdrop blocking for free, with zero new
// dependency (mirrors the slider's native-primitive choice, SPEC-210 PLAN D5). A dialog without
// a `title` is unrepresentable in this component's prop contract — every dialog is labelled by
// construction, the same binding-guard-style discipline InsightCard applies to `explanation`.
export function Dialog({
  open,
  onClose,
  title,
  children,
  className,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  className?: string;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      aria-labelledby={titleId}
      onClose={onClose}
      onClick={(event) => {
        // The <dialog> element itself has no padding (all content lives in the inner div
        // below), so a click reporting the dialog as its own target can only be a backdrop
        // click — never a click inside the visible modal.
        if (event.target === ref.current) onClose();
      }}
      className={cn(
        "fixed inset-0 m-auto rounded-lg border border-hairline bg-surface p-0 text-on-dark",
        "backdrop:bg-black/60",
        className,
      )}
    >
      <div className="w-full max-w-md p-6">
        <div className="mb-4 flex items-center justify-between gap-4">
          <h2 id={titleId} className="text-lg font-semibold">
            {title}
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Fechar"
            className="rounded-md p-1 text-muted-strong transition-colors hover:bg-elevated hover:text-on-dark focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-info"
          >
            <IconClose />
          </button>
        </div>
        {children}
      </div>
    </dialog>
  );
}
