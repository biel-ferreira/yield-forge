"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useLogout, useSession } from "@/lib/auth/session";

// Placeholder authenticated page proving the session gate works. The real dashboard +
// app shell replace this in Phase 5.
export default function DashboardStub() {
  const { user } = useSession();
  const logout = useLogout();
  const router = useRouter();

  return (
    <main className="mx-auto max-w-2xl px-6 py-16">
      <h1 className="font-serif text-3xl font-semibold text-on-dark">Autenticado ✓</h1>
      <p className="mt-2 text-body">
        Olá, <span className="text-primary-tint">{user?.email}</span>. O shell do app (sidebar,
        top bar, copiloto flutuante) chega na Phase 5.
      </p>
      <Button
        className="mt-6"
        variant="secondary"
        disabled={logout.isPending}
        onClick={() => logout.mutate(undefined, { onSuccess: () => router.replace("/login") })}
      >
        {logout.isPending ? "Saindo…" : "Sair"}
      </Button>
    </main>
  );
}
