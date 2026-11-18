import { expect, test } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

// docs/plans/2026-11-17-guest-access.md, Task 8: share an issued RFT from an
// invited firm's cell, open the link in a fresh browser, drop the quote,
// see the PM's cell change, revoke, and the link is refused.
test("an invited firm gets a link, sends its quote to its own cell, and loses the link when revoked", async ({ page, browser }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Guest links");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const row = await (await page.request.post(`/api/projects/${project}/procurement/rows`, { headers: origin, data: { discipline_id: "consultant.town_planner" } })).json();
  await page.request.post(`/api/projects/${project}/procurement/packages/${row.id}/invitations`, { headers: origin, data: { name: "Acme Planning" } });

  // Before the RFT is issued the link button says so.
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  const planner = page.getByRole("row", { name: /^Town Planner/ });
  const share = planner.getByRole("button", { name: "Share the RFT with Acme Planning" });
  await expect(share).toBeDisabled();
  await expect(share).toHaveAttribute("title", "Issue the RFT first");

  const report = await (await page.request.post(`/api/projects/${project}/reports`, { headers: origin, data: { kind: "rfp", package_id: row.id } })).json();
  await page.request.post(`/api/reports/${report.id}/draft`, { headers: origin, data: { use_last_completed: true } });
  expect((await page.request.post(`/__e2e/issue-rft?report=${report.id}`)).status()).toBe(204);
  await page.reload();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await expect(share).toBeEnabled();
  await share.click();

  const dialog = page.getByRole("dialog");
  await expect(dialog).toContainText("Share Town Planner with Acme Planning");
  await dialog.getByLabel("Shared with").fill("acme@example.test");
  await dialog.getByRole("button", { name: "Create link" }).click();
  const linkField = dialog.getByLabel("Link (shown once)");
  await expect(linkField).toBeVisible();
  const link = await linkField.inputValue();
  expect(link).toMatch(/\/g#t=[0-9a-f]{64}$/);
  await expect(dialog.getByRole("table", { name: "Who has a link" })).toContainText("acme@example.test");
  await dialog.getByRole("button", { name: "Done" }).click();
  const acmeCell = planner.locator("td.firm-slot").nth(0);
  await expect(acmeCell).toContainText("link not opened");

  // The firm opens the link in its own browser: no session, no workbench.
  const guestContext = await browser.newContext();
  const guest = await guestContext.newPage();
  await guest.goto(link);
  await expect(guest.getByRole("heading", { level: 1 })).toBeVisible();
  await expect(guest).toHaveURL(/\/g$/);
  await expect(guest.getByText("to acme@example.test")).toBeVisible();
  await expect(guest.getByRole("button", { name: "Download" })).toHaveCount(0);
  await expect(guest.getByRole("link", { name: "Download" }).first()).toBeVisible();
  await expect(guest.getByRole("navigation")).toHaveCount(0);
  await expect(acmeCell).toContainText("link opened", { timeout: 15_000 });

  // The quote: the stand-in reader prices each schedule line it finds.
  const schedule = await (await page.request.get(`/api/projects/${project}/procurement/packages/${row.id}/prices`)).json();
  const lines = schedule.items.filter((i: { base: boolean }) => i.base).map((i: { label: string }) => `${i.label}~100000`).join(",");
  const pdf = await (await page.request.get(`/__e2e/proposal.pdf?firm=Acme%20Planning&abn=9&lines=${encodeURIComponent(lines)}`)).body();
  await guest.locator('input[type="file"]').setInputFiles({ name: "quote.pdf", mimeType: "application/pdf", buffer: pdf });
  await expect(guest.getByText(/Received quote\.pdf/).first()).toBeVisible({ timeout: 20_000 });
  await expect(acmeCell.locator('button.firm-chip:not([data-status="invited"])')).toContainText("Acme Planning", { timeout: 20_000 });
  await expect(acmeCell).not.toContainText("Invited");

  // Revoke: the PM withdraws it; the firm's next click is refused.
  const links = await (await page.request.get(`/api/projects/${project}/shares?package=${row.id}`)).json();
  expect(links.shares).toHaveLength(1);
  expect((await page.request.delete(`/api/projects/${project}/shares/${links.shares[0].id}`, { headers: origin })).status()).toBe(200);
  await guest.reload();
  await expect(guest.getByRole("alert")).toHaveText("This link has expired or been withdrawn. Ask the person who sent it for a new one.");
  await guestContext.close();
});
