import { expect, test, type Page } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

// Combined engagements, slice 5: rows made before groups get a prompt to
// combine; a combined proposal on a split project waits unplaced with
// "Combine for this proposal"; "engaged by" is no longer a setting.

async function newProject(page: Page, name: string): Promise<string> {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  return `/api/projects/${new URL(page.url()).pathname.split("/").pop()!}`;
}

async function addRows(page: Page, api: string, disciplines: string[]): Promise<{ id: string; version: number }[]> {
  const rows = [];
  for (const discipline of disciplines) {
    const res = await page.request.post(`${api}/procurement/rows`, { headers: origin, data: { discipline_id: discipline } });
    expect(res.status()).toBe(201);
    rows.push(await res.json());
  }
  return rows;
}

test("a Services Engineer row beside split services is offered as one engagement", async ({ page }, testInfo) => {
  const api = await newProject(page, "Combine prompt");
  const rows = await addRows(page, api, ["consultant.services", "consultant.mechanical", "consultant.electrical"]);
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  const prompt = page.getByRole("note", { name: "Combine suggestion" });
  await expect(prompt).toContainText("covers services that also have their own rows");
  for (const name of ["Mechanical", "Electrical"]) await expect(prompt.getByLabel(new RegExp(`^Include ${name}`))).toBeChecked();
  // No "engaged by" control anywhere on Procurement, and the API refuses it.
  await expect(page.getByLabel(/engaged by/i)).toHaveCount(0);
  await expect(page.getByText(/engaged by/i)).toHaveCount(0);
  const refused = await page.request.patch(`${api}/packages/${rows[1].id}`, { headers: origin, data: { version: rows[1].version, engaged_by: "lead_consultant" } });
  expect(refused.status()).toBe(422);
  expect(await refused.text()).toContain("combine");
  await page.screenshot({ path: testInfo.outputPath("prompt.png") });

  await prompt.getByRole("button", { name: "Combine 3", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toContainText("become one engagement");
  await dialog.getByRole("button", { name: "Combine", exact: true }).click();
  await expect(page.getByRole("button", { name: /^Uncombine / })).toBeVisible();
  await expect(prompt).toHaveCount(0);
});

test("a dismissed combine prompt stays dismissed", async ({ page }) => {
  const api = await newProject(page, "Dismissed prompt");
  await addRows(page, api, ["consultant.services", "consultant.hydraulic"]);
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  const prompt = page.getByRole("note", { name: "Combine suggestion" });
  await prompt.getByRole("button", { name: "Dismiss", exact: true }).click();
  await expect(prompt).toHaveCount(0);
  await page.reload();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await expect(page.getByRole("button", { name: "New discipline" })).toBeVisible();
  await expect(prompt).toHaveCount(0);
});

test("a combined proposal on a split project is combined for by the PM", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const api = await newProject(page, "Combined proposal");
  const [mechanical] = await addRows(page, api, ["consultant.mechanical", "consultant.electrical"]);
  // A proposal filed as Services Engineer, waiting for a package: uploaded
  // on a row, detached, and its discipline set as the filing would.
  const pdf = await (await page.request.get(`/__e2e/proposal.pdf?firm=${encodeURIComponent("Volt Services Pty Ltd")}&abn=47&lines=${encodeURIComponent("Design~500000")}`)).body();
  const up = await page.request.post(`${api}/procurement/packages/${mechanical.id}/submissions?name=${encodeURIComponent("Services fee proposal.pdf")}`, { headers: { ...origin, "Content-Type": "application/pdf" }, data: pdf });
  expect(up.status()).toBe(201);
  const { submission_id: sub } = await up.json();
  const prices = `${api}/procurement/packages/${mechanical.id}/prices`;
  await expect.poll(async () => (await (await page.request.get(prices)).json()).columns[0]?.submission.status, { timeout: 20_000 }).not.toBe("reading");
  const document = (await (await page.request.get(prices)).json()).columns[0].submission.document_id;
  expect((await page.request.patch(`${api}/submissions/${sub}`, { headers: origin, data: { package_id: "" } })).ok()).toBeTruthy();
  expect((await page.request.put(`/api/documents/${document}/fields/discipline`, { headers: origin, data: { value: "consultant.services" } })).ok()).toBeTruthy();

  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  const waiting = page.locator(".procurement-unassigned");
  await expect(waiting).toContainText("1 proposal needs a package.");
  await expect(waiting).toContainText("covers several disciplines");
  await expect(waiting.getByLabel("Place Services fee proposal.pdf on one discipline")).toBeVisible();
  await waiting.getByRole("button", { name: "Combine for this proposal", exact: true }).click();
  // Nothing is ticked for the PM.
  await expect(waiting.getByRole("button", { name: "Combine 0", exact: true })).toBeDisabled();
  await waiting.getByLabel(/^Mechanical.* for Services fee proposal.pdf$/).check();
  await waiting.getByLabel(/^Electrical.* for Services fee proposal.pdf$/).check();
  await page.screenshot({ path: testInfo.outputPath("combine-for-proposal.png") });
  await waiting.getByRole("button", { name: "Combine 2", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toContainText("The proposal is placed on it.");
  const start = Date.now();
  await dialog.getByRole("button", { name: "Combine", exact: true }).click();
  const group = page.getByRole("row", { name: /^Electrical.* and Mechanical/ });
  await expect(group).toBeVisible();
  testInfo.annotations.push({ type: "timing", description: `combine for proposal UI: ${Date.now() - start}ms` });
  await expect(group).toContainText("Volt Services");
  await expect(waiting).toHaveCount(0);
});
