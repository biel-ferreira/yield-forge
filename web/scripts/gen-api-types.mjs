// Writes web/lib/api/schema.ts from api/openapi.yaml. Run: `npm run gen:api`.
import { mkdirSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { generateApiTypes, SCHEMA_URL } from "./_api-types.mjs";

const out = fileURLToPath(SCHEMA_URL);
mkdirSync(fileURLToPath(new URL("../lib/api/", import.meta.url)), { recursive: true });
writeFileSync(out, await generateApiTypes());
console.log("✓ wrote lib/api/schema.ts from api/openapi.yaml");
