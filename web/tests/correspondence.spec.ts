import { createHmac, randomUUID } from "node:crypto";
import { expect, test, type APIRequestContext } from "@playwright/test";

// Email intake Task 9 and correspondence Task 6, on the e2e server's test
// signing key and mail sink: nothing reaches Mailgun. Posts are hand-written
// in Mailgun's documented forward() shape with dummy content only.

const origin = { Origin: "http://127.0.0.1:4173" };
const KEY = "e2e-test-signing-key";

interface Raw { From: string; To: string; Subject: string; Body: string; MessageID: string; InReplyTo?: string }

// mailgun posts as Mailgun's raw-MIME route does: the message in body-mime,
// DKIM-signed by its From domain through the e2e server's stub DNS.
async function mailgun(request: APIRequestContext, recipient: string, sender: string, raw: Raw) {
  const body = await (await request.post("/__e2e/mail-raw", { data: raw })).text();
  const timestamp = String(Math.floor(Date.now() / 1000));
  const token = randomUUID();
  const signature = createHmac("sha256", KEY).update(timestamp + token).digest("hex");
  return request.post("/api/mail/inbound/mime", { multipart: { recipient, sender, from: raw.From, subject: raw.Subject, "body-mime": body, timestamp, token, signature } });
}

test("the project's address, a held message accepted, and a transmittal sent and replied to", async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  // A free account sends after a week (review MEDIUM-3).
  await page.request.post("/__e2e/age-accounts");
  await page.getByLabel("New project").fill("Mail house");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;

  // The address line: random, never the project's name, with Copy.
  const line = page.locator(".mail-address");
  await expect(line).toContainText("Email documents to");
  const address = (await line.locator("code").innerText()).trim();
  expect(address).toMatch(/^p-[0-9a-f]{32}@in\.e2e\.test$/);
  expect(address.toLowerCase()).not.toContain("mail");
  // No toggle until the project has a message (rule 5).
  await expect(page.getByRole("tab", { name: "Messages" })).toHaveCount(0);

  // Someone outside the team: held, with its reason in plain words.
  const held = await mailgun(page.request, address, "someone@outside.test", {
    From: "Someone <someone@outside.test>", To: address, Subject: "Wall detail", Body: "Please see the detail.", MessageID: `<${randomUUID()}@outside.test>`,
  });
  expect(held.status()).toBe(200);
  await expect(line).toContainText("1 held", { timeout: 10_000 });
  await page.getByRole("button", { name: "1 held" }).click();
  const messages = page.getByRole("table", { name: /^Messages/ });
  const wall = messages.locator("tbody").filter({ hasText: "Wall detail" });
  await expect(wall).toContainText("Not from your team");
  await wall.locator("tr.reg-line").click();
  await expect(wall.locator(".mail-text")).toHaveText("Please see the detail.");
  await wall.getByRole("button", { name: "Accept" }).click();
  await expect(wall).not.toContainText("Not from your team", { timeout: 10_000 });

  // Back to documents: the accepted body landed as a document.
  await page.getByRole("tablist", { name: "Repository" }).getByRole("tab", { name: "Documents" }).click();
  await expect(page.locator(".reg-doc").filter({ hasText: "Wall detail" })).toHaveCount(1, { timeout: 15_000 });

  // Send… from the documents' right-click menu: the sink gets it, the record shows Out.
  await page.locator(".reg-doc").filter({ hasText: "Wall detail" }).locator("tr.reg-line").click({ button: "right" });
  await page.getByRole("menuitem", { name: "Send…" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("To").fill("engineer@consult.test");
  await dialog.getByLabel("Subject").fill("TR Wall detail");
  await dialog.getByRole("button", { name: "Send" }).click();
  await expect(dialog).toHaveCount(0);
  await expect.poll(async () => (await (await page.request.get("/__e2e/mail-sent")).json()).sent?.length ?? 0, { timeout: 10_000 }).toBeGreaterThan(0);
  const sent = (await (await page.request.get("/__e2e/mail-sent")).json()).sent as { to: string[]; subject: string }[];
  expect(sent.some(m => m.subject.includes("TR Wall detail") && m.to.includes("engineer@consult.test"))).toBeTruthy();
  await page.getByRole("tab", { name: "Messages" }).click();
  const out = messages.locator("tbody").filter({ hasText: "TR-001" });
  await expect(out).toContainText("Out");
  await expect(out).toContainText("TR-001");

  // The engineer replies: filed (A1), threaded under the transmittal, which says Replied.
  const record = await (await page.request.get(`/api/projects/${project}/mail`)).json();
  const tr = record.messages.find((m: { subject: string }) => m.subject === "TR Wall detail");
  const one = await (await page.request.get(`/api/projects/${project}/mail/${tr.id}`)).json();
  expect(one.direction).toBe("out");
  const reply = await mailgun(page.request, address, "engineer@consult.test", {
    From: "engineer@consult.test", To: address, Subject: "Re: TR Wall detail", Body: "Received.", MessageID: `<${randomUUID()}@consult.test>`,
    InReplyTo: await messageID(page.request, project, tr.id),
  });
  expect(reply.status()).toBe(200);
  await expect(out).toContainText("Replied", { timeout: 10_000 });
  await expect(messages.locator("tbody").filter({ hasText: "Re: TR Wall detail" })).toHaveAttribute("data-reply", "true");

  // Frozen: there is no Delete on a message, and discard refuses a filed one.
  const discard = await page.request.post(`/api/projects/${project}/mail/${tr.id}/discard`, { headers: origin });
  expect(discard.status()).toBe(409);
});

// The Message-ID is not in the screen's JSON; the test reads it the way a
// mail client would see it, from the sink's copy of the sent message.
async function messageID(request: APIRequestContext, project: string, id: string): Promise<string> {
  void project; void id;
  const res = await request.get("/__e2e/mail-sent?raw=message-id");
  const body = await res.json();
  return body.message_id ?? "";
}

// The register panel (20 November 2026): an RFI goes out from its sheet
// through the one send route, with its To, its question as the note and
// its transmittal; the engineer's reply joins its history and is used as
// the answer. The screenshots are the evidence note's.
test("an RFI sent from its sheet, its reply in the history, used as the answer", async ({ page }) => {
  await page.setViewportSize({ width: 1650, height: 920 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.request.post("/__e2e/age-accounts");
  await page.getByLabel("New project").fill("Seven Hills house");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const address = (await page.locator(".mail-address code").innerText()).trim();
  const add = await page.request.post(`/api/projects/${project}/ops`, { headers: origin, data: { area: "delivery", ops: [
    { op: "delivery.add", fields: { kind: "rfi", title: "Basement wall waterproofing membrane", raised_on: "2026-11-20", owner_text: "MBM Structural", question: "Please confirm the membrane to the basement walls and its laps at the footing." } },
    { op: "delivery.add", fields: { kind: "rfi", title: "Stair balustrade height at landing", raised_on: "2026-11-20" } }] } });
  expect(add.ok(), await add.text()).toBeTruthy();
  await page.reload();

  await page.getByRole("tablist", { name: "Right panel" }).getByRole("tab", { name: "Register" }).click();
  await page.locator(".register-panel tr", { hasText: "Basement wall waterproofing" }).locator('td[data-field="title"]').click();
  const sheet = page.locator(".register-sheet");
  await sheet.getByLabel("To", { exact: true }).fill("eng@mbm.test");
  await sheet.getByLabel("Question", { exact: true }).click();
  // A seeded document has no stored bytes to attach, so this one goes with
  // none; the transmittal from the Documents side is delivery.spec's.
  const send = sheet.getByRole("button", { name: "Send", exact: true });
  await expect(send).toHaveAttribute("title", "Send to 1 address with 0 documents");
  await send.click();
  await expect(sheet.locator(".register-send-what")).toHaveText("Sent as TR-001. It is in Messages.");
  await expect.poll(async () => ((await (await page.request.get("/__e2e/mail-sent")).json()).sent ?? []).some((m: { subject: string; to: string[] }) => m.subject === "RFI-001 Basement wall waterproofing membrane" && m.to.includes("eng@mbm.test")), { timeout: 10_000 }).toBeTruthy();
  const history = sheet.getByRole("list", { name: "History" });
  await expect(history.locator('li[data-kind="sent"]')).toContainText("Sent to eng@mbm.test · 0 documents · TR-001");

  // The engineer replies in the thread: it files and joins the history.
  const record = await (await page.request.get(`/api/projects/${project}/mail`)).json();
  const tr = record.messages.find((m: { subject: string }) => m.subject.startsWith("RFI-001"));
  const reply = await mailgun(page.request, address, "eng@mbm.test", {
    From: "MBM Structural <eng@mbm.test>", To: address, Subject: "Re: RFI-001 Basement wall waterproofing membrane",
    Body: "Use the sheet membrane, laps 150 mm.\nTurn it 300 mm up the wall.\n\nOn Thu, PM wrote:\n> Please confirm", MessageID: `<${randomUUID()}@mbm.test>`,
    InReplyTo: await messageID(page.request, project, tr.id),
  });
  expect(reply.status()).toBe(200);
  await expect(async () => {
    await page.reload();
    await page.locator(".register-panel tr", { hasText: "Basement wall waterproofing" }).locator('td[data-field="title"]').click();
    await expect(history.locator('li[data-kind="reply"]')).toContainText("Reply from MBM Structural", { timeout: 1_000 });
  }).toPass({ timeout: 15_000 });
  await expect(history.locator('li[data-kind="reply"] .register-quote')).toHaveText("Use the sheet membrane, laps 150 mm.\nTurn it 300 mm up the wall.");
  await page.screenshot({ path: "../docs/evidence/img/2026-11-20-register-panel-register.png" });
  await page.getByRole("tablist", { name: "Right panel" }).getByRole("tab", { name: "Documents" }).click();
  await history.scrollIntoViewIfNeeded();
  await page.screenshot({ path: "../docs/evidence/img/2026-11-20-register-panel-documents.png" });

  // Use as the answer: the reply is the answer and the RFI is with the PM.
  await history.locator('li[data-kind="reply"]').getByRole("button", { name: "Use as the answer" }).click();
  await expect(history.locator('li[data-kind="reply"]')).toHaveAttribute("data-tone", "answer");
  await expect(sheet.locator(".register-sheet-head .register-status")).toHaveText("Answered · with you");
  await expect(history.locator("li").last()).toContainText("→ Answered");
  await history.scrollIntoViewIfNeeded();
  await page.screenshot({ path: "../docs/evidence/img/2026-11-20-register-panel-answered.png" });
  // "Open the reply" opens it in the repository's Messages.
  await history.locator('li[data-kind="reply"]').getByRole("button", { name: "Open the reply" }).click();
  await expect(page.getByRole("tablist", { name: "Repository" }).getByRole("tab", { name: "Messages" })).toHaveAttribute("aria-selected", "true");
});
