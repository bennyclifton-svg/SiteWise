import { expect, test } from "@playwright/test";
import { writeFile } from "node:fs/promises";

// The Cost Plan's Progress claims tab and the payment schedule sheet
// (docs/plans/2026-11-26-payment-schedules.md, slice 2): drop the
// contractor's claim on its contract, create the schedule as claimed, cut
// one line by percent with its reason, see the totals, issue. Every figure
// is the server's.
test("a dropped claim becomes a payment schedule the PM cuts and issues", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Progress claims");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  await page.request.post(`/__e2e/contract?project=${project}`);

  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await expect(page.getByRole("tab")).toHaveText(["Cost Plan", "Invoices", "Variations", "Progress claims"]);
  await page.getByRole("tab", { name: "Progress claims" }).click();
  const list = page.getByRole("region", { name: "Progress claims" });
  await expect(list.getByLabel("Main Works figures")).toContainText("150,000.00");
  await expect(list).toContainText("No claims yet");

  // Drop the claim on its contract: read, listed, nothing assessed yet.
  const claim = await (await page.request.get("/__e2e/claim.md")).body();
  await page.getByTestId("claim-input").setInputFiles({ name: "PC-04.md", mimeType: "text/markdown", buffer: claim });
  const row = list.locator("tr.pc-claim");
  await expect(row).toContainText("PC-04", { timeout: 20_000 });
  await expect(row).toContainText("Draft");

  // Create schedule: every line as claimed, and the sheet opens.
  await row.getByRole("button", { name: "Create schedule" }).click();
  const sheet = page.getByRole("region", { name: "Payment schedule PC-04" });
  await expect(sheet.getByRole("heading", { level: 1 })).toContainText("Payment schedule PC-04 · Main Works");
  const dollars = sheet.getByLabel("Substructure: completed to date, dollars");
  await expect(dollars).toHaveValue("60,000.00");
  await expect(sheet.getByLabel("Substructure: completed to date, percent")).toHaveValue("60.0");
  await expect(sheet.getByText("Ready to issue")).toBeVisible();
  await expect(sheet.getByText("Progress claim to be paid in full.")).toBeVisible();
  const total = sheet.getByRole("row", { name: /Total claim inc retention/ });
  await expect(total).toContainText("83,700.00");

  // Cut one line by percent: the dollars are the server's, and a reason is asked for.
  await sheet.getByLabel("Substructure: completed to date, percent").fill("50");
  await sheet.getByLabel("Substructure: completed to date, percent").press("Enter");
  await expect(dollars).toHaveValue("50,000.00");
  await expect(sheet.getByText("claimed 60,000.00")).toBeVisible();
  await expect(total).toContainText("73,700.00");
  const issue = sheet.getByRole("button", { name: "Issue payment schedule" });
  await expect(issue).toBeDisabled();
  await expect(sheet.getByText("1 line needs a reason before Issue.")).toBeVisible();
  await sheet.getByLabel("Reason for Substructure").fill("Footings to grid C not poured");
  await sheet.getByLabel("Reason for Substructure").press("Enter");
  await expect(issue).toBeEnabled();
  // The payment list carries this claim's row and totals.
  const payments = sheet.getByRole("table").filter({ hasText: "Payment list" });
  await expect(payments).toContainText("73,700.00");
  await expect(payments).toContainText("81,070.00");
  await page.screenshot({ path: testInfo.outputPath("schedule-draft.png"), fullPage: true });

  // Issue is the PM's press: frozen, and the claim counts at what was scheduled.
  await issue.click();
  await expect(sheet.getByRole("link", { name: "Download PDF" })).toBeVisible();
  await expect(sheet.getByLabel("Substructure: completed to date, dollars")).toHaveCount(0);
  const ledger = await (await page.request.get(`/api/projects/${project}/invoices`)).json() as { summary: { invoiced: string }; contracts: { paid_to_date: string }[] };
  expect(ledger.summary.invoiced).toBe("76000.00");
  expect(ledger.contracts[0].paid_to_date).toBe("73700.00");

  // Served, then back to the list: the row says so and the banner is gone.
  await sheet.getByLabel("Schedule served on").fill("2026-10-20");
  await sheet.getByLabel("Schedule served on").blur();
  await page.getByRole("button", { name: "Back to Progress claims" }).click();
  // Beside the open document panel the list keeps number, claimed,
  // scheduled, payable, due and status; the rest is on the sheet.
  await expect(row.getByText("Served")).toBeVisible();
  await expect(row.getByText("81,070.00")).toBeVisible();
  await expect(row.getByText("Awaiting invoice")).toBeHidden();
  await expect(list.getByLabel("Main Works figures")).toContainText("73,700.00");
  await expect(page.locator(".claim-banner")).toHaveCount(0);
  await page.screenshot({ path: testInfo.outputPath("claims-list.png"), fullPage: true });
});

