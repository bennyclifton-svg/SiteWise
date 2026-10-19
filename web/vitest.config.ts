import { defineConfig } from "vitest/config";

// Unit tests only. Playwright owns tests/*.spec.ts; Vitest must not pick
// them up, and these tests need no React plugin or browser.
export default defineConfig({
  test: {
    include: ["src/**/*.test.ts"],
    environment: "node",
  },
});
