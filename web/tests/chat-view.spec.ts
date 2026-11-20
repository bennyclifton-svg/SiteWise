import { expect, test, type Page } from "@playwright/test";
import { dropTender, freshOwner, headers, modelSwitch, tenderPackage } from "./tender-helpers";

// Chat that moves the screen (docs/plans/2026-11-18-chat-agency.md, Task 2),
// against the stand-in model in web/tests/e2eserver/chat.go: "screen DO
// TABLE NAME, NAME" is one view call on the rows the outline names.

async function say(page: Page, text: string): Promise<void> {
  const input = page.getByRole("textbox", { name: "Tell SiteWise what to change, or ask about this project" });
  await input.fill(text);
  await input.press("Enter");
  await expect(page.getByRole("region", { name: "Chat" }).getByRole("button", { name: "Send" })).toBeVisible();
}

async function newProject(page: Page, name: string): Promise<string> {
  const created = await page.request.post("/api/projects", { headers, data: { name } });
  expect(created.ok()).toBeTruthy();
  return (await created.json()).id as string;
}

test("cost lines: select, act on these, show only, clear, Escape, undo that", async ({ page }) => {
  await freshOwner(page, "chat-view-costs");
  await modelSwitch(page, true);
  const id = await newProject(page, "Chat moves cost lines");
  const ops = await page.request.post(`/api/projects/${id}/ops`, { headers, data: { area: "cost_plan", ops: [{ op: "cost_plan.add_line", fields: { labels: ["Footings", "Slab", "Roof"], section: "construction", system: "structure" } }] } });
  expect(ops.ok()).toBeTruthy();
  await page.goto(`/projects/${id}?view=costs`);
  const chat = page.getByRole("region", { name: "Chat" });
  const row = (label: string) => page.locator("tr.cost-row", { hasText: label });
  await expect(row("Roof")).toBeVisible();
  // A new plan holds starter lines of its own; only these three are ours.
  const all = await page.locator("tr.cost-row").count();
  const excluded = async () => {
    const plan = await (await page.request.get(`/api/projects/${id}/cost-plan`)).json();
    return (plan.items as { label: string; excluded: boolean }[]).filter(i => i.excluded && ["Footings", "Slab", "Roof"].includes(i.label)).map(i => i.label).sort();
  };

  await say(page, "screen select cost_lines Slab, Roof");
  await expect(chat).toContainText("Selected 2 cost lines.");
  await expect(row("Slab")).toHaveAttribute("aria-selected", "true");
  await expect(row("Roof")).toHaveAttribute("aria-selected", "true");
  await expect(row("Footings")).toHaveAttribute("aria-selected", "false");

  // "These" is what the screen has selected; nothing else changes.
  await say(page, "Exclude these");
  await expect(chat.locator(".chat-card").last()).toContainText("Changed · Cost Plan");
  await expect.poll(excluded).toEqual(["Roof", "Slab"]);

  await say(page, "screen only cost_lines Footings");
  await expect(page.locator(".reg-filter")).toContainText("Showing 1 cost line chosen in chat.");
  await expect(page.locator("tr.cost-row")).toHaveCount(1);
  await page.locator(".reg-filter").getByRole("button", { name: "Clear" }).click();
  await expect(page.locator("tr.cost-row")).toHaveCount(all);

  await say(page, "screen only cost_lines Slab");
  await expect(page.locator("tr.cost-row")).toHaveCount(1);
  await page.locator("table.cost-grid").focus();
  await page.keyboard.press("Escape");
  await expect(page.locator("tr.cost-row")).toHaveCount(all);
  await expect(page.locator(".reg-filter")).toHaveCount(0);

  // "Undo that" is answered without the model and puts the lines back.
  await say(page, "undo that");
  await expect(chat).toContainText("Undone:");
  await expect.poll(excluded).toEqual([]);
});

test("named filters: the cost lines with no budget, and the Clear line names the filter", async ({ page }) => {
  await freshOwner(page, "chat-view-filter");
  await modelSwitch(page, true);
  const id = await newProject(page, "Chat filters cost lines");
  const ops = await page.request.post(`/api/projects/${id}/ops`, { headers, data: { area: "cost_plan", ops: [
    { op: "cost_plan.add_line", fields: { label: "Footings", section: "construction", system: "structure", amount: "1000" } },
    { op: "cost_plan.add_line", fields: { labels: ["Slab", "Roof"], section: "construction", system: "structure" } },
  ] } });
  expect(ops.ok()).toBeTruthy();
  await page.goto(`/projects/${id}?view=costs`);
  const chat = page.getByRole("region", { name: "Chat" });
  const row = (label: string) => page.locator("tr.cost-row", { hasText: label });
  await expect(row("Footings")).toBeVisible();
  const all = await page.locator("tr.cost-row").count();

  await say(page, "Show the cost lines with no budget");
  const line = page.locator(".reg-filter");
  await expect(line).toContainText(/^Showing \d+ cost lines? with no budget\. · Clear$/);
  await expect(line).not.toContainText("chosen in chat");
  await expect(chat).toContainText(/Showing \d+ cost lines? with no budget\./);
  await expect(row("Slab")).toBeVisible();
  await expect(row("Roof")).toBeVisible();
  await expect(row("Footings")).toHaveCount(0);
  // The count on the line is the rows left on screen.
  const shown = Number((await line.textContent())?.match(/Showing (\d+)/)?.[1]);
  await expect(page.locator("tr.cost-row")).toHaveCount(shown);
  await line.getByRole("button", { name: "Clear" }).click();
  await expect(page.locator("tr.cost-row")).toHaveCount(all);

  // Rows picked by id still say they were chosen in chat.
  await say(page, "screen only cost_lines Footings");
  await expect(line).toContainText("Showing 1 cost line chosen in chat.");

  // A filter that matches nothing shows the table and says so.
  await say(page, "filter select cost_lines over_budget");
  await expect(chat).toContainText("No cost line has claimed more than its budget.");
});

