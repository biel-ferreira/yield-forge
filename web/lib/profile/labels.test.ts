import { describe, it, expect } from "vitest";
import {
  RISK_PROFILES,
  RISK_PROFILE_LABELS,
  OBJECTIVES,
  OBJECTIVE_LABELS,
} from "@/lib/profile/labels";

describe("profile labels (FR-2106)", () => {
  it("maps every risk profile to a non-empty pt-BR label", () => {
    expect(RISK_PROFILES).toEqual(["conservative", "moderate", "aggressive"]);
    for (const r of RISK_PROFILES) expect(RISK_PROFILE_LABELS[r]).toBeTruthy();
    expect(RISK_PROFILE_LABELS.moderate).toBe("Moderado");
  });

  it("maps every objective to a non-empty pt-BR label", () => {
    expect(OBJECTIVES).toEqual([
      "retirement",
      "passive_income",
      "wealth_preservation",
      "long_term_growth",
    ]);
    for (const o of OBJECTIVES) expect(OBJECTIVE_LABELS[o]).toBeTruthy();
    expect(OBJECTIVE_LABELS.passive_income).toBe("Renda passiva");
  });
});
