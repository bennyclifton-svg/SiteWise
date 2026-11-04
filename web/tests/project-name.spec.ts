import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";

test("project header edits in place, persists, cancels and recovers a failed save", async ({ page }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Original project");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const path = new URL(page.url()).pathname;
  const name = page.getByRole("textbox", { name: "Project name", exact: true });
  await expect(name).toHaveValue("Original project");
  await name.click({ position: { x: 25, y: 15 } });
  const caret = await name.evaluate((el: HTMLInputElement) => [el.selectionStart!, el.selectionEnd!]);
  expect(caret[0]).toBeGreaterThan(0);
  expect(caret[0]).toBeLessThan("Original project".length);
  expect(caret[0]).toBe(caret[1]);
  await name.fill("  Renamed project  ");
  await name.press("Enter");
  await expect(name).toHaveValue("Renamed project");
  await page.reload();
  await expect(name).toHaveValue("Renamed project");
  await name.fill("Cancelled");
  await name.press("Escape");
  await expect(name).toHaveValue("Renamed project");
  expect((await (await page.request.get(`/api${path}`)).json()).name).toBe("Renamed project");
  await name.fill("   ");
  await name.press("Enter");
  await expect(page.getByRole("alert")).toContainText("Enter a project name");
  await name.press("Escape");
  await page.route(`**/api${path}`, route => route.request().method() === "PATCH" ? route.fulfill({ status: 503 }) : route.continue());
  await name.fill("Retry project");
  await name.press("Tab");
  await expect(page.getByRole("alert")).toContainText("wasn’t saved");
  await expect(name).toHaveValue("Retry project");
  await page.unroute(`**/api${path}`);
  await name.click();
  await name.press("Tab");
  await expect(page.getByRole("alert")).toHaveCount(0);
  await page.reload();
  await expect(name).toHaveValue("Retry project");
  await page.screenshot({ path: "../tmp/project-name-desktop.png" });
  await page.setViewportSize({ width: 390, height: 844 });
  await name.focus();
  await page.screenshot({ path: "../tmp/project-name-mobile.png" });

  const headers = { Origin: new URL(page.url()).origin };
  expect((await page.request.patch(`/api${path}`, { headers, data: { name: " " } })).status()).toBe(400);
  expect((await page.request.patch(`/api${path}`, { headers: { Origin: "https://wrong.example" }, data: { name: "Wrong" } })).status()).toBe(403);
  const { token: other } = await (await page.request.post("/__e2e/invite?org=b")).json();
  await page.request.post("/api/session", { headers, data: { token: other } });
  expect((await page.request.patch(`/api${path}`, { headers, data: { name: "Wrong org" } })).status()).toBe(404);
});

test("project rename meets the profile edit speed budget", async ({ page }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await expect(page.getByLabel("New project")).toBeVisible();
  const result = await page.evaluate(async () => {
    const { id } = await (await fetch("/api/projects", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: "Rename speed" }) })).json();
    const samples: number[] = [];
    for (let i = 0; i < 25; i++) {
      const start = performance.now();
      const res = await fetch(`/api/projects/${id}`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: `Rename speed ${i}` }) });
      if (!res.ok) throw new Error(`rename: ${res.status}`);
      samples.push(performance.now() - start);
    }
    samples.sort((a,b) => a-b);
    return { p50: samples[12], p90: samples[22] };
  });
  const budget = JSON.parse(readFileSync(new URL("../../bench/budgets.json", import.meta.url), "utf8")).paths.find((p: { name: string }) => p.name === "profile_edit");
  console.log("Project rename latency (ms):", result);
  expect(result.p50).toBeLessThanOrEqual(budget.p50_us / 1000);
  expect(result.p90).toBeLessThanOrEqual(budget.p90_us / 1000);
});