test("a table not on screen is shown first, then moved", async ({ page }) => {
  await freshOwner(page, "chat-view-program");
  await modelSwitch(page, true);
  const id = await newProject(page, "Chat moves the program");
  expect((await page.request.post(`/api/projects/${id}/program/prepare`, { headers, data: { surface: "program" } })).ok()).toBeTruthy();
  const program = await (await page.request.get(`/api/projects/${id}/program`)).json();
  const stages = (program.activities as { label: string; kind: string }[]).filter(a => a.kind === "stage").map(a => a.label);
  expect(stages.length).toBeGreaterThan(1);
  await page.goto(`/projects/${id}`);
  const chat = page.getByRole("region", { name: "Chat" });

  await say(page, `screen select program ${stages[1]}`);
  await expect(chat).toContainText("Selected 1 Program row.");
  await expect(page.getByRole("heading", { name: "Program", exact: true })).toBeVisible();
  await expect(page.locator(".program-head .pg-chip")).toContainText("1 selected");

  await say(page, `screen only program ${stages[0]}`);
  await expect(page.locator(".program .reg-filter")).toContainText("Showing 1 Program row chosen in chat.");
  await page.locator(".program .reg-filter").getByRole("button", { name: "Clear" }).click();
  await expect(page.locator(".program .reg-filter")).toHaveCount(0);
});

test("procurement rows and the award dialog: chat opens it, the PM presses Award", async ({ page }) => {
  await freshOwner(page, "chat-view-award");
  await modelSwitch(page, true);
  const p = await tenderPackage(page, "Chat opens the award");
  await dropTender(page, p, "Harbour Planning", 11, p.rows.map(r => [r.label, 1000] as [string, number]));
  await page.goto(`/projects/${p.project}`);
  const chat = page.getByRole("region", { name: "Chat" });

  await say(page, "screen select procurement Town Planner");
  await expect(chat).toContainText("Selected 1 package.");
  await expect(page.getByRole("row", { name: /^Town Planner/ })).toHaveAttribute("aria-selected", "true");
  await say(page, "screen only procurement Town Planner");
  await expect(page.locator(".procurement .reg-filter")).toContainText("Showing 1 package chosen in chat.");
  await say(page, "screen clear procurement");
  await expect(page.locator(".procurement .reg-filter")).toHaveCount(0);

  await say(page, "screen award Harbour Planning");
  await expect(chat).toContainText("Press Award there");
  const dialog = page.locator("dialog[open]");
  await expect(dialog).toBeVisible();
  // Nothing was awarded: the PM has not pressed anything.
  const table = await (await page.request.get(`${p.base}/prices`)).json();
  expect((table.columns as { submission: { status: string } }[]).every(c => c.submission.status !== "awarded")).toBeTruthy();
});

test("documents: select and open from the register", async ({ page }) => {
  await freshOwner(page, "chat-view-docs");
  await modelSwitch(page, true);
  const id = await newProject(page, "Chat moves documents");
  await page.goto(`/projects/${id}`);
  await page.getByTestId("file-input").setInputFiles([0, 1, 2].map(i => ({
    name: `Memo ${i}.md`, mimeType: "text/markdown", buffer: Buffer.from(`# Memo ${i}\nDocument number: M-10${i}\nRevision: A\n`),
  })));
  await expect(page.locator("tbody.reg-doc")).toHaveCount(3);
  await expect.poll(async () => (await (await page.request.get(`/api/projects/${id}/documents`)).json()).documents.every((d: { status: string }) => d.status !== "pending")).toBe(true);
  const chat = page.getByRole("region", { name: "Chat" });
  const titles = (await (await page.request.get(`/api/projects/${id}/documents`)).json()).documents as { id: string }[];
  expect(titles.length).toBe(3);

  await say(page, "screen select documents Memo 1, Memo 2");
  await expect(chat).toContainText("Selected 2 documents.");
  await expect.poll(() => page.locator("tbody.reg-doc[data-selected]").count()).toBe(2);
  await say(page, "screen only documents Memo 0");
  await expect(page.locator(".register .reg-filter")).toContainText("Showing 1 document chosen in chat.");
  await expect(page.locator("tbody.reg-doc")).toHaveCount(1);
  await page.locator(".register .reg-filter").getByRole("button", { name: "Clear" }).click();
  await expect(page.locator("tbody.reg-doc")).toHaveCount(3);
  // Open is the register's own Open: the document fills the middle panel.
  await say(page, "screen open documents Memo 2");
  await expect(page.getByRole("region", { name: "Document viewer" })).toHaveAttribute("data-filename", "Memo 2.md");
});

