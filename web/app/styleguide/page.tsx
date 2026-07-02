"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { InsightCard } from "@/components/insight-card";
import { NonAdviceDisclaimer } from "@/components/non-advice-disclaimer";
import { AllocationBar } from "@/components/allocation-bar";
import { Segmented } from "@/components/ui/segmented";
import { ChipToggleGroup } from "@/components/ui/chip-toggle";
import { Slider } from "@/components/ui/slider";
import {
  RISK_PROFILES,
  RISK_PROFILE_LABELS,
  OBJECTIVES,
  OBJECTIVE_LABELS,
  type RiskProfile,
  type Objective,
} from "@/lib/profile/labels";

// Public dev styleguide at /styleguide — a living gallery of the Aurora tokens + components.
// Not part of the authenticated app; kept for reference.
export default function StyleguidePage() {
  const [light, setLight] = useState(false);
  const [risk, setRisk] = useState<RiskProfile | null>("moderate");
  const [objectives, setObjectives] = useState<Objective[]>(["passive_income"]);
  const [horizon, setHorizon] = useState(10);

  function toggleObjective(o: Objective) {
    setObjectives((cur) => (cur.includes(o) ? cur.filter((x) => x !== o) : [...cur, o]));
  }

  function toggleTheme() {
    const next = !light;
    setLight(next);
    const el = document.documentElement;
    if (next) el.setAttribute("data-theme", "light");
    else el.removeAttribute("data-theme");
  }

  return (
    <main className="mx-auto max-w-4xl px-6 py-12">
      <header className="mb-10 flex items-end justify-between">
        <div>
          <h1 className="font-serif text-4xl font-semibold text-on-dark">YieldForge</h1>
          <p className="mt-1 text-sm text-muted-strong">Aurora design system · styleguide</p>
        </div>
        <Button variant="outline" size="sm" onClick={toggleTheme}>
          Tema: {light ? "Claro" : "Escuro"}
        </Button>
      </header>

      <Section title="Controles de formulário (SPEC-210)">
        <Card className="space-y-6 p-6">
          <div>
            <label className="mb-2 block text-xs font-medium text-muted">Perfil de risco</label>
            <Segmented
              ariaLabel="Perfil de risco"
              value={risk}
              onChange={setRisk}
              options={RISK_PROFILES.map((v) => ({ value: v, label: RISK_PROFILE_LABELS[v] }))}
            />
          </div>
          <div>
            <label className="mb-2 block text-xs font-medium text-muted">Objetivos</label>
            <ChipToggleGroup
              ariaLabel="Objetivos"
              selected={objectives}
              onToggle={toggleObjective}
              options={OBJECTIVES.map((v) => ({ value: v, label: OBJECTIVE_LABELS[v] }))}
            />
          </div>
          <div>
            <label className="mb-2 block text-xs font-medium text-muted">Horizonte</label>
            <Slider
              ariaLabel="Horizonte de investimento (anos)"
              min={1}
              max={50}
              value={horizon}
              onChange={setHorizon}
              formatValue={(v) => `${v} ${v === 1 ? "ano" : "anos"}`}
              className="max-w-sm"
            />
          </div>
        </Card>
      </Section>

      <Section title="Tipografia & números">
        <Card className="p-6">
          <p className="font-serif text-3xl font-semibold text-on-dark">
            Seu patrimônio, iluminado
          </p>
          <p className="mt-3 max-w-prose text-[15px] leading-relaxed text-body">
            Sua carteira está concentrada em FIIs de logística. Considere avaliar outros segmentos —{" "}
            <span className="text-primary-tint">saiba o porquê</span>.
          </p>
          <p className="tabular mt-4 font-mono text-4xl font-semibold text-on-dark">
            R$ 297.924,80
          </p>
          <p className="tabular mt-1 font-mono text-sm text-gain">▲ +5,3% no mês</p>
        </Card>
      </Section>

      <Section title="Botões (sem ordens de compra/venda — FR-014)">
        <div className="flex flex-wrap items-center gap-3">
          <Button>Salvar carteira</Button>
          <Button variant="outline">Solicitar acesso</Button>
          <Button variant="secondary">Cancelar</Button>
          <Button variant="ghost">Voltar</Button>
          <Button variant="link">ver detalhes →</Button>
          <Button disabled>Desabilitado</Button>
        </div>
      </Section>

      <Section title="Badges & input">
        <div className="flex flex-wrap items-center gap-3">
          <Badge>Neutro</Badge>
          <Badge variant="caution">Atenção</Badge>
          <Badge variant="gain">+2,4%</Badge>
          <Badge variant="info">Info</Badge>
        </div>
        <Input className="mt-4 max-w-sm" placeholder="Pergunte ao copiloto…" />
      </Section>

      <Section title="Alocação por setor (spectrum bar)">
        <Card className="p-6">
          <AllocationBar
            segments={[
              { label: "Logística", bps: 2600, color: "var(--aurora-1)" },
              { label: "Shoppings", bps: 2400, color: "var(--aurora-3)" },
              { label: "Papel (CRI)", bps: 2000, color: "var(--aurora-2)" },
              { label: "Lajes corp.", bps: 1600, color: "var(--aurora-4)" },
              { label: "Outros", bps: 1400, color: "var(--aurora-5)" },
            ]}
          />
        </Card>
      </Section>

      <Section title="Superfícies de IA (com as travas)">
        <div className="space-y-4">
          <InsightCard
            category="Alocação"
            explanation="Logística representa 62% da sua carteira de FIIs. Uma concentração acima de ~40% num único segmento aumenta a sensibilidade a um choque setorial."
          >
            Sua exposição a FIIs de logística está acima da faixa típica de diversificação.
          </InsightCard>
          <InsightCard
            attention
            category="Contexto de mercado"
            explanation="Com a SELIC em 10,50% a.a., títulos pós-fixados oferecem retorno real relevante — uma consideração para o seu objetivo de renda passiva, não uma ordem de compra."
          >
            A SELIC atual favorece a parcela de renda fixa da sua estratégia de aportes.
          </InsightCard>
          <NonAdviceDisclaimer />
        </div>
      </Section>
    </main>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="mb-10">
      <h2 className="mb-4 text-xs font-semibold uppercase tracking-wide text-muted">{title}</h2>
      {children}
    </section>
  );
}
