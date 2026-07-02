import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AllocationBar } from "@/components/allocation-bar";

// helper: the first inline-styled element that carries a width (the spectrum segment)
function segmentWidth(container: HTMLElement): string | undefined {
  return Array.from(container.querySelectorAll<HTMLElement>("[style]")).find((el) => el.style.width)
    ?.style.width;
}

describe("AllocationBar", () => {
  // Regression: a fractional bps must produce a DOT-decimal CSS width. A pt-BR
  // comma value ("16,5%") is invalid CSS and collapses the segment to zero width.
  it("uses a dot-decimal CSS width for fractional shares", () => {
    const { container } = render(
      <AllocationBar segments={[{ label: "Logística", bps: 1650, color: "var(--aurora-1)" }]} />,
    );
    expect(segmentWidth(container)).toBe("16.5%");
  });

  it("uses a plain % for whole shares", () => {
    const { container } = render(
      <AllocationBar segments={[{ label: "Logística", bps: 2600, color: "var(--aurora-1)" }]} />,
    );
    expect(segmentWidth(container)).toBe("26%");
  });

  it("shows the pt-BR percent (comma) in the legend text", () => {
    render(
      <AllocationBar segments={[{ label: "Logística", bps: 1650, color: "var(--aurora-1)" }]} />,
    );
    expect(screen.getByText("16,5%")).toBeInTheDocument();
    expect(screen.getByText("Logística")).toBeInTheDocument();
  });
});