test("the project report opens with its issue form; chat issues nothing", async ({ page }) => {
  await freshOwner(page, "chat-view-report");
  await modelSwitch(page, true);
  const id = await newProject(page, "Chat opens the project report");
  const ops = await page.request.post(`/api/projects/${id}/ops`, { headers, data: { area: "reports", ops: [{ op: "reports.update", fields: {} }] } });
  expect(ops.ok(), await ops.text()).toBeTruthy();
  await page.goto(`/projects/${id}?view=costs`);
  const chat = page.getByRole("region", { name: "Chat" });
  await say(page, "screen report issue");
  await expect(chat).toContainText("Opened the project report's issue form.");
  await expect(page.getByRole("heading", { name: "Project report No. 1", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Issue saved report", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Freeze and issue PDF", exact: true })).toBeVisible();
  // The reader is in the middle; the right-hand panel shows the Register,
  // whose band holds Project report.
  await expect(page.getByRole("tablist", { name: "Right panel" }).getByRole("tab", { name: "Register", exact: true })).toHaveAttribute("aria-selected", "true");
  const list = await (await page.request.get(`/api/projects/${id}/reports`)).json();
  const view = await (await page.request.get(`/api/reports/${list[0].id}`)).json();
  expect(view.issues).toHaveLength(0);
});

// The register in the right-hand panel (20 November 2026): show switches
// the panel and leaves the middle view; open puts the row's sheet in the
// middle; select acts on the list; Send for an RFI opens its sheet filled
// in, and chat sends nothing.
test("register: show switches the panel, open puts the sheet in the middle, Send for an RFI fills the sheet", async ({ page }) => {
  await freshOwner(page, "chat-view-register");
  await modelSwitch(page, true);
  const id = await newProject(page, "Chat moves the register");
  const add = await page.request.post(`/api/projects/${id}/ops`, { headers, data: { area: "delivery", ops: [
    { op: "delivery.add", fields: { kind: "rfi", title: "Slab edge set-down", raised_on: "2026-11-20" } },
    { op: "delivery.add", fields: { kind: "rfi", title: "Stair balustrade", raised_on: "2026-11-20" } }] } });
  expect(add.ok(), await add.text()).toBeTruthy();
  const view = await (await page.request.get(`/api/projects/${id}/delivery`)).json();
  const slab = view.items.find((i: { title: string }) => i.title === "Slab edge set-down").id as string;
  const stair = view.items.find((i: { title: string }) => i.title === "Stair balustrade").id as string;
  await page.goto(`/projects/${id}?view=costs`);
  const chat = page.getByRole("region", { name: "Chat" });
  const tab = (name: string) => page.getByRole("tablist", { name: "Right panel" }).getByRole("tab", { name, exact: true });
  await tab("Documents").click();

  await say(page, "screen show register");
  await expect(chat).toContainText("Showed the register rows.");
  await expect(tab("Register")).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("button", { name: "Costs", exact: true })).toHaveAttribute("aria-current", "page");

  await say(page, `screen select register ${stair.slice(0, 8)}`);
  await expect(page.locator(".register-panel tr", { hasText: "Stair balustrade" })).toHaveAttribute("aria-selected", "true");

  await say(page, `screen open register ${slab.slice(0, 8)}`);
  await expect(page.locator(".register-sheet").getByRole("heading", { name: "RFI-001 · Slab edge set-down" })).toBeVisible();

  // Back to the documents; Send for the other RFI opens its sheet with the
  // fields in, and nothing is sent.
  await tab("Documents").click();
  await say(page, "screen send rfi-002 to site@builder.test");
  await expect(chat).toContainText("Opened RFI-002 to send to site@builder.test");
  const sheet = page.locator(".register-sheet");
  await expect(sheet.getByRole("heading", { name: "RFI-002 · Stair balustrade" })).toBeVisible();
  await expect(sheet.getByLabel("To", { exact: true })).toHaveValue("site@builder.test");
  await expect(sheet.getByLabel("Subject", { exact: true })).toHaveValue("RFI-002 from chat");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  const sent = (await (await page.request.get("/__e2e/mail-sent")).json()).sent ?? [];
  expect(sent.some((m: { subject: string }) => m.subject.includes("RFI-002"))).toBeFalsy();
});
