import { existsSync, mkdirSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig, devices } from "@playwright/test";

// The e2e server is the real Go server with a recorded Jev and the dedicated
// sitewise_test database. Run `npm run build` first: the server embeds dist/.
const repo = resolve(import.meta.dirname, "..");
const localGo = resolve(repo, ".tools/go/bin/go.exe");
const go = existsSync(localGo) ? localGo : "go";
const server = resolve(repo, `.tools/sitewise-e2e${process.platform === "win32" ? ".exe" : ""}`);
const temp = resolve(repo, "tmp");
mkdirSync(temp, { recursive: true });
const serverEnv = {
  GOCACHE: resolve(repo, ".tools/go-cache"),
  GOPATH: resolve(repo, ".tools/go-path"),
  GOTOOLCHAIN: "local",
  TEMP: temp,
  TMP: temp,
  SITEWISE_TEST_DATABASE_URL:
    process.env.SITEWISE_TEST_DATABASE_URL ?? "postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable",
};
const port = 4173;

export default defineConfig({
  testDir: "tests",
  testMatch: "*.spec.ts",
  timeout: 60_000,
  expect: { timeout: 10_000 },
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  globalTeardown: "./tests/global-teardown.ts",
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    // Avoid go run's temporary child; teardown stops the actual test binary.
    command: `"${go}" build -o "${server}" ./web/tests/e2eserver && "${server}" -addr 127.0.0.1:${port}`,
    cwd: repo,
    url: `http://127.0.0.1:${port}/index.html`,
    timeout: 240_000,
    reuseExistingServer: false,
    stdout: "pipe",
    env: serverEnv,
  },
});
