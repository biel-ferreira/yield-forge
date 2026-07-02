import type { components } from "@/lib/api/schema";

// Enum ↔ pt-BR label mapping (SPEC-210 FR-2106, BR-2104). The wire always carries the API's
// enum values; the UI always shows the pt-BR labels. Typed from the generated contract, so a
// new enum value fails the build HERE (a missing Record key) rather than silently.
type ProfileReq = components["schemas"]["ProfileRequest"];

export type RiskProfile = NonNullable<ProfileReq["risk_profile"]>;
export type Objective = NonNullable<ProfileReq["objectives"]>[number];

export const RISK_PROFILE_LABELS: Record<RiskProfile, string> = {
  conservative: "Conservador",
  moderate: "Moderado",
  aggressive: "Agressivo",
};

export const OBJECTIVE_LABELS: Record<Objective, string> = {
  retirement: "Aposentadoria",
  passive_income: "Renda passiva",
  wealth_preservation: "Preservação de patrimônio",
  long_term_growth: "Crescimento de longo prazo",
};

// Stable render order.
export const RISK_PROFILES = Object.keys(RISK_PROFILE_LABELS) as RiskProfile[];
export const OBJECTIVES = Object.keys(OBJECTIVE_LABELS) as Objective[];
