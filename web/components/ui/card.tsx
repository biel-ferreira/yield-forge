import type { HTMLAttributes } from "react";
import { cn } from "@/lib/cn";

// Glass surface with a subtle sheen + colored ambient glow (dropped on light). (SPEC-200 FR-2004)
export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("glass rounded-xl", className)} {...props} />;
}
