import { expect, test, type Page } from "@playwright/test";

const headers = { Origin: "http://127.0.0.1:4173" };
const proposalLabel = (kind: string) => kind === "investigation"
  ? "Capacity of the existing structure or ground for the new load"
  : `Synthetic ${kind} proposal`;
async function setup(page: Page, kind: string) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill(`Synthetic ${kind} review`);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Proposals", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  const created = await page.request.post(`/api/projects/${project}/works`, { headers, data: { title: "Synthetic rooftop plant", part_id: profile.parts[0].id, system_id: "mechanical.air-conditioning", action: "new" } });
  expect(created.ok()).toBeTruthy();
  const work = await created.json();
  const built = await page.request.post(`/__e2e/proposals?project=${project}&kind=${kind}`);
  expect(built.ok(), await built.text()).toBeTruthy();
  await page.getByRole("button", { name: "Proposals", exact: true }).click();
  await expect(page.getByText(/^Showing \d+ of \d+\./)).toBeVisible();
  const showAll = page.getByRole("button", { name: "Show all proposals", exact: true });
  if (await showAll.isVisible()) await showAll.click();
  await page.getByRole("button", { name: new RegExp(proposalLabel(kind)) }).first().click();
  await expect(page.getByText(/Draft knowledge\. Acceptance/)).toBeVisible();
  return { project, work };
}

