"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useLogin, useRegister } from "@/lib/auth/session";

export default function RegisterPage() {
  const router = useRouter();
  const register = useRegister();
  const login = useLogin();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const pending = register.isPending || login.isPending;
  const error = register.error ?? login.error;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      // Register creates the account (no session), then log in to start one.
      await register.mutateAsync({ email, password });
      await login.mutateAsync({ email, password });
      router.replace("/dashboard");
    } catch {
      // surfaced via mutation error state below
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-6">
      <Card className="w-full max-w-sm p-8">
        <h1 className="font-serif text-3xl font-semibold text-on-dark">Criar conta</h1>
        <p className="mt-1 text-sm text-muted-strong">Comece a acompanhar sua carteira.</p>

        <form onSubmit={onSubmit} className="mt-6 space-y-4">
          <div>
            <label htmlFor="email" className="mb-1.5 block text-xs font-medium text-muted">
              E-mail
            </label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="voce@exemplo.com"
            />
          </div>
          <div>
            <label htmlFor="password" className="mb-1.5 block text-xs font-medium text-muted">
              Senha (mín. 8 caracteres)
            </label>
            <Input
              id="password"
              type="password"
              autoComplete="new-password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
            />
          </div>

          {error && (
            <p className="text-xs text-loss">
              {error instanceof Error ? error.message : "Falha ao criar a conta."}
            </p>
          )}

          <Button type="submit" className="w-full" disabled={pending}>
            {pending ? "Criando…" : "Criar conta"}
          </Button>
        </form>

        <p className="mt-4 text-center text-xs text-muted">
          Já tem conta?{" "}
          <Link href="/login" className="text-primary-tint">
            Entrar
          </Link>
        </p>
      </Card>
    </main>
  );
}
