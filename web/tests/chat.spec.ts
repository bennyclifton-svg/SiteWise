import { expect, test, type Page } from "@playwright/test";
import { freshOwner, headers, modelSwitch } from "./tender-helpers";

// The chat strip at the foot of the middle panel, against a stand-in model
// (web/tests/e2eserver/chat.go). No text leaves the test server.

async function project(page: Page, name: string): Promise<string> {
  const created = await page.request.post("/api/projects", { headers, data: { name } });
  expect(created.ok()).toBeTruthy();
  return (await created.json()).id as string;
}

async function say(page: Page, text: string): Promise<void> {
  const input = page.getByRole("textbox", { name: "Tell SiteWise what to change, or ask about this project" });
  await input.fill(text);
  await input.press("Enter");
}

test("a typed command applies, the screen behind updates, and Undo restores it", async ({ page }) => {
  await freshOwner(page, "chat");
  await modelSwitch(page, true);
  const id = await project(page, "Chat strip");
  await page.goto(`/projects/${id}?view=costs`);
  const chat = page.getByRole("region", { name: "Chat" });
  await expect(chat).toBeVisible();

  await say(page, "Add a joinery line to Fitout, $45,000");
  const card = chat.locator(".chat-card").last();
  await expect(card).toContainText("Changed · Cost Plan");
  await expect(card).toContainText('Added "Joinery" to Construction');
  await expect(card).toContainText("$45,000");
  // The Cost Plan behind the strip shows the line without a reload.
  await expect(page.locator(".costs")).toContainText("Joinery");

  await card.getByRole("button", { name: "Undo" }).click();
  await expect(card).toContainText("Undone");
  await expect(page.locator(".costs")).not.toContainText("Joinery");
  const plan = await (await page.request.get(`/api/projects/${id}/cost-plan`)).json();
  expect((plan.items ?? []).filter((i: { label: string }) => i.label === "Joinery")).toHaveLength(0);
});

test("the strip expands over the panel and collapses back", async ({ page }) => {
  await freshOwner(page, "chat-expand");
  await modelSwitch(page, true);
  const id = await project(page, "Chat expand");
  await page.goto(`/projects/${id}`);
  const chat = page.getByRole("region", { name: "Chat" });
  await say(page, "Change it");
  await expect(chat).toContainText("Which line do you mean?");
  await say(page, "Add a joinery line, $45,000");
  await expect(chat.locator(".chat-card")).toContainText("Joinery");
  // Collapsed, only the latest turn shows; expanded, the whole conversation.
  await expect(chat.locator(".chat-turn")).toHaveCount(1);
  await page.getByRole("button", { name: "Expand chat" }).click();
  await expect(chat.locator(".chat-turn")).toHaveCount(2);
  await expect(chat).toContainText("Which line do you mean?");
  await page.getByRole("button", { name: "Collapse chat" }).click();
  await expect(chat.locator(".chat-turn")).toHaveCount(1);
  // The conversation is still there after a reload.
  await page.reload();
  await page.getByRole("button", { name: "Expand chat" }).click();
  await expect(page.getByRole("region", { name: "Chat" }).locator(".chat-turn")).toHaveCount(2);
});

test("a change the PM vouches for waits for Apply", async ({ page }) => {
  await freshOwner(page, "chat-click");
  await modelSwitch(page, true);
  const id = await project(page, "Chat click");
  await page.goto(`/projects/${id}?view=costs`);
  const chat = page.getByRole("region", { name: "Chat" });
  await say(page, "Add a joinery line, $45,000");
  await expect(chat.locator(".chat-card")).toContainText("Joinery");
  await say(page, "Freeze the cost plan");
  const card = chat.locator(".chat-card").last();
  await expect(card).toContainText("Ready to apply");
  await expect(card).toContainText("yours to confirm");
  let plan = await (await page.request.get(`/api/projects/${id}/cost-plan`)).json();
  expect(plan.baseline_version_id).toBe("");
  await card.getByRole("button", { name: "Apply" }).click();
  await expect(card).toContainText("Froze the cost plan");
  plan = await (await page.request.get(`/api/projects/${id}/cost-plan`)).json();
  expect(plan.baseline_version_id).not.toBe("");
});

test("with the model down or the switch off, the workbench works as before", async ({ page }) => {
  await freshOwner(page, "chat-down");
  const id = await project(page, "Chat down");
  await page.goto(`/projects/${id}?view=costs`);
  const chat = page.getByRole("region", { name: "Chat" });
  // The switch is off: the strip says where to turn it on.
  await expect(chat.getByRole("textbox")).toBeDisabled();
  await expect(chat.getByRole("textbox")).toHaveAttribute("placeholder", /owner can switch it on under Account/);
  await expect(chat.getByRole("link", { name: "Open Account" })).toBeVisible();

  await modelSwitch(page, true);
  await page.reload();
  await say(page, "chat is down");
  await expect(chat).toContainText("Chat is unavailable. Nothing was changed.");
  // The cost plan behind the strip is still there and still answers.
  await expect(page.locator(".costs")).toBeVisible();
  const plan = await page.request.get(`/api/projects/${id}/cost-plan`);
  expect(plan.ok()).toBeTruthy();
});
