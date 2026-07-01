import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

export function EmptyState({
  title,
  description,
  className,
  children,
}: {
  title: string;
  description?: string;
  className?: string;
  children?: ReactNode;
}) {
  return (
    <div
      className={cn(
        "glass flex flex-col items-center rounded-xl px-6 py-16 text-center",
        className,
      )}
    >
      <h2 className="font-serif text-xl font-semibold text-on-dark">{title}</h2>
      {description && <p className="mt-2 max-w-md text-sm text-muted-strong">{description}</p>}
      {children && <div className="mt-5">{children}</div>}
    </div>
  );
}
