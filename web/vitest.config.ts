import { defineConfig, configDefaults } from "vitest/config";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL(".", import.meta.url)).replace(/[\\/]$/, "");

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": root }, // mirror tsconfig paths "@/*"
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
    css: false,
    exclude: [...configDefaults.exclude, "e2e/**"], // Playwright owns e2e/
  },
});