for (const kind of ["investigation", "discipline", "obligation", "approval", "hold_point"]) {
  test(`${kind} proposal creates a real planning record and safely undoes it`, async ({ page }) => {
    const { project, work } = await setup(page, kind);
    const pkg = await (await page.request.post(`/api/projects/${project}/packages`, { headers, data: { title: "Synthetic services", kind: "services" } })).json();
    await page.getByRole("button", { name: "Reload saved proposals", exact: true }).click();
    await page.getByRole("button", { name: "Review acceptance", exact: true }).click();
    if (kind === "obligation") {
      await expect(page.getByRole("button", { name: "Accept for planning", exact: true })).toBeDisabled();
      await page.getByLabel("Package (required)", { exact: true }).selectOption(pkg.id);
      await page.getByLabel("Work item (optional)", { exact: true }).selectOption(work.id);
      await page.getByLabel("Responsibility role (optional)", { exact: true }).selectOption("design");
      await page.getByLabel("Stage (optional)", { exact: true }).selectOption(pkg.stages[0].id);
      await page.setViewportSize({ width: 1360, height: 1000 });
      await page.screenshot({ path: "../.tools/wp45-proposals-desktop.png", fullPage: true });
      await page.setViewportSize({ width: 390, height: 844 });
      await page.screenshot({ path: "../.tools/wp45-proposals-mobile.png", fullPage: true });
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
    }
    if (kind === "approval" || kind === "hold_point") {
      await page.getByLabel("Package (optional)", { exact: true }).selectOption(pkg.id);
      await page.getByLabel("Work item (optional)", { exact: true }).selectOption(work.id);
    }
    await page.getByRole("button", { name: "Accept for planning", exact: true }).click();
    await expect(page.getByRole("region", { name: "Saved decision", exact: true })).toBeVisible();
    const list = (await (await page.request.get(`/api/projects/${project}/proposals?show=all`)).json()).items;
    const decision = list.find((p: { label: string; decision?: unknown }) => p.label === proposalLabel(kind) && p.decision).decision;
    expect(decision.decision).toBe("accepted");
    const expectedType = kind === "investigation" ? "work_item" : kind === "discipline" ? "package" : kind === "obligation" ? "package_scope_item" : "delivery_item";
    expect(decision.created_record_type).toBe(expectedType);
    await page.getByRole("button", { name: "Undo decision", exact: true }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeFocused();
    await dialog.getByRole("button", { name: "Undo decision", exact: true }).click();
    await expect(page.getByRole("button", { name: "Review acceptance", exact: true })).toBeVisible();
    if (kind === "investigation") {
      let rows = (await (await page.request.get(`/api/projects/${project}/works`)).json()).items;
      expect(rows.some((w: { id: string }) => w.id === decision.created_record_id)).toBeFalsy();
      await page.getByRole("button", { name: "Review acceptance", exact: true }).click();
      await page.getByRole("button", { name: "Accept for planning", exact: true }).click();
      await expect(page.getByRole("region", { name: "Saved decision", exact: true })).toBeVisible();
      rows = (await (await page.request.get(`/api/projects/${project}/works`)).json()).items;
      const accepted = rows.find((w: { id: string }) => w.id === decision.created_record_id);
      expect(accepted).toBeTruthy();
      expect((await page.request.patch(`/api/projects/${project}/works/${accepted.id}`, { headers, data: { version: accepted.version, title: "Edited synthetic investigation" } })).ok()).toBeTruthy();
      await page.getByRole("button", { name: "Undo decision", exact: true }).click();
      await page.getByRole("dialog").getByRole("button", { name: "Undo decision", exact: true }).click();
      await expect(page.getByRole("alert")).toContainText("Undo was refused");
      await page.setViewportSize({ width: 1360, height: 1000 });
      await page.screenshot({ path: "../.tools/wp45-proposals-undo-blocked.png", fullPage: true });
    }
  });
}

test("proposal review preserves choices and refuses stale inputs", async ({ page }) => {
  const { project } = await setup(page, "investigation");
  await page.getByRole("button", { name: "Dismiss proposal", exact: true }).click();
  await page.getByLabel("Reason for dismissal (optional)").fill("Synthetic owner review pending");
  await page.getByRole("button", { name: "Works", exact: true }).click();
  await page.getByRole("button", { name: "Proposals", exact: true }).click();
  await expect(page.getByLabel("Reason for dismissal (optional)")).toHaveValue("Synthetic owner review pending");
  await page.route(`**/api/projects/${project}/proposals*`, async route => {
    const response = await route.fetch(); const data = await response.json();
    data.items.forEach((p: { inputs_fingerprint: string }) => p.inputs_fingerprint = "f".repeat(64));
    await route.fulfill({ response, json: data });
  });
  await page.getByRole("button", { name: "Reload saved proposals", exact: true }).click();
  await expect(page.getByText("Saved proposal changed", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Save dismissal", exact: true })).toBeDisabled();
  await expect(page.getByLabel("Reason for dismissal (optional)")).toHaveValue("Synthetic owner review pending");
  await page.screenshot({ path: "../.tools/wp45-proposals-stale.png", fullPage: true });
  await page.unroute(`**/api/projects/${project}/proposals*`);
  await page.getByRole("button", { name: "Cancel review", exact: true }).click();
  await page.getByRole("button", { name: "Reload saved proposals", exact: true }).click();
  await page.getByRole("button", { name: "Dismiss proposal", exact: true }).click();
  await page.getByLabel("Reason for dismissal (optional)").fill("Explicit synthetic dismissal");
  await page.getByRole("button", { name: "Save dismissal", exact: true }).click();
  await expect(page.getByRole("region", { name: "Saved decision", exact: true })).toContainText("Explicit synthetic dismissal");
});

test("proposal loading failures remain distinct from an empty list", async ({ page }) => {
  await setup(page, "approval");
  await page.route("**/api/projects/*/proposals*", route => route.fulfill({ status: 500, body: "unavailable" }));
  await page.getByRole("button", { name: "Reload saved proposals", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("Could not load saved proposals");
  await expect(page.getByRole("button", { name: "Review acceptance", exact: true })).toBeDisabled();
  await page.unroute("**/api/projects/*/proposals*");
  await page.getByRole("button", { name: "Reload saved proposals", exact: true }).click();
  await expect(page.getByRole("alert")).not.toBeVisible();
  await expect(page.getByRole("button", { name: "Review acceptance", exact: true })).toBeEnabled();
});
