import { expect, test } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

interface Item { key: string; label: string; base: boolean }

// Combined engagements, slice 3: a proposal on a group is compared on the
// group's fee schedule (each discipline's stages, then combined stages),
// awarded under each discipline's fee line and the group's line, and the
// cost plan shows one contract sum expanding to the members.
test("a group's proposal is compared by discipline, awarded under member lines and shown as one sum", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Group award");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const api = `/api/projects/${project}`;
  const ids: string[] = [];
  for (const discipline of ["consultant.structural", "consultant.civil"]) {
    const row = await (await page.request.post(`${api}/procurement/rows`, { headers: origin, data: { discipline_id: discipline } })).json();
    ids.push(row.id);
  }
  expect((await page.request.post(`${api}/cost-plan/prepare`, { headers: origin, data: {} })).ok()).toBeTruthy();
  const group = await (await page.request.post(`${api}/packages/combine`, { headers: origin, data: { package_ids: ids } })).json();
  const prices = `${api}/procurement/packages/${group.id}/prices`;
  const schedule = await (await page.request.get(prices)).json();
  const items: Item[] = schedule.items;
  const combined = items.find(i => i.key.startsWith("combined:stage:"))!;
  expect(combined).toBeTruthy();
  const stage = combined.key.slice("combined:".length);
  // The firm prices a stage of each discipline by name, and gives one
  // stage once for both (entered in the combined cell below).
  const keys: string[] = schedule.group.members.map((m: { key: string }) => m.key);
  const member = keys.map(k => items.find(i => i.base && i.key.startsWith(`${k}:stage:`) && !i.key.endsWith(`:${stage}`))!);
  const lines = member.map((i, n) => `${i.label}~${100000 + 1000 * n}`).join(",");
  const pdf = await (await page.request.get(`/__e2e/proposal.pdf?firm=${encodeURIComponent("Karrak Engineers Pty Ltd")}&abn=9&lines=${encodeURIComponent(lines)}`)).body();
  const up = await page.request.post(`${api}/procurement/packages/${group.id}/submissions?name=${encodeURIComponent("Fee proposal (Karrak).pdf")}`, { headers: { ...origin, "Content-Type": "application/pdf" }, data: pdf });
  expect(up.status()).toBe(201);
  await expect.poll(async () => (await (await page.request.get(prices)).json()).columns[0]?.submission.status, { timeout: 20_000 }).not.toBe("reading");

  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("row", { name: new RegExp(`^${group.title}`) }).getByRole("button", { name: `Review ${group.title}`, exact: true }).click();
  const table = page.locator(".prices-table");
  await expect(table.getByRole("rowheader", { name: /^Combined stages/ })).toBeVisible();
  await expect(table.getByRole("rowheader", { name: /^Structural/ }).first()).toBeVisible();
  await expect(table.getByRole("rowheader", { name: /^Civil/ }).first()).toBeVisible();
  // The stage given once is entered once, in the combined cell.
  await table.getByRole("button", { name: new RegExp(`^${combined.label}, Karrak Engineers Pty Ltd: `) }).click();
  await table.getByLabel("Amount ex GST").fill("5000");
  await table.getByRole("button", { name: "Save", exact: true }).click();
  await expect(table.getByRole("button", { name: new RegExp(`^${combined.label}, Karrak Engineers Pty Ltd: \\$5,000\\.00`) })).toBeVisible();
  const expected = member.reduce((sum, _, n) => sum + 100000 + 1000 * n, 0) + 500000;
  const dollars = (c: number) => `$${(c / 100).toLocaleString("en-AU", { minimumFractionDigits: 2 })}`;
  await expect(table.locator("tr.prices-total")).toContainText(`${dollars(expected)} known`);
  await page.screenshot({ path: testInfo.outputPath("group-prices.png"), fullPage: true });

  await table.getByRole("button", { name: "Award", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText("To be written:")).toContainText(dollars(expected));
  await dialog.getByRole("button", { name: "Next: review" }).click();
  await expect(dialog.getByText(/under Structural/).first()).toBeVisible();
  await expect(dialog.getByText(new RegExp(`under ${group.title}`)).first()).toBeVisible();
  const start = Date.now();
  await dialog.getByRole("button", { name: "Award and write to cost plan" }).click();
  await expect(page.getByText(/Awarded to Karrak Engineers/)).toBeVisible();
  testInfo.annotations.push({ type: "timing", description: `group award UI: ${Date.now() - start}ms` });

  // One contract sum for the group, expanding to each discipline.
  await page.getByRole("button", { name: "Open cost plan" }).click();
  const toggle = page.getByRole("button", { name: new RegExp(`^(Collapse|Expand) ${group.title}$`) });
  await expect(toggle).toBeVisible();
  // Collapsed, the group's row is the one contract sum, computed in Go.
  if ((await toggle.getAttribute("aria-expanded")) === "true") await toggle.click();
  await expect(page.getByRole("row").filter({ has: page.getByRole("button", { name: `Expand ${group.title}`, exact: true }) })).toContainText((expected / 100).toLocaleString("en-AU")); // the cost plan shows whole dollars
  await page.getByRole("button", { name: `Expand ${group.title}`, exact: true }).click();
  await page.screenshot({ path: testInfo.outputPath("group-cost-plan.png"), fullPage: true });
  const plan = await (await page.request.get(`${api}/cost-plan`)).json();
  const groupLine = plan.items.find((i: { package_id: string; parent_item_id: string; line_kind: string }) => i.package_id === group.id && !i.parent_item_id && i.line_kind === "fee");
  const beneath = plan.items.filter((i: { parent_item_id: string; package_id: string }) => i.parent_item_id === groupLine.cost_item_id && ids.includes(i.package_id));
  expect(beneath).toHaveLength(2);
});
