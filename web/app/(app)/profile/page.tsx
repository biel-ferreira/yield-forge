"use client";

import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Segmented } from "@/components/ui/segmented";
import { ChipToggleGroup } from "@/components/ui/chip-toggle";
import { Slider } from "@/components/ui/slider";
import { useProfile, useSaveProfile, type Profile } from "@/lib/profile/profile";
import {
  RISK_PROFILES,
  RISK_PROFILE_LABELS,
  OBJECTIVES,
  OBJECTIVE_LABELS,
  type RiskProfile,
  type Objective,
} from "@/lib/profile/labels";

export default function ProfilePage() {
  const { profile, isLoading, isError, refetch } = useProfile();

  if (isLoading) {
    return (
      <div className="mx-auto max-w-xl space-y-4">
        <div className="h-8 w-40 animate-pulse rounded-md bg-elevated" />
        <div className="h-80 animate-pulse rounded-xl bg-elevated" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="mx-auto flex max-w-xl flex-col items-center gap-4 py-16 text-center">
        <p className="text-sm text-loss">
          Não foi possível carregar seu perfil. Verifique sua conexão.
        </p>
        <Button variant="secondary" onClick={() => refetch()}>
          Tentar novamente
        </Button>
      </div>
    );
  }

  // `profile` is Profile | null (null = first run, from GET /profile 404). The form
  // initializes from it on mount (no effect); it mounts only after the load resolves.
  return <ProfileForm initial={profile} />;
}

function ProfileForm({ initial }: { initial: Profile | null }) {
  const save = useSaveProfile();
  const [risk, setRisk] = useState<RiskProfile | null>(initial?.risk_profile ?? null);
  const [objectives, setObjectives] = useState<Objective[]>(initial?.objectives ?? []);
  const [horizon, setHorizon] = useState<number>(initial?.horizon_years ?? 10);

  const firstRun = initial === null;
  const valid = risk !== null && objectives.length >= 1 && horizon >= 1 && horizon <= 50;

  // Any edit clears the previous save result (success/error message).
  function onRisk(value: RiskProfile) {
    setRisk(value);
    save.reset();
  }
  function onToggle(value: Objective) {
    setObjectives((cur) =>
      cur.includes(value) ? cur.filter((x) => x !== value) : [...cur, value],
    );
    save.reset();
  }
  function onHorizon(value: number) {
    setHorizon(value);
    save.reset();
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (risk === null || !valid) return;
    // The body is exactly the contract — no user_id (identity is the session, BR-2101).
    save.mutate({ risk_profile: risk, objectives, horizon_years: horizon });
  }

  return (
    <form onSubmit={onSubmit} className="mx-auto max-w-xl">
      <h1 className="font-serif text-2xl font-semibold text-on-dark">
        {firstRun ? "Defina seu perfil" : "Seu perfil"}
      </h1>
      <p className="mt-1 text-sm text-muted-strong">
        Seu perfil de risco, objetivos e horizonte personalizam os insights do copiloto.
      </p>

      <Card className="mt-6 space-y-7 p-6">
        <fieldset>
          <legend className="mb-2.5 text-xs font-semibold uppercase tracking-wide text-muted">
            Perfil de risco
          </legend>
          <Segmented
            ariaLabel="Perfil de risco"
            value={risk}
            onChange={onRisk}
            options={RISK_PROFILES.map((v) => ({ value: v, label: RISK_PROFILE_LABELS[v] }))}
          />
        </fieldset>

        <fieldset>
          <legend className="mb-2.5 text-xs font-semibold uppercase tracking-wide text-muted">
            Objetivos
          </legend>
          <ChipToggleGroup
            ariaLabel="Objetivos"
            selected={objectives}
            onToggle={onToggle}
            options={OBJECTIVES.map((v) => ({ value: v, label: OBJECTIVE_LABELS[v] }))}
          />
          {objectives.length === 0 && (
            <p className="mt-2 text-xs text-muted">Selecione ao menos um objetivo.</p>
          )}
        </fieldset>

        <fieldset>
          <legend className="mb-2.5 text-xs font-semibold uppercase tracking-wide text-muted">
            Horizonte de investimento
          </legend>
          <Slider
            ariaLabel="Horizonte de investimento (anos)"
            min={1}
            max={50}
            value={horizon}
            onChange={onHorizon}
            formatValue={(v) => `${v} ${v === 1 ? "ano" : "anos"}`}
          />
        </fieldset>
      </Card>

      <div className="mt-5 flex items-center gap-4">
        <Button type="submit" disabled={!valid || save.isPending}>
          {save.isPending ? "Salvando…" : "Salvar"}
        </Button>
        {save.isError && (
          <span className="text-xs text-loss">
            {save.error instanceof Error ? save.error.message : "Falha ao salvar."}
          </span>
        )}
        {save.isSuccess && <span className="text-xs text-gain">Perfil salvo ✓</span>}
      </div>
    </form>
  );
}
