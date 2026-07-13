import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

// Aurora buttons (SPEC-200 FR-2004). Note: there is deliberately NO buy/sell/order
// variant — the copilot never issues a transaction order (FR-014).
type Variant = "primary" | "outline" | "secondary" | "ghost" | "link" | "destructive";
type Size = "sm" | "md";

const base =
  "inline-flex items-center justify-center gap-2 rounded-md font-semibold text-sm leading-none transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-info disabled:pointer-events-none disabled:opacity-60";

const variants: Record<Variant, string> = {
  // gold fill + soft glow
  primary: "bg-primary text-on-primary glow-gold hover:bg-primary-active",
  // preferred lighter treatment: gold glowing outline
  outline:
    "border border-primary/50 bg-primary/5 text-primary-tint glow-gold-soft hover:bg-primary/10",
  secondary: "border border-hairline bg-elevated text-on-dark hover:bg-hairline/50",
  ghost: "text-muted-strong hover:bg-elevated hover:text-on-dark",
  link: "text-primary-tint underline-offset-4 hover:underline",
  // SPEC-211: an irreversible action (e.g. confirm-delete). Mirrors Badge's soft-tint pattern
  // for the `loss` semantic token (never a solid fill — CLAUDE.md reserves gain/loss/caution/info
  // as figure colors) rather than inventing a separate saturated "danger" brand color.
  destructive: "border border-loss/50 bg-loss/5 text-loss hover:bg-loss/10",
};

const sizes: Record<Size, string> = {
  sm: "h-9 px-3.5",
  md: "h-11 px-5",
};

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
}

export function Button({ className, variant = "primary", size = "md", ...props }: ButtonProps) {
  return <button className={cn(base, variants[variant], sizes[size], className)} {...props} />;
}
