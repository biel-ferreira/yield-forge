import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { NonAdviceDisclaimer } from "@/components/non-advice-disclaimer";

// FR-014: the non-advice disclaimer is required on AI surfaces.
describe("NonAdviceDisclaimer", () => {
  it("renders the default non-advice text", () => {
    render(<NonAdviceDisclaimer />);
    expect(screen.getByText(/não recomendação de investimento/i)).toBeInTheDocument();
  });

  it("renders the extended variant (explicit no-orders language)", () => {
    render(<NonAdviceDisclaimer extended />);
    expect(screen.getByText(/não emite ordens de compra ou venda/i)).toBeInTheDocument();
  });
});
