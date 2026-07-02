"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { NAV_ITEMS, isActive } from "@/lib/shell/nav";
import { cn } from "@/lib/cn";

// Bottom tab bar for mobile (< md); the desktop sidebar is hidden there. (SPEC-200 FR-2006)
export function MobileTabBar() {
  const pathname = usePathname();
  return (
    <nav className="fixed inset-x-0 bottom-0 z-40 flex border-t border-hairline bg-surface md:hidden">
      {NAV_ITEMS.map(({ href, label, Icon }) => {
        const active = isActive(pathname, href);
        return (
          <Link
            key={href}
            href={href}
            className={cn(
              "flex flex-1 flex-col items-center gap-1 py-2 text-[10px] font-medium",
              active ? "text-primary" : "text-muted",
            )}
          >
            <Icon width={20} height={20} />
            {label}
          </Link>
        );
      })}
    </nav>
  );
}
