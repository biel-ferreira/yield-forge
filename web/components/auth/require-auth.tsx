"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";
import { useSession } from "@/lib/auth/session";
import { Button } from "@/components/ui/button";

// Client-side gate for authenticated routes (D2 CSR default, SPEC-200 FR-2003).
// States: loading → neutral placeholder; a transient error (server unreachable) → retry,
// NOT a redirect (never eject an actually-authenticated user on a blip); a confirmed
// unauthenticated result (401 → null) → redirect to /login without rendering anything protected.
export function RequireAuth({ children }: { children: ReactNode }) {
  const { isLoading, isError, isAuthenticated, refetch } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !isError && !isAuthenticated) {
      router.replace("/login");
    }
  }, [isLoading, isError, isAuthenticated, router]);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-muted">
        Carregando…
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4 px-6 text-center">
        <p className="text-sm text-loss">
          Não foi possível verificar sua sessão. Verifique sua conexão e tente novamente.
        </p>
        <Button variant="secondary" onClick={() => refetch()}>
          Tentar novamente
        </Button>
      </div>
    );
  }

  if (!isAuthenticated) return null; // redirecting to /login

  return <>{children}</>;
}
