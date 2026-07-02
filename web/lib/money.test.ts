import { describe, it, expect } from "vitest";
import { formatCentavos, formatBps, formatShareBps } from "@/lib/money";

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