// Slice 3: the lines a schedule prints are the contract's breakdown. The PM
// adds a provisional sum from the sheet, assesses it, and marks it final;
// the sheet says when the breakdown no longer adds up to its cost line.
test("the PM adds a provisional sum to a contract's breakdown and marks it final", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Contract breakdown");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  await page.request.post(`/__e2e/contract?project=${project}`);

  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await page.getByRole("tab", { name: "Progress claims" }).click();
  const claim = await (await page.request.get("/__e2e/claim.md")).body();
  await page.getByTestId("claim-input").setInputFiles({ name: "PC-04.md", mimeType: "text/markdown", buffer: claim });
  const row = page.getByRole("region", { name: "Progress claims" }).locator("tr.pc-claim");
  await expect(row).toContainText("PC-04", { timeout: 20_000 });
  await row.getByRole("button", { name: "Create schedule" }).click();
  const sheet = page.getByRole("region", { name: "Payment schedule PC-04" });
  await expect(sheet.getByLabel("Substructure: completed to date, dollars")).toHaveValue("60,000.00");
  await expect(sheet.getByRole("table").filter({ hasText: "Provisional sum adjustment" })).toHaveCount(0);

  // Edit breakdown: a grey row under each section adds a line.
  await sheet.getByRole("button", { name: "Schedule PC-04 actions" }).click();
  await page.getByRole("menuitem", { name: "Edit breakdown" }).click();
  await sheet.getByLabel("New provisional sum", { exact: true }).fill("Rock, 40 tonne @ $120");
  await sheet.getByLabel("New provisional sum amount").fill("4800");
  await sheet.getByLabel("New provisional sum amount").press("Enter");
  const provisional = sheet.getByRole("table").filter({ hasText: "Provisional sum adjustment" });
  await expect(provisional.getByLabel("Description of Rock, 40 tonne @ $120")).toHaveValue("Rock, 40 tonne @ $120");
  // The sum sits under a cost line whose own value has not changed, and the sheet says so.
  await expect(sheet.getByLabel("Breakdown checks")).toContainText("Breakdown is $4,800.00 over the contract line Substructure.");
  await sheet.getByLabel("Sum for Substructure").fill("95200");
  await sheet.getByLabel("Sum for Substructure").press("Enter");
  await expect(sheet.getByLabel("Breakdown checks")).toHaveCount(0);
  await sheet.getByRole("button", { name: "Done", exact: true }).click();

  // Assess the sum by percent, and mark it final.
  await provisional.getByLabel("Rock, 40 tonne @ $120: completed to date, percent").fill("50");
  await provisional.getByLabel("Rock, 40 tonne @ $120: completed to date, percent").press("Enter");
  await expect(provisional.getByLabel("Rock, 40 tonne @ $120: completed to date, dollars")).toHaveValue("2,400.00");
  // The tick shows what is saved, so it changes when the save returns.
  await provisional.getByLabel("Rock, 40 tonne @ $120 is final").click();
  await expect(provisional.getByLabel("Rock, 40 tonne @ $120 is final")).toBeChecked();
  // Settling a sum on this claim is not claiming it again.
  await expect(sheet.getByText("Marked final before this claim")).toHaveCount(0);
  await page.screenshot({ path: testInfo.outputPath("breakdown.png"), fullPage: true });

  // Issued: the contract's adjusted sum takes the final sum at what was spent.
  await sheet.getByRole("button", { name: "Issue payment schedule" }).click();
  await expect(sheet.getByRole("link", { name: "Download PDF" })).toBeVisible();
  const ledger = await (await page.request.get(`/api/projects/${project}/invoices`)).json() as { contracts: { adjusted: string; provisional_adjustment: string }[] };
  expect(ledger.contracts[0].provisional_adjustment).toBe("-2400.00");
  expect(ledger.contracts[0].adjusted).toBe("147600.00");
});

