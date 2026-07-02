"use client";

import { usePathname } from "next/navigation";
import { NAV_ITEMS, isActive } from "@/lib/shell/nav";
import { useSession } from "@/lib/auth/session";
import { Button } from "@/components/ui/button";
import { ThemeToggle } from "./theme-toggle";

// Top bar: current-section title + greeting, theme toggle, one primary action. (SPEC-200 FR-2006)
export function TopBar() {
  const pathname = usePathname();
  const { user } = useSession();
  const current = NAV_ITEMS.find((n) => isActive(pathname, n.href));
  const name = user?.email?.split("@")[0] ?? "";

  return (
    <header className="sticky top-0 z-30 flex items-center justify-between gap-4 border-b border-hairline bg-canvas/70 px-5 py-3 backdrop-blur">
      <div>
        <h1 className="font-serif text-xl font-semibold text-on-dark">
          {current?.label ?? "YieldForge"}
        </h1>
        {name && <p className="text-xs text-muted">Olá, {name}</p>}
      </div>
      <div className="flex items-center gap-2.5">
        <ThemeToggle />
        <Button variant="outline" size="sm">
          + Adicionar ativo
        </Button>
      </div>
    </header>
  );
}
