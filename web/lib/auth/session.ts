import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

// Session against SPEC-003. Identity is resolved by the server via /auth/me — never
// trusted from client state (BR-2001). (SPEC-200 FR-2003)
export type User = components["schemas"]["User"];
export type Credentials = components["schemas"]["Credentials"];

const ME_KEY = ["auth", "me"] as const;

/** Pull the `{ error }` message out of a backend error body, if present. */
function backendError(error: unknown): string | undefined {
  if (error && typeof error === "object" && "error" in error) {
    const message = (error as Record<string, unknown>).error;
    if (typeof message === "string") return message;
  }
  return undefined;
}

/** The authenticated user (or null). The server (/auth/me) is the authority. */
export function useSession() {
  const query = useQuery({
    queryKey: ME_KEY,
    queryFn: async () => {
      const { data, response } = await api.GET("/auth/me");
      if (response.status === 401) return null; // not authenticated
      if (!data) throw new Error("failed to load session");
      return data;
    },
    retry: false,
    staleTime: 60_000,
  });

  return {
    user: query.data ?? null,
    isLoading: query.isLoading,
    // isError = the server couldn't be reached / non-401 failure (distinct from a
    // confirmed-unauthenticated 401, which resolves to `data: null`).
    isError: query.isError,
    isAuthenticated: !!query.data,
    refetch: query.refetch,
  };
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (creds: Credentials) => {
      const { data, error, response } = await api.POST("/auth/login", { body: creds });
      if (error || !data) {
        throw new Error(backendError(error) ?? `login failed (${response.status})`);
      }
      return data;
    },
    onSuccess: (user) => qc.setQueryData(ME_KEY, user),
  });
}

export function useRegister() {
  return useMutation({
    mutationFn: async (creds: Credentials) => {
      const { data, error, response } = await api.POST("/auth/register", { body: creds });
      if (error || !data) {
        throw new Error(backendError(error) ?? `register failed (${response.status})`);
      }
      // Register creates the account but does NOT start a session (no Set-Cookie);
      // the caller then logs in. So we do not mark the user authenticated here.
      return data;
    },
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.POST("/auth/logout", {});
    },
    onSuccess: () => {
      qc.setQueryData(ME_KEY, null);
      qc.clear(); // drop every cached protected query on logout
    },
  });
}
