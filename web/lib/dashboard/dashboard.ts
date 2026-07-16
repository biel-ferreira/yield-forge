import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

// The computed portfolio dashboard (SPEC-212) — the frontend face of SPEC-103. Every figure is
// read verbatim from the backend; this screen never sums, subtracts, or otherwise recomputes a
// money/percentage value client-side (BR-2121).
export type Dashboard = components["schemas"]["DashboardResponse"];

// Exported so lib/portfolio/holdings.ts's mutations can invalidate it (SPEC-212 review finding):
// staleTime is 30s globally (app/providers.tsx), so without this, a Carteira→Painel SPA
// navigation within that window would silently serve pre-mutation figures — undermining
// BR-2121's "every figure is authoritative" premise for exactly as long as the cache lives.
export const DASHBOARD_KEY = ["dashboard"] as const;

/** The caller's computed dashboard (GET /dashboard). */
export function useDashboard() {
  const query = useQuery({
    queryKey: DASHBOARD_KEY,
    queryFn: async () => {
      const { data } = await api.GET("/dashboard");
      if (!data) throw new Error("failed to load dashboard");
      return data;
    },
  });
  return {
    dashboard: query.data ?? null,
    isLoading: query.isLoading,
    isError: query.isError,
    refetch: query.refetch,
  };
}
