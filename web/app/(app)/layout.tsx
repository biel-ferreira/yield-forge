import type { ReactNode } from "react";
import { RequireAuth } from "@/components/auth/require-auth";
import { AppSidebar } from "@/components/shell/app-sidebar";
import { MobileTabBar } from "@/components/shell/mobile-tabbar";
import { TopBar } from "@/components/shell/top-bar";
import { CopilotLauncher } from "@/components/shell/copilot-launcher";

// The authenticated app shell (SPEC-200 FR-2006): sidebar (md+) / bottom tabs (mobile),
// top bar, content area, and the GLOBAL floating copilot launcher on every screen.
// Everything here is gated behind the session by RequireAuth.
export default function AppLayout({ children }: { children: ReactNode }) {
  return (
    <RequireAuth>
      <div className="md:grid md:grid-cols-[248px_1fr]">
        <AppSidebar />
        <div className="flex min-h-screen flex-col">
          <TopBar />
          <main className="mx-auto w-full max-w-[1200px] flex-1 px-5 py-6 pb-24 md:pb-8">
            {children}
          </main>
        </div>
      </div>
      <MobileTabBar />
      <CopilotLauncher />
    </RequireAuth>
  );
}