// Slice 4: the contract's terms are typed once from the sheet. Retention is a
// percent of a base; the claim says what it holds as a percent and the
// server works out the dollars. The parties and the PM's note print on the
// schedule; nothing is filled in that the PM did not type.
test("the PM types the contract's terms, holds retention by percent and adds a note", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Contract terms");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  await page.request.post(`/__e2e/contract?project=${project}`);

  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await page.getByRole("tab", { name: "Progress claims" }).click();
  const claim = await (await page.request.get("/__e2e/claim.md")).body();
  await page.getByTestId("claim-input").setInputFiles({ name: "PC-04.md", mimeType: "text/markdown", buffer: claim });
  const row = page.getByRole("region", { name: "Progress claims" }).locator("tr.pc-claim");
  await expect(row).toContainText("PC-04", { timeout: 20_000 });
  await row.getByRole("button", { name: "Create schedule" }).click();
  const sheet = page.getByRole("region", { name: "Payment schedule PC-04" });
  await expect(sheet.getByLabel("Substructure: completed to date, dollars")).toHaveValue("60,000.00");
  // With no terms, retention is the dollars held to date, as the claim says.
  await expect(sheet.getByLabel("Retention held to date, dollars")).toHaveValue("2,300.00");
  await expect(sheet.getByText("This is a payment schedule.")).toBeVisible();

  // Contract terms, from the three dots: each field saves when it is left.
  await sheet.getByRole("button", { name: "Schedule PC-04 actions" }).click();
  await page.getByRole("menuitem", { name: "Contract terms" }).click();
  const terms = sheet.locator(".ps-terms");
  await terms.getByLabel("Contract date").fill("2026-02-25");
  await terms.getByLabel("Contract date").blur();
  await expect(sheet.getByText("25 Feb 2026")).toBeVisible();
  await terms.getByLabel("Retention rate, percent").fill("5");
  await terms.getByLabel("Retention rate, percent").press("Enter");
  const totals = sheet.getByRole("table").filter({ hasText: "Total and retention" });
  await expect(totals.getByRole("row", { name: /Retention limit: 5% of the works sub total/ })).toContainText("7,500.00");

  // The claim's own 2,300 stands until the PM says otherwise: hold 2.5%.
  await expect(totals.getByLabel("Retention this claim, dollars")).toHaveValue("2,300.00");
  await totals.getByLabel("Retention this claim, percent").fill("2.5");
  await totals.getByLabel("Retention this claim, percent").press("Enter");
  await expect(totals.getByLabel("Retention this claim, dollars")).toHaveValue("3,750.00");
  await expect(sheet.getByRole("row", { name: /Total claim inc retention/ })).toContainText("82,250.00");
  // Holding more than the contractor claimed pays less than claimed, so it needs its reason.
  await expect(sheet.getByText("1 line needs a reason before Issue.")).toBeVisible();
  await sheet.getByLabel("Reason for retention").fill("The contract holds 2.5% of the works at this stage");
  await sheet.getByLabel("Reason for retention").press("Enter");
  await expect(sheet.getByText("1 line needs a reason before Issue.")).toHaveCount(0);
  // Past the limit is refused, in words, and nothing changes.
  await totals.getByLabel("Retention this claim, percent").fill("6");
  await totals.getByLabel("Retention this claim, percent").press("Enter");
  await expect(sheet.getByRole("alert")).toBeVisible();
  await expect(sheet.getByRole("row", { name: /Total claim inc retention/ })).toContainText("82,250.00");

  // The parties, typed once for the project; a number that is neither an ACN nor an ABN is said.
  await terms.getByLabel("Principal", { exact: true }).fill("Harbour Trust");
  await terms.getByLabel("Principal", { exact: true }).press("Enter");
  const details = sheet.locator(".ps-details");
  await expect(details).toContainText("Harbour Trust");
  await terms.getByLabel("Rep's ACN or ABN").fill("605 0814 883");
  await terms.getByLabel("Rep's ACN or ABN").press("Enter");
  await expect(details.locator("p.ps-flag")).toContainText("605 0814 883");
  await expect(details.locator("p.ps-flag")).toContainText("It prints as typed.");
  await terms.getByLabel("Rep's ACN or ABN").fill("605 081 488");
  await terms.getByLabel("Rep's ACN or ABN").press("Enter");
  await expect(details.locator("p.ps-flag")).toHaveCount(0);
  await expect(details).toContainText("ACN 605 081 488");

  // The PM's own words, after SiteWise's sentence.
  await sheet.getByLabel("Your note, printed on the schedule").fill("Retention is held on the works only.");
  await sheet.getByLabel("Your note, printed on the schedule").blur();
  await page.screenshot({ path: testInfo.outputPath("terms-totals.png") });
  await sheet.getByRole("heading", { level: 1 }).scrollIntoViewIfNeeded();
  await page.screenshot({ path: testInfo.outputPath("terms.png") });
  await terms.getByRole("button", { name: "Close" }).click();
  await expect(terms).toHaveCount(0);

  await sheet.getByRole("button", { name: "Issue payment schedule" }).click();
  await expect(sheet.getByRole("link", { name: "Download PDF" })).toBeVisible();
  await expect(sheet.getByText("Retention is held on the works only.")).toBeVisible();
  const ledger = await (await page.request.get(`/api/projects/${project}/invoices`)).json() as { contracts: { retention_percent: string; contract_date: string; paid_to_date: string }[] };
  expect(ledger.contracts[0].retention_percent).toBe("5");
  expect(ledger.contracts[0].contract_date).toBe("2026-02-25");
  expect(ledger.contracts[0].paid_to_date).toBe("82250.00");
  await page.screenshot({ path: testInfo.outputPath("terms-issued.png") });
});

