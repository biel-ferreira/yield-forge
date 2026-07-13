import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import { ApiError, backendError } from "@/lib/api/error";
import type { components } from "@/lib/api/schema";

// Portfolio holdings (SPEC-211) — the frontend face of SPEC-102 (FII) + SPEC-109 (fixed income,
// incl. the resolved indexer_type/effective_annual_rate_bps). Types come from the generated
// contract; no hand-written DTOs (BR-2116).
export type FIIHolding = components["schemas"]["FIIHoldingResponse"];
export type FIIHoldingInput = components["schemas"]["FIIHoldingRequest"];
export type FixedIncomeHolding = components["schemas"]["FixedIncomeResponse"];
export type FixedIncomeInput = components["schemas"]["FixedIncomeRequest"];

const FII_KEY = ["holdings", "fii"] as const;
const FIXED_INCOME_KEY = ["holdings", "fixed-income"] as const;

/** The caller's FII holdings (GET /holdings/fii). */
export function useFIIHoldings() {
  const query = useQuery({
    queryKey: FII_KEY,
    queryFn: async () => {
      const { data } = await api.GET("/holdings/fii");
      if (!data) throw new Error("failed to load FII holdings");
      return data;
    },
  });
  return {
    holdings: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    refetch: query.refetch,
  };
}

/** Create an FII holding (POST /holdings/fii). Identity from the session — no user_id (BR-2111). */
export function useCreateFIIHolding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: FIIHoldingInput) => {
      const { data, error, response } = await api.POST("/holdings/fii", { body: input });
      if (error || !data) {
        throw new ApiError(
          response.status,
          backendError(error) ?? `create failed (${response.status})`,
        );
      }
      return data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: FII_KEY }),
  });
}

/** Update an owned FII holding (PUT /holdings/fii/{id}). A 404 throws ApiError(404, ...) — BR-2111. */
export function useUpdateFIIHolding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: FIIHoldingInput }) => {
      const { data, error, response } = await api.PUT("/holdings/fii/{id}", {
        params: { path: { id } },
        body: input,
      });
      if (error || !data) {
        throw new ApiError(
          response.status,
          backendError(error) ?? `update failed (${response.status})`,
        );
      }
      return data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: FII_KEY }),
  });
}

/** Delete an owned FII holding (DELETE /holdings/fii/{id}). A 404 is treated as success (BR-2111). */
export function useDeleteFIIHolding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error, response } = await api.DELETE("/holdings/fii/{id}", {
        params: { path: { id } },
      });
      if (error && response.status !== 404) {
        throw new ApiError(
          response.status,
          backendError(error) ?? `delete failed (${response.status})`,
        );
      }
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: FII_KEY }),
  });
}

/** The caller's fixed-income holdings (GET /holdings/fixed-income) — incl. the resolved indexer rate. */
export function useFixedIncomeHoldings() {
  const query = useQuery({
    queryKey: FIXED_INCOME_KEY,
    queryFn: async () => {
      const { data } = await api.GET("/holdings/fixed-income");
      if (!data) throw new Error("failed to load fixed-income holdings");
      return data;
    },
  });
  return {
    holdings: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    refetch: query.refetch,
  };
}

/** Create a fixed-income holding (POST /holdings/fixed-income). No user_id (BR-2111). */
export function useCreateFixedIncomeHolding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: FixedIncomeInput) => {
      const { data, error, response } = await api.POST("/holdings/fixed-income", { body: input });
      if (error || !data) {
        throw new ApiError(
          response.status,
          backendError(error) ?? `create failed (${response.status})`,
        );
      }
      return data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: FIXED_INCOME_KEY }),
  });
}

/** Update an owned fixed-income holding (PUT /holdings/fixed-income/{id}). A 404 throws ApiError(404, ...). */
export function useUpdateFixedIncomeHolding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: FixedIncomeInput }) => {
      const { data, error, response } = await api.PUT("/holdings/fixed-income/{id}", {
        params: { path: { id } },
        body: input,
      });
      if (error || !data) {
        throw new ApiError(
          response.status,
          backendError(error) ?? `update failed (${response.status})`,
        );
      }
      return data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: FIXED_INCOME_KEY }),
  });
}

/** Delete an owned fixed-income holding (DELETE /holdings/fixed-income/{id}). A 404 = success (BR-2111). */
export function useDeleteFixedIncomeHolding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error, response } = await api.DELETE("/holdings/fixed-income/{id}", {
        params: { path: { id } },
      });
      if (error && response.status !== 404) {
        throw new ApiError(
          response.status,
          backendError(error) ?? `delete failed (${response.status})`,
        );
      }
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: FIXED_INCOME_KEY }),
  });
}
