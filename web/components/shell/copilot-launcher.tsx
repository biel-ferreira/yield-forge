"use client";

import { useState } from "react";
import { IconClose, IconCopilot } from "./icons";
import { NonAdviceDisclaimer } from "@/components/non-advice-disclaimer";

// The GLOBAL floating copilot launcher. SPEC-200 owns only the mount point + open/closed
// shell state; the conversational widget internals (streaming turns, etc.) are SPEC-215.
export function CopilotLauncher() {
  const [open, setOpen] = useState(false);

  return (
    <div className="fixed bottom-5 right-5 z-50 md:bottom-7 md:right-7">
      {open ? (
        <div
          className="flex w-[min(92vw,380px)] flex-col overflow-hidden rounded-[20px] border border-hairline bg-surface"
          style={{
            boxShadow: "0 40px 90px -28px rgba(0,0,0,.75), 0 0 70px -22px rgba(233,169,76,.4)",
          }}
        >
          <div
            className="flex items-center gap-2.5 border-b border-hairline px-3.5 py-3"
            style={{ background: "linear-gradient(180deg, rgba(233,169,76,.10), transparent)" }}
          >
            <span
              className="h-7 w-7 flex-none rounded-lg"
              style={{
                background: "linear-gradient(135deg, var(--primary), var(--aurora-2))",
                boxShadow: "0 0 18px -3px rgba(233,169,76,.7)",
              }}
            />
            <div className="flex-1">
              <div className="text-sm font-semibold text-on-dark">Copiloto</div>
              <div className="text-[11px] text-muted">Pergunte sobre sua carteira</div>
            </div>
            <button
              onClick={() => setOpen(false)}
              aria-label="Fechar copiloto"
              className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-elevated hover:text-on-dark"
            >
              <IconClose width={16} height={16} />
            </button>
          </div>
          <div className="flex flex-col gap-3 p-5">
            <p className="text-sm leading-relaxed text-body">
              O copiloto conversacional chega na{" "}
              <span className="text-primary-tint">SPEC-215</span>. O shell já reserva este espaço
              flutuante, disponível em todas as telas.
            </p>
            <NonAdviceDisclaimer />
          </div>
        </div>
      ) : (
        <button
          onClick={() => setOpen(true)}
          aria-label="Abrir copiloto"
          className="flex h-14 w-14 items-center justify-center rounded-full text-on-primary"
          style={{
            background: "linear-gradient(135deg, var(--primary), var(--primary-active))",
            boxShadow: "0 0 30px -4px rgba(233,169,76,.7)",
          }}
        >
          <IconCopilot width={24} height={24} />
        </button>
      )}
    </div>
  );
}