// Slice 5: once issued, the one action is Send. It opens the Send dialog
// filled in with the issued PDF; the PM presses Send there. The day the
// message went is entered as the day served, for the PM to confirm.
test("an issued schedule is sent from the sheet and its served date is confirmed", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  // A free plan sends only from an account a week old.
  await page.request.post("/__e2e/age-accounts");
  await page.getByLabel("New project").fill("Schedule send");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  await page.request.post(`/__e2e/contract?project=${project}`);

  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await page.getByRole("tab", { name: "Progress claims" }).click();
  const claim = await (await page.request.get("/__e2e/claim.md")).body();
  await page.getByTestId("claim-input").setInputFiles({ name: "PC-04.md", mimeType: "text/markdown", buffer: claim });
  const row = page.getByRole("region", { name: "Progress claims" }).locator("tr.pc-claim");
  await expect(row).toContainText("PC-04", { timeout: 20_000 });
  await row.getByRole("button", { name: "Create schedule" }).click();
  const sheet = page.getByRole("region", { name: "Payment schedule PC-04" });
  await expect(sheet.getByLabel("Substructure: completed to date, dollars")).toHaveValue("60,000.00");

  // The principal's address, typed once for the project.
  await sheet.getByRole("button", { name: "Schedule PC-04 actions" }).click();
  await page.getByRole("menuitem", { name: "Contract terms" }).click();
  const terms = sheet.locator(".ps-terms");
  await terms.getByLabel("Principal's email").fill("principal@client.test");
  await terms.getByLabel("Principal's email").press("Enter");
  await expect(terms.getByLabel("Principal's email")).toHaveValue("principal@client.test");
  await terms.getByRole("button", { name: "Close" }).click();
  // Who approves it is on the sheet before it is issued.
  await expect(sheet.locator(".ps-approval")).toContainText("Approved by");

  // Issue: the frozen PDF, in the owner's layout.
  await sheet.getByRole("button", { name: "Issue payment schedule" }).click();
  const link = sheet.getByRole("link", { name: "Download PDF" });
  await expect(link).toBeVisible();
  const pdf = await page.request.get((await link.getAttribute("href"))!);
  expect(pdf.status()).toBe(200);
  expect((await pdf.body()).subarray(0, 4).toString()).toBe("%PDF");
  await writeFile(testInfo.outputPath("schedule.pdf"), await pdf.body());
  await expect(sheet.locator(".register-chip").first()).toHaveText("Issued");

  // Send: filled in, and nothing leaves until the PM presses Send there.
  await sheet.getByRole("button", { name: "Send", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByLabel("Subject")).toHaveValue("Payment schedule PC-04 · Main Works");
  await expect(dialog.getByLabel("To")).toHaveValue(/principal@client\.test/);
  await expect(dialog).toContainText("The payment schedule goes as it was issued");
  await page.screenshot({ path: testInfo.outputPath("send-dialog.png") });
  await dialog.getByRole("button", { name: "Send" }).click();
  await expect(dialog).toHaveCount(0);
  // The sink reports the raw header, and this subject is encoded for its middle dot; the address says which message it is.
  await expect.poll(async () => ((await (await page.request.get("/__e2e/mail-sent")).json()).sent as { to: string[] }[] | undefined)?.some(m => m.to.includes("principal@client.test")) ?? false, { timeout: 15_000 }).toBeTruthy();

  // Sent: the day it went is the day served, by rule, for the PM to confirm.
  await expect(sheet.getByText(/Sent from SiteWise on/)).toBeVisible({ timeout: 15_000 });
  await expect(sheet.locator(".register-chip").first()).toHaveText("Sent");
  await expect(sheet.getByRole("button", { name: "Send", exact: true })).toHaveCount(0);
  await sheet.locator(".ps-issued").scrollIntoViewIfNeeded();
  await page.screenshot({ path: testInfo.outputPath("sent.png") });
  await sheet.getByRole("button", { name: /^Confirm / }).click();
  await expect(sheet.locator(".register-chip").first()).toHaveText("Served");
  await expect(sheet.getByRole("button", { name: /^Confirm / })).toHaveCount(0);
});

