import type { ReactNode } from "react";
import { RequireAuth } from "@/components/auth/require-auth";

// The (app) route group is the authenticated area. Its layout gates every child route
// behind the session (SPEC-200 FR-2003). The real app shell (sidebar + top bar + copilot
// slot) is added here in Phase 5.
export default function AppLayout({ children }: { children: ReactNode }) {
  return <RequireAuth>{children}</RequireAuth>;
}
