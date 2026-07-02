import { cn } from "@/lib/cn";

// Accessible range slider (SPEC-210 FR-2104, D5): a native <input type="range"> — keyboard-
// accessible by construction (arrow keys), bounded to [min, max], gold accent, with the value
// shown numerically. Zero runtime dependency (ADR-0003).
export function Slider({
  value,
  min,
  max,
  onChange,
  ariaLabel,
  className,
  formatValue = (v) => String(v),
}: {
  value: number;
  min: number;
  max: number;
  onChange: (value: number) => void;
  ariaLabel?: string;
  className?: string;
  formatValue?: (value: number) => string;
}) {
  return (
    <div className={cn("flex items-center gap-4", className)}>
      <input
        type="range"
        min={min}
        max={max}
        step={1}
        value={value}
        aria-label={ariaLabel}
        onChange={(e) => onChange(Number(e.target.value))}
        style={{ accentColor: "var(--primary)" }}
        className="h-1.5 flex-1 cursor-pointer appearance-none rounded-full bg-elevated focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-info"
      />
      <span className="tabular w-20 text-right font-mono text-sm text-on-dark">
        {formatValue(value)}
      </span>
    </div>
  );
}
