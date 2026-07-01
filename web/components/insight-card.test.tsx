import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { InsightCard } from "@/components/insight-card";

// FR-013: the explanation is a required prop (TS-enforced) and always renders.
describe("InsightCard", () => {
  it("renders the headline and its required explanation", () => {
    render(
      <InsightCard category="Alocação" explanation="Logística é 62% dos seus FIIs.">
        Sua exposição a logística está alta.
      </InsightCard>,
    );
    expect(screen.getByText("Sua exposição a logística está alta.")).toBeInTheDocument();
    expect(screen.getByText("Logística é 62% dos seus FIIs.")).toBeInTheDocument();
    expect(screen.getByText("Por quê")).toBeInTheDocument();
  });

  it("shows the attention badge when flagged", () => {
    render(
      <InsightCard attention explanation="por quê">
        headline
      </InsightCard>,
    );
    expect(screen.getByText("Atenção")).toBeInTheDocument();
  });
});
