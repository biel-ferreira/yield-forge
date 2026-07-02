import { cn } from "@/lib/cn";

export interface ChipOption<T extends string> {
  value: T;
  label: string;
}

// Multi-select toggle chips (SPEC-210 FR-2103). Each chip is an accessible checkbox; selected
// uses the gold-tinted `objective-chip-selected` treatment. Generic over the value union.
export function ChipToggleGroup<T extends string>({
  options,
  selected,
  onToggle,
  ariaLabel,
  className,
}: {
  options: ChipOption<T>[];
  selected: readonly T[];
  onToggle: (value: T) => void;
  ariaLabel?: string;
  className?: string;
}) {
  return (
    <div role="group" aria-label={ariaLabel} className={cn("flex flex-wrap gap-2.5", className)}>
      {options.map((opt) => {
        const on = selected.includes(opt.value);
        return (
          <button
            key={opt.value}
            type="button"
            role="checkbox"
            aria-checked={on}
            onClick={() => onToggle(opt.value)}
            className={cn(
              "rounded-full px-3.5 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-info",
              on
                ? "border border-primary/40 bg-primary/10 text-primary-tint"
                : "bg-elevated text-body hover:text-on-dark",
            )}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}
