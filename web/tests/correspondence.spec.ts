import { createHmac, randomUUID } from "node:crypto";
import { expect, test, type APIRequestContext } from "@playwright/test";

// Email intake Task 9 and correspondence Task 6, on the e2e server's test
// signing key and mail sink: nothing reaches Mailgun. Posts are hand-written
// in Mailgun's documented forward() shape with dummy content only.

const origin = { Origin: "http://127.0.0.1:4173" };
const KEY = "e2e-test-signing-key";

async function mailgun(request: APIRequestContext, fields: Record<string, string>) {
  const timestamp = String(Math.floor(Date.now() / 1000));
  const token = randomUUID();
  const signature = createHmac("sha256", KEY).update(timestamp + token).digest("hex");
  return request.post("/api/mail/inbound", { form: { ...fields, timestamp, token, signature } });
}

test("the project's address, a held message accepted, and a transmittal sent and replied to", async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
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
  const held = await mailgun(page.request, {
    recipient: address, sender: "someone@outside.test", from: "Someone <someone@outside.test>", subject: "Wall detail",
    "body-plain": "Please see the detail.", "message-headers": JSON.stringify([["Message-Id", `<${randomUUID()}@outside.test>`], ["X-Mailgun-Spf", "Pass"]]),
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
  await page.getByRole("tab", { name: "Documents" }).click();
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
  const reply = await mailgun(page.request, {
    recipient: address, sender: "engineer@consult.test", from: "engineer@consult.test", subject: "Re: TR Wall detail", "body-plain": "Received.",
    "message-headers": JSON.stringify([["Message-Id", `<${randomUUID()}@consult.test>`], ["In-Reply-To", await messageID(page.request, project, tr.id)], ["X-Mailgun-Spf", "Pass"]]),
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
