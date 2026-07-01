"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";
import { useSession } from "@/lib/auth/session";

// Client-side gate for authenticated routes (D2 CSR default, SPEC-200 FR-2003).
// While the session resolves it shows a neutral loading state — never protected content;
// if unauthenticated it redirects to /login without rendering anything protected.
export function RequireAuth({ children }: { children: ReactNode }) {
  const { isLoading, isAuthenticated } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace("/login");
    }
  }, [isLoading, isAuthenticated, router]);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-muted">
        Carregando…
      </div>
    );
  }

  if (!isAuthenticated) return null; // redirecting to /login

  return <>{children}</>;
}
