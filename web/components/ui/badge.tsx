import type { HTMLAttributes } from "react";
import { cn } from "@/lib/cn";

type BadgeVariant = "neutral" | "caution" | "gain" | "info";

const variants: Record<BadgeVariant, string> = {
  neutral: "bg-elevated text-body",
  caution: "border border-caution/25 bg-caution/10 text-caution",
  gain: "bg-gain/10 text-gain",
  info: "bg-info/10 text-info",
};

export function Badge({
  className,
  variant = "neutral",
  ...props
}: HTMLAttributes<HTMLSpanElement> & { variant?: BadgeVariant }) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold",
        variants[variant],
        className,
      )}
      {...props}
    />
  );
}
