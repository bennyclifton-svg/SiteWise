import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { expect, test, type Browser, type Locator, type Page } from "@playwright/test";

const fixtures = resolve(import.meta.dirname, "../../testdata/identity");
const fixture = (name: string) => readFileSync(resolve(fixtures, name));

async function signIn(browser: Browser, org: "a" | "b"): Promise<Page> {
  const context = await browser.newContext();
  const page = await context.newPage();
  const res = await page.request.post(`/__e2e/invite?org=${org}`);
  expect(res.ok()).toBeTruthy();
  const { token } = (await res.json()) as { token: string };
  await page.goto(`/#token=${token}`);
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();
  // The token is consumed and removed from the address bar.
  expect(page.url()).not.toContain("token");
  return page;
}

function block(page: Page, filename: string): Locator {
  return page.locator("article.tb").filter({ has: page.getByRole("heading", { name: filename, exact: true }) });
}

function cell(b: Locator, field: string): Locator {
  return b.locator(`.cell[data-field="${field}"]`);
}

async function dropFile(page: Page, name: string, bytes: Buffer, type: string) {
  const dataTransfer = await page.evaluateHandle(
    ({ name, b64, type }) => {
      const dt = new DataTransfer();
      const raw = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
      dt.items.add(new File([raw], name, { type }));
      return dt;
    },
    { name, b64: bytes.toString("base64"), type },
  );
  await page.dispatchEvent("main", "dragenter", { dataTransfer });
  await expect(page.locator(".drop")).toHaveAttribute("data-over", "true");
  await page.dispatchEvent("main", "drop", { dataTransfer });
}

test("file each format, correct a field, reconnect, and keep other orgs out", async ({ browser }) => {
  const page = await signIn(browser, "a");

  await page.getByLabel("New project").fill("14 Hale Street, Petersham");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("heading", { name: "14 Hale Street, Petersham" })).toBeVisible();
  const projectUrl = page.url();
  const projectId = projectUrl.split("/projects/")[1];
  await expect(page.getByRole("status")).toHaveText("Live");

  // A real drag and drop.
  await dropFile(page, "identity-page.pdf", fixture("identity-page.pdf"), "application/pdf");
  const pdf = block(page, "identity-page.pdf");
  await expect(pdf.locator(".tb-state")).toHaveText(/Filed in \d+\.\d\d s/);

  // The keyboard path: the Choose files button opens the picker.
  const chooser = page.waitForEvent("filechooser");
  await page.getByRole("button", { name: "Choose files" }).focus();
  await page.keyboard.press("Enter");
  await (
    await chooser
  ).setFiles([
    { name: "docx-table.docx", mimeType: "application/octet-stream", buffer: fixture("docx-table.docx") },
    { name: "merged-cells.xlsx", mimeType: "application/octet-stream", buffer: fixture("merged-cells.xlsx") },
    { name: "scanned-empty.pdf", mimeType: "application/pdf", buffer: fixture("scanned-empty.pdf") },
    { name: "slow-padded-document.docx", mimeType: "application/octet-stream", buffer: fixture("padded-document.docx") },
  ]);

  for (const name of ["docx-table.docx", "merged-cells.xlsx"]) {
    await expect(block(page, name).locator(".tb-state")).toHaveText(/Filed/);
    await expect(cell(block(page, name), "number")).toBeVisible();
  }

  // Stored but not filed says so in words, not only colour.
  const scan = block(page, "scanned-empty.pdf");
  await expect(scan.locator(".tb-state")).toHaveText("Stored · not filed");
  await expect(scan).toContainText("no text layer");

  // Jev missed its deadline: the unanswered boxes read "Not checked" and
  // carry no value, so they cannot pass for a judgement.
  const slow = block(page, "slow-padded-document.docx");
  await expect(slow.locator(".tb-state")).toHaveText(/Filed/, { timeout: 15_000 });
  const grey = slow.locator('.cell[data-state="unchecked"]');
  await expect(grey.first()).toContainText("Not checked");
  await expect(grey.first().locator(".cell-value")).toHaveText("—");

  // Every filed box states its band in text.
  const whys = await pdf.locator(".cell-why").allInnerTexts();
  expect(whys.length).toBe(8);
  for (const why of whys) {
    expect(why).toMatch(/From document|Confirmed · Jev|Check · Jev|Not set|Not checked|Set by you|No earlier filing/);
  }

  // Correct the title in place; focus returns to the box.
  const title = pdf.locator('button.cell[data-field="title"]');
  await title.click();
  const input = pdf.getByLabel("Title");
  await expect(input).toBeFocused();
  await input.fill("Ground Floor Plan");
  await input.press("Enter");
  await expect(title).toContainText("Ground Floor Plan");
  await expect(title).toContainText("Set by you");
  await expect(title).toHaveAttribute("data-state", "user");
  await expect(title).toBeFocused();
  await expect(page.locator(".tally")).toContainText("filed");

  // Reload: the correction is durable.
  await page.reload();
  await expect(cell(block(page, "identity-page.pdf"), "title")).toContainText("Ground Floor Plan");
  await expect(page.getByRole("status")).toHaveText("Live");

  // Drop the stream, correct from elsewhere, reconnect: the event replays
  // from the cursor without a reload.
  const docId = (await block(page, "identity-page.pdf").getAttribute("id"))!.replace("doc-", "");
  await page.context().setOffline(true);
  expect((await page.request.post("/__e2e/drop-streams")).ok()).toBeTruthy();
  await expect(page.getByRole("status")).not.toHaveText("Live");
  const put = await page.request.put(`/api/documents/${docId}/fields/revision`, {
    data: { value: "C" },
    headers: { Origin: new URL(projectUrl).origin },
  });
  expect(put.ok()).toBeTruthy();
  await page.context().setOffline(false);
  await expect(page.getByRole("status")).toHaveText("Live", { timeout: 20_000 });
  const rev = cell(block(page, "identity-page.pdf"), "revision");
  await expect(rev).toContainText("C", { timeout: 20_000 });
  await expect(rev).toContainText("Set by you");

  // Re-dropping stored bytes returns the existing filing.
  await dropFile(page, "identity-page.pdf", fixture("identity-page.pdf"), "application/pdf");
  // The upload row folds into the existing filing rather than adding one.
  await expect(block(page, "identity-page.pdf")).toHaveCount(1);
  await expect(block(page, "identity-page.pdf").locator(".tb-state")).toHaveText("Already filed in this project");

  // Another org cannot open the project or its documents.
  const other = await signIn(browser, "b");
  await expect(other.getByText("No projects yet.")).toBeVisible();
  await other.goto(projectUrl);
  await expect(other.getByText("This project isn't available.")).toBeVisible();
  const list = await other.request.get(`/api/projects/${projectId}/documents`);
  expect(list.status()).toBe(404);
  const doc = await other.request.get(`/api/documents/${docId}`);
  expect(doc.status()).toBe(404);
  const steal = await other.request.put(`/api/documents/${docId}/fields/title`, {
    data: { value: "stolen" },
    headers: { Origin: new URL(projectUrl).origin },
  });
  expect(steal.status()).toBe(404);
});
