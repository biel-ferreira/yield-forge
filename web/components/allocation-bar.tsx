import { cn } from "@/lib/cn";
import { formatShareBps } from "@/lib/money";

export interface AllocationSegment {
  label: string;
  /** Share in basis points (1% = 100 bps) — integer, money-as-integer convention (BR-2003). */
  bps: number;
  /** Segment color (token var or hex). */
  color: string;
}

/**
 * The signature Aurora spectrum bar, used as the allocation-by-sector view. Decorative
 * glow, but the data (sector shares) is real and legible. (SPEC-200 FR-2004)
 */
export function AllocationBar({
  segments,
  className,
}: {
  segments: AllocationSegment[];
  className?: string;
}) {
  return (
    <div className={cn("w-full", className)}>
      <div
        className="flex h-3.5 overflow-hidden rounded-full"
        style={{ boxShadow: "0 0 22px -2px rgba(99,102,241,.5)" }}
      >
        {segments.map((s) => (
          <div key={s.label} style={{ width: formatShareBps(s.bps), background: s.color }} />
        ))}
      </div>
      <div className="mt-3 flex flex-wrap gap-x-5 gap-y-2">
        {segments.map((s) => (
          <div key={s.label} className="flex items-center gap-2 text-[13px] text-body">
            <span
              className="h-2.5 w-2.5 flex-none rounded-full"
              style={{ background: s.color, boxShadow: `0 0 8px 1px ${s.color}` }}
            />
            <span>{s.label}</span>
            <span className="tabular font-mono text-muted">{formatShareBps(s.bps)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
