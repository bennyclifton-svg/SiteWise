import { expect, test } from "@playwright/test";

test("visual suggestions need an explicit action and preserve edited fields", async ({ page }) => {
  const invite = await page.request.post("/__e2e/invite?org=a");
  const { token } = await invite.json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Visual review check");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("heading", { name: "Visual review check" })).toBeVisible();
  const project = page.url().split("/projects/")[1];
  const doc = { id: "33333333-3333-4333-8333-000000000997", project_id: project, visual_title: "Access assessment report", filename: "Consultant report.pdf", status: "filed", created_at: "2026-10-19T00:00:00Z", fields: [{field: "revision", value: "Owner revision", decided_by: "user", band: "green"}] };
  await page.route(`**/api/projects/${project}/documents`, r => r.fulfill({json: {cursor: 0, project: {id: project, name: "Visual review check"}, documents: [doc]}}));
  await page.route(`**/api/documents/${doc.id}`, r => r.fulfill({json: doc}));
  await page.route(`**/api/documents/${doc.id}/visual-review`, r => r.fulfill({json: {status: "ready", result: {pages: 3, suggestions: {title: {value: "Access assessment report", page: 1}, revision: {value: "B", page: 2}}, images: []}}}));
  let writes = 0;
  await page.route(`**/api/documents/${doc.id}/fields/title`, async r => {
    writes++; expect(r.request().postDataJSON()).toEqual({value: "Access assessment report"});
    doc.fields.push({field: "title",value: "Access assessment report",decided_by: "user",band: "green"});
    await r.fulfill({json: doc});
  });
  await page.addInitScript(() => {
    const Native = window.EventSource;
    window.EventSource = class extends Native {
      constructor(url: string | URL, options?: EventSourceInit) {
        super(url, options);
        (window as unknown as { reviewStream: EventSource }).reviewStream = this;
      }
    };
  });
  doc.visual_title = "";
  await page.reload();
  await expect(page.locator("tbody.reg-doc .reg-line .reg-title")).not.toContainText("Access assessment report");
  doc.visual_title = "Access assessment report";
  await page.waitForFunction(() => Boolean((window as unknown as { reviewStream?: EventSource }).reviewStream));
  await page.evaluate(id => {
    (window as unknown as { reviewStream: EventSource }).reviewStream.dispatchEvent(new MessageEvent("visual", {
      data: JSON.stringify({id: 1000000000, kind: "visual", document_id: id, payload: {}}),
    }));
  }, doc.id);
  await expect(page.locator("tbody.reg-doc .reg-line .reg-title")).toContainText("Access assessment report");
  await expect(page.locator("tbody.reg-doc .reg-line .reg-title")).toContainText("Suggested · review");
  await page.getByRole("button", { name: /^Open document:/ }).click();
  await page.getByRole("button", {name: "Filing details", exact: true}).click();
  const panel = page.getByRole("region", {name: "Image-assisted filing review"});
  await expect(panel.getByText("Access assessment report", {exact: true})).toBeVisible();
  expect(writes).toBe(0);
  await expect(panel.getByRole("button", {name: "Use revision"})).toHaveCount(0);
  await page.getByRole("button", {name: "Close document", exact: true}).click();
  await page.getByRole("button", {name: "Accept suggested title", exact: true}).click();
  await expect(page.getByRole("button", {name: "Accept suggested title", exact: true})).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^Open document:/ })).toHaveAttribute("aria-expanded", "false");
  await page.getByRole("button", { name: /^Open document:/ }).click();
  await page.getByRole("button", {name: "Filing details", exact: true}).click();
  await expect(panel.getByText("Matches saved value")).toBeVisible();
  expect(writes).toBe(1);
  await page.setViewportSize({width: 390,height: 844});
  const bounds = await panel.boundingBox();
  expect(bounds!.x+bounds!.width).toBeLessThanOrEqual(391);
  doc.status = "split";
  doc.fields = [];
  await page.reload();
  await expect(page.locator("tbody.reg-doc .reg-line .reg-title")).toContainText("Consultant report.pdf");
  await expect(page.locator("tbody.reg-doc .reg-line .reg-title")).toContainText("Original drawing set");
  await expect(page.locator("tbody.reg-doc .reg-line .reg-title")).not.toContainText("Title needs review");
});

test("Petersham access title arrives without a completion event or refresh", async ({page}) => {
  const {token} = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Petersham access recovery");
  await page.getByRole("button", {name:"Create project"}).click();
  await expect(page.getByRole("heading", {name:"Petersham access recovery"})).toBeVisible();
  const project = page.url().split("/projects/")[1];
  const doc = {id:"33333333-3333-4333-8333-000000000998", project_id:project, filename:"22372_Vista Access_CC_Issue A.pdf",status:"filed",created_at:"2026-10-19T00:00:00Z",visual_status:"queued",visual_title:"",fields:[{field:"title",value:"",band:"blank",decided_by:"jev"}]};
  let reads=0, writes=0;
  await page.route(`**/api/projects/${project}/documents`, route => {reads++; return route.fulfill({json:{cursor:0,project:{id:project,name:"Petersham access recovery"},documents:[doc]}});});
  await page.route("**/api/events?*", route => route.fulfill({status:200,contentType:"text/event-stream",body:": no completion event\n\n"}));
  await page.route(`**/api/documents/${doc.id}/fields/title`, route => {
    writes++; expect(route.request().postDataJSON()).toEqual({value:"Construction Certificate ACCESS REPORT"});
    doc.fields=[{field:"title",value:"Construction Certificate ACCESS REPORT",band:"green",decided_by:"user"}];
    return route.fulfill({json:doc});
  });
  await page.reload();
  await expect(page.getByText("Waiting for image reading…", {exact:true})).toBeVisible();
  doc.visual_status="reading";
  await expect(page.getByText("Reading document images…", {exact:true})).toBeVisible();
  await expect(page.locator(".reg-doc")).toHaveAttribute("data-activity", "active");
  await expect(page.locator(".reg-line .activity-icon svg")).toHaveCSS("animation-name", "activity-turn");
  doc.visual_status="ready";doc.visual_title="Construction Certificate ACCESS REPORT";
  await expect(page.locator(".reg-line .reg-title")).toContainText(doc.visual_title);
  expect(reads).toBeGreaterThan(1);expect(writes).toBe(0);
  await page.getByRole("button",{name:"Accept suggested title",exact:true}).click();
  await expect(page.locator(".reg-line .reg-title")).toHaveAttribute("data-user","true");
  await expect(page.getByRole("button",{name:"Accept suggested title",exact:true})).toHaveCount(0);
  expect(writes).toBe(1);
});

