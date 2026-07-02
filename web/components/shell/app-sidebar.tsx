"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { NAV_ITEMS, isActive } from "@/lib/shell/nav";
import { useLogout, useSession } from "@/lib/auth/session";
import { cn } from "@/lib/cn";

// Desktop sidebar (md+). On mobile the nav collapses to MobileTabBar. (SPEC-200 FR-2006)
export function AppSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user } = useSession();
  const logout = useLogout();

  return (
    <aside className="sticky top-0 hidden h-screen flex-col border-r border-hairline bg-surface p-3.5 md:flex">
      <div className="flex items-center gap-2.5 px-2 py-3">
        <span
          className="h-7 w-7 rounded-lg"
          style={{
            background: "linear-gradient(135deg, var(--primary), var(--aurora-2))",
            boxShadow: "0 0 20px -4px rgba(233,169,76,.6)",
          }}
        />
        <span className="font-serif text-lg font-semibold text-on-dark">YieldForge</span>
      </div>

      <nav className="mt-2 flex flex-col gap-0.5">
        {NAV_ITEMS.map(({ href, label, Icon }) => {
          const active = isActive(pathname, href);
          return (
            <Link
              key={href}
              href={href}
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2.5 text-sm font-medium transition-colors",
                active
                  ? "bg-elevated text-on-dark"
                  : "text-muted-strong hover:bg-elevated hover:text-body",
              )}
              style={active ? { boxShadow: "inset 2px 0 0 var(--primary)" } : undefined}
            >
              <Icon className={active ? "text-primary" : undefined} />
              {label}
            </Link>
          );
        })}
      </nav>

      <div className="mt-auto border-t border-hairline px-1 py-3">
        <div className="flex items-center gap-2.5 px-1">
          <span
            className="h-8 w-8 flex-none rounded-full"
            style={{ background: "linear-gradient(135deg, var(--aurora-1), var(--aurora-2))" }}
          />
          <div className="min-w-0">
            <div className="truncate text-[13px] font-semibold text-on-dark">
              {user?.email ?? "—"}
            </div>
            <div className="text-xs text-muted">Plano gratuito</div>
          </div>
        </div>
        <button
          onClick={() => logout.mutate(undefined, { onSuccess: () => router.replace("/login") })}
          disabled={logout.isPending}
          className="mt-2 w-full rounded-md px-2 py-1.5 text-left text-xs text-muted transition-colors hover:bg-elevated hover:text-loss"
        >
          {logout.isPending ? "Saindo…" : "Sair"}
        </button>
      </div>
    </aside>
  );
}
