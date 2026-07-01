import type { InputHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn(
        "h-11 w-full rounded-md border border-hairline bg-surface px-3.5 text-sm text-on-dark placeholder:text-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-info",
        className,
      )}
      {...props}
    />
  );
}