// Slice 6: while a claim waits for its tax invoice, an invoice no rule can
// place is held out of the totals; the PM places it with one press. One
// that matches the claim's figures links itself and is never counted.
test("an invoice the rule cannot place is held until the PM places it", async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Placing invoices");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  await page.request.post(`/__e2e/contract?project=${project}`);

  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await page.getByRole("tab", { name: "Progress claims" }).click();
  const claim = await (await page.request.get("/__e2e/claim.md")).body();
  await page.getByTestId("claim-input").setInputFiles({ name: "PC-04.md", mimeType: "text/markdown", buffer: claim });
  const list = page.getByRole("region", { name: "Progress claims" });
  await expect(list.locator("tr.pc-claim")).toContainText("PC-04", { timeout: 20_000 });

  // An invoice for an amount no claim figure matches: held, with two choices.
  const seeded = await page.request.post(`/__e2e/contract-invoice?project=${project}&number=INV-77&total=1234500&gst=112227`);
  expect(seeded.status(), await seeded.text()).toBe(200);
  const before = await (await page.request.get(`/api/projects/${project}/invoices`)).json() as { invoices: { number: string; held: boolean; status: string; supplier_id: string }[] };
  expect(JSON.stringify(before.invoices.map(i => [i.number, i.held, i.status, i.supplier_id]))).toContain("INV-77");
  expect(before.invoices.find(i => i.number === "INV-77")?.held, JSON.stringify(before.invoices)).toBe(true);
  // The seed writes no event, so the page is opened again to read it.
  await page.reload();
  const place = list.getByRole("region", { name: "Invoices to place" });
  await expect(place).toContainText("1 invoice to place", { timeout: 10_000 });
  await expect(place).toContainText("Invoice INV-77");
  await place.getByRole("button", { name: "For claim PC-04" }).click();
  await expect(place).toHaveCount(0);
  await expect(list.locator("tr.pc-claim")).toContainText("INV-77");
  const ledger = await (await page.request.get(`/api/projects/${project}/invoices`)).json() as { invoices: { number: string; settles_claim?: string; counted: boolean }[]; claims: { invoice_id: string; awaiting_invoice: boolean }[] };
  const placed = ledger.invoices.find(i => i.number === "INV-77")!;
  expect(placed.settles_claim).toBe(ledger.claims[0].invoice_id);
  expect(placed.counted).toBe(false);
  expect(ledger.claims[0].awaiting_invoice).toBe(false);

  // The Invoices tab says which claim it is for.
  await page.getByRole("tab", { name: "Invoices" }).click();
  await expect(page.getByText("· for claim PC-04")).toBeVisible();
});
