import { writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The Go binary embeds dist/. A fresh clone builds Go before the first web
// build, so dist/.gitkeep is restored after Vite empties the directory.
const keepDist = {
  name: "keep-dist",
  closeBundle() {
    writeFileSync(resolve(import.meta.dirname, "dist/.gitkeep"), "");
  },
};

export default defineConfig({
  plugins: [react(), keepDist],
  build: { outDir: "dist", emptyOutDir: true, sourcemap: false },
  server: {
    // In development run the Go server on :8080 with
    // SITEWISE_PUBLIC_ORIGIN=http://localhost:5173 so origin checks pass.
    proxy: { "/api": { target: "http://127.0.0.1:8080" } },
  },
});
