import { describe, it, expect } from "vitest";
import {
  formatCentavos,
  formatBps,
  formatShareBps,
  parseCentavos,
  parseBps,
  centavosToInputString,
  bpsToInputString,
} from "@/lib/money";

// The money/rate render edge (BR-2003 / FR-2005): integer in, exact pt-BR string out.
describe("formatCentavos", () => {
  it.each([
    [1234567, "R$ 12.345,67"],
    [-120449, "-R$ 1.204,49"],
    [0, "R$ 0,00"],
    [5, "R$ 0,05"],
    [100, "R$ 1,00"],
  ])("%d → %s", (centavos, expected) => {
    expect(formatCentavos(centavos)).toBe(expected);
  });
});

describe("formatBps", () => {
  it.each([
    [1050, "10,50%"],
    [82, "0,82%"],
    [0, "0,00%"],
    [-250, "-2,50%"],
  ])("%d → %s", (bps, expected) => {
    expect(formatBps(bps)).toBe(expected);
  });
});

describe("formatShareBps (compact percent for allocation)", () => {
  it.each([
    [2600, "26%"],
    [1650, "16,5%"],
    [82, "0,82%"],
    [10000, "100%"],
  ])("%d → %s", (bps, expected) => {
    expect(formatShareBps(bps)).toBe(expected);
  });
});

// The input-side counterpart (SPEC-211 FR-2119, BR-2112): pt-BR string in, integer out —
// never a float, never silently coerced to 0 on bad input.
describe("parseCentavos", () => {
  it.each([
    ["1.234,56", 123456],
    ["10,5", 1050],
    ["0", 0],
    ["1234", 123400],
    ["-120,44", -12044],
    ["  157,50  ", 15750], // surrounding whitespace trimmed
  ])("%s → %d", (input, expected) => {
    expect(parseCentavos(input)).toBe(expected);
  });

  it.each([
    ["", "empty"],
    ["abc", "non-numeric"],
    ["12.34,56", "malformed thousands grouping"],
    ["1,2,3", "multiple commas"],
  ])("%s (%s) → null, never coerced to 0", (input) => {
    expect(parseCentavos(input)).toBeNull();
  });

  it("round-trips with formatCentavos for a representative value", () => {
    expect(parseCentavos(formatCentavos(123456).replace("R$ ", ""))).toBe(123456);
  });
});

describe("parseBps", () => {
  it.each([
    ["10,5", 1050],
    ["120", 12000],
    ["5,80", 580],
    ["0", 0],
  ])("%s → %d", (input, expected) => {
    expect(parseBps(input)).toBe(expected);
  });

  it.each([
    ["", "empty"],
    ["dez", "non-numeric"],
  ])("%s (%s) → null", (input) => {
    expect(parseBps(input)).toBeNull();
  });

  it("round-trips with formatBps for a representative value", () => {
    expect(parseBps(formatBps(1050).replace("%", ""))).toBe(1050);
  });
});

// The edit-prefill counterparts (SPEC-211 FR-2117): pure integer math only — never `/100` +
// `toFixed` (float arithmetic on a monetary value, flagged in review) — round-tripping exactly
// through parseCentavos/parseBps.
describe("centavosToInputString", () => {
  it.each([
    [15750, "157,50"],
    [1_000_000, "10.000,00"],
    [0, "0,00"],
    [5, "0,05"],
    [-12044, "-120,44"],
  ])("%d → %s", (centavos, expected) => {
    expect(centavosToInputString(centavos)).toBe(expected);
  });

  it("round-trips exactly through parseCentavos", () => {
    for (const v of [15750, 1_000_000, 0, 5, 123456]) {
      expect(parseCentavos(centavosToInputString(v))).toBe(v);
    }
  });
});

describe("bpsToInputString", () => {
  it.each([
    [12_000, "120,00"],
    [580, "5,80"],
    [1000, "10,00"],
    [0, "0,00"],
  ])("%d → %s", (bps, expected) => {
    expect(bpsToInputString(bps)).toBe(expected);
  });

  it("round-trips exactly through parseBps", () => {
    for (const v of [12_000, 580, 1000, 0]) {
      expect(parseBps(bpsToInputString(v))).toBe(v);
    }
  });
});
