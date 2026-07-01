"use client";

import { Button } from "@/components/ui/button";

// Route-level error boundary for the authenticated area. Surfaces the message (the API's
// `{"error":"..."}` envelope bubbles up as an Error) with a retry. (SPEC-200 FR-2006)
export default function AppError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="glass flex flex-col items-center rounded-xl px-6 py-16 text-center">
      <h2 className="font-serif text-xl font-semibold text-on-dark">Algo deu errado</h2>
      <p className="mt-2 max-w-md text-sm text-loss">
        {error.message || "Ocorreu um erro inesperado."}
      </p>
      <Button className="mt-5" variant="secondary" onClick={reset}>
        Tentar novamente
      </Button>
    </div>
  );
}
