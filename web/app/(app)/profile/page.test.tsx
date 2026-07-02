import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ProfilePage from "@/app/(app)/profile/page";
import * as profileHooks from "@/lib/profile/profile";
import type { Profile } from "@/lib/profile/profile";

vi.mock("@/lib/profile/profile", () => ({
  useProfile: vi.fn(),
  useSaveProfile: vi.fn(),
}));

const useProfile = vi.mocked(profileHooks.useProfile);
const useSaveProfile = vi.mocked(profileHooks.useSaveProfile);

type ProfileState = ReturnType<typeof profileHooks.useProfile>;
type SaveHook = ReturnType<typeof profileHooks.useSaveProfile>;

function profileState(profile: Profile | null): ProfileState {
  return { profile, isLoading: false, isError: false, refetch: vi.fn() } as unknown as ProfileState;
}
function saveHook(over: Partial<SaveHook> = {}): SaveHook {
  return {
    mutate: vi.fn(),
    reset: vi.fn(),
    isPending: false,
    isError: false,
    isSuccess: false,
    error: null,
    ...over,
  } as unknown as SaveHook;
}

beforeEach(() => {
  vi.clearAllMocks();
  useSaveProfile.mockReturnValue(saveHook());
});

describe("ProfilePage (SPEC-210)", () => {
  it("first run (404 → null): first-run heading + disabled Salvar", () => {
    useProfile.mockReturnValue(profileState(null));
    render(<ProfilePage />);
    expect(screen.getByRole("heading", { name: "Defina seu perfil" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  });

  it("existing profile: prefills and enables Salvar", () => {
    useProfile.mockReturnValue(
      profileState({
        risk_profile: "aggressive",
        objectives: ["retirement"],
        horizon_years: 20,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      }),
    );
    render(<ProfilePage />);
    expect(screen.getByRole("heading", { name: "Seu perfil" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Agressivo" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Aposentadoria" })).toBeChecked();
    expect(screen.getByRole("button", { name: "Salvar" })).toBeEnabled();
  });

  it("gates Salvar on validation (≥1 objective required)", async () => {
    const user = userEvent.setup();
    useProfile.mockReturnValue(profileState(null));
    render(<ProfilePage />);
    const salvar = screen.getByRole("button", { name: "Salvar" });
    expect(salvar).toBeDisabled();
    await user.click(screen.getByRole("radio", { name: "Moderado" }));
    expect(salvar).toBeDisabled(); // risk set, still no objective
    await user.click(screen.getByRole("checkbox", { name: "Renda passiva" }));
    expect(salvar).toBeEnabled();
    await user.click(screen.getByRole("checkbox", { name: "Renda passiva" })); // deselect
    expect(salvar).toBeDisabled();
  });

  it("submits the exact ProfileRequest (no user_id)", async () => {
    const user = userEvent.setup();
    const save = saveHook();
    useSaveProfile.mockReturnValue(save);
    useProfile.mockReturnValue(profileState(null));
    render(<ProfilePage />);
    await user.click(screen.getByRole("radio", { name: "Conservador" }));
    await user.click(screen.getByRole("checkbox", { name: "Aposentadoria" }));
    await user.click(screen.getByRole("button", { name: "Salvar" }));
    expect(save.mutate).toHaveBeenCalledTimes(1);
    expect(save.mutate).toHaveBeenCalledWith({
      risk_profile: "conservative",
      objectives: ["retirement"],
      horizon_years: 10,
    });
  });
});
