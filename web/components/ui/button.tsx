import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

// Aurora buttons (SPEC-200 FR-2004). Note: there is deliberately NO buy/sell/order
// variant — the copilot never issues a transaction order (FR-014).
type Variant = "primary" | "outline" | "secondary" | "ghost" | "link";
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
