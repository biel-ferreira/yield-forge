import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import { backendError } from "@/lib/api/error";
import type { components } from "@/lib/api/schema";

// The investor profile (SPEC-210) — the frontend face of SPEC-101. Types come from the
// generated contract; no hand-written DTOs (BR-2104).
export type Profile = components["schemas"]["ProfileResponse"];
export type ProfileInput = components["schemas"]["ProfileRequest"];

const PROFILE_KEY = ["profile"] as const;

/** The saved profile, or `null` when unset (GET /profile → 404 = first run). */
export function useProfile() {
  const query = useQuery({
    queryKey: PROFILE_KEY,
    queryFn: async () => {
      const { data, response } = await api.GET("/profile");
      if (response.status === 404) return null; // no profile yet — first run
      if (!data) throw new Error("failed to load profile");
      return data;
    },
    retry: false,
  });

  return {
    profile: query.data ?? null,
    isLoading: query.isLoading,
    // isError = couldn't load (network/5xx) — distinct from the 404 first-run, which is `null`.
    isError: query.isError,
    refetch: query.refetch,
  };
}

/** Create-or-update the profile (PUT /profile). Identity from the session — no `user_id` (BR-2101). */
export function useSaveProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ProfileInput) => {
      const { data, error, response } = await api.PUT("/profile", { body: input });
      if (error || !data) {
        throw new Error(backendError(error) ?? `save failed (${response.status})`);
      }
      return data;
    },
    onSuccess: (saved) => qc.setQueryData(PROFILE_KEY, saved),
  });
}
