import createClient from "openapi-fetch";
import type { paths } from "./schema";

// Typed API client generated from api/openapi.yaml — no hand-written DTOs (SPEC-200 FR-2002).
//
// The browser calls the app's OWN origin under /api/*; Next.js rewrites to the backend
// (D1 same-origin proxy, wired in Phase 4) so the SPEC-003 HttpOnly session cookie stays
// same-origin. `credentials: "include"` sends it. Identity is resolved by the server via
// /auth/me — never trusted from client state (BR-2001).
export const api = createClient<paths>({
  baseUrl: "/api",
  credentials: "include",
});
