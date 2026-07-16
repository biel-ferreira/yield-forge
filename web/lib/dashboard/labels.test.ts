import { describe, it, expect } from "vitest";
import { SECTOR_LABELS, sectorLabel } from "@/lib/dashboard/labels";

describe("sector labels (SPEC-212 FR-2124)", () => {
  it("maps every known backend sector to a non-empty pt-BR label", () => {
    for (const key of ["logistics", "offices", "shopping", "hybrid", "paper", "other"]) {
      expect(SECTOR_LABELS[key]).toBeTruthy();
    }
    expect(SECTOR_LABELS.logistics).toBe("Logística");
  });

  it("labels 'other' distinctly as a data-quality signal, not a generic sector name", () => {
    expect(SECTOR_LABELS.other).toBe("Outros / sem cotação");
  });

  it("sectorLabel returns the mapped pt-BR label for a known sector", () => {
    expect(sectorLabel("shopping")).toBe("Shoppings");
  });

  it("sectorLabel falls back to a capitalized raw value for an unmapped sector (D6)", () => {
    // The wire's `sector` field is plain `string`, not a closed enum — a future/unknown backend
    // value must degrade gracefully, not render blank or throw.
    expect(sectorLabel("greenfield")).toBe("Greenfield");
  });

  it("sectorLabel handles an empty string without throwing", () => {
    expect(sectorLabel("")).toBe("");
  });
});
