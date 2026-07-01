// Shared generator for the typed API client (SPEC-200 FR-2002).
// Single source of truth: the checked-in api/openapi.yaml. Both `gen:api` (writes
// lib/api/schema.ts) and `check:api` (drift guard) go through here, so they can never
// disagree on how types are produced.
import openapiTS, { astToString } from "openapi-typescript";

// scripts/ -> web/ -> repo root -> api/openapi.yaml
const SPEC_URL = new URL("../../api/openapi.yaml", import.meta.url);

const HEADER = `// AUTO-GENERATED from api/openapi.yaml — DO NOT EDIT BY HAND.
// Regenerate with \`npm run gen:api\`; the web-ci drift check fails if this is stale.
// (SPEC-200 FR-2002 — the API contract is generated, never hand-written.)

`;

/** @returns {Promise<string>} the generated schema.ts contents */
export async function generateApiTypes() {
  const ast = await openapiTS(SPEC_URL);
  return HEADER + astToString(ast).replace(/\r\n/g, "\n").trimEnd() + "\n";
}

export const SCHEMA_URL = new URL("../lib/api/schema.ts", import.meta.url);
