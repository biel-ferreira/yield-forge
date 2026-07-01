import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

export interface InsightCardProps {
  /** Short insight headline. */
  children: ReactNode;
  /**
   * The human-readable "por quê". REQUIRED by contract — an insight without its
   * explanation is unrepresentable (FR-013, explainability), mirroring the backend
   * `Gated` Insighter. Parse-don't-validate, applied to component props.
   */
  explanation: ReactNode;
  category?: string;
  attention?: boolean;
  className?: string;
}

export function InsightCard({
  children,
  explanation,
  category,
  attention,
  className,
}: InsightCardProps) {
  return (
    <article className={cn("glass relative overflow-hidden rounded-lg p-5 pl-6", className)}>
      {/* glowing gradient left edge */}
      <span
        aria-hidden
        className="absolute inset-y-0 left-0 w-[3px]"
        style={{
          background: "linear-gradient(180deg, var(--primary), var(--aurora-2))",
          boxShadow: "0 0 16px 1px rgba(233,169,76,.6)",
        }}
      />
      {attention && (
        <span className="mb-2 inline-flex items-center rounded-full border border-caution/25 bg-caution/10 px-2.5 py-1 text-xs font-semibold text-caution">
          Atenção
        </span>
      )}
      {category && (
        <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted">
          {category}
        </div>
      )}
      <div className="text-[15px] font-semibold leading-snug text-on-dark">{children}</div>
      <div className="mt-3 rounded-md border border-hairline bg-elevated p-3">
        <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-primary-tint">
          Por quê
        </div>
        <div className="text-[13px] leading-relaxed text-muted-strong">{explanation}</div>
      </div>
    </article>
  );
}
