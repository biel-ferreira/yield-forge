// Drift guard: fails if the committed lib/api/schema.ts differs from a fresh
// generation of api/openapi.yaml — the client mirror of the backend openapi_test.go.
// Run: `npm run check:api` (also in web-ci).
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { generateApiTypes, SCHEMA_URL } from "./_api-types.mjs";

const out = fileURLToPath(SCHEMA_URL);
const fresh = (await generateApiTypes()).replace(/\r\n/g, "\n");

let committed = "";
try {
  committed = readFileSync(out, "utf8").replace(/\r\n/g, "\n");
} catch {
  console.error("✖ lib/api/schema.ts is missing. Run `npm run gen:api` and commit it.");
  process.exit(1);
}

if (fresh !== committed) {
  console.error(
    "✖ lib/api/schema.ts is out of date vs api/openapi.yaml.\n" +
      "  Run `npm run gen:api` and commit the result.",
  );
  process.exit(1);
}

console.log("✓ API types are in lockstep with api/openapi.yaml");
