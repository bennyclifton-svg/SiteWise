import { expect, test, type Page } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

async function signIn(page: Page, name: string) {
  const invite = await page.request.post("/__e2e/invite?org=a");
  const { token } = await invite.json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  return new URL(page.url()).pathname.split("/").pop()!;
}

async function refreshDraft(page: Page) {
  const toolbar = page.locator(".report-toolbar");
  // The version is kept off screen (owner, 7 Nov 2026) but stays on the toolbar.
  const version = await toolbar.getAttribute("data-version");
  expect(version).not.toBeNull();
  await page.getByRole("button", { name: "Update", exact: true }).click();
  // Existing controls can still show the previous value while the new draft
  // loads. Wait for its version before testing persistence or another edit.
  await expect(toolbar).toHaveAttribute("data-version", String(Number(version) + 1));
}

test("one RFP report: page one edited where it prints, checks beside their values, issue at the end", async ({ page }) => {
  const project = await signIn(page, "RFP editor regression");
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  const part = profile.parts[0].id;
  for (const [key, value] of [["hdr.work_type", "extend"], ["hdr.cond.planning", "da"], ["hdr.cond.procurement_route", "traditional"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin, data: { part_id: part, value } });
    expect(r.ok(), `${key}: ${await r.text()}`).toBeTruthy();
  }
  for (const data of [{ kind: "services", title: "Architect", discipline_id: "consultant.architect" }, { kind: "services", title: "Civil" }]) {
    const r = await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data });
    expect(r.ok()).toBeTruthy();
  }
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Procurement", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Create RFP for Architect", exact: true }).click();
  await expect(page.getByRole("button", { name: "Update", exact: true })).toBeVisible();

  // One report: page one is the project, always open in the Profile card's
  // columns; no decisions banner, no stale block.
  await expect(page.getByRole("complementary", { name: "Finish this RFP" })).toHaveCount(0);
  // Scope by stage: the stage table sits under 2. Scope of services.
  await expect(page.locator(".rfp-stage-scope")).toHaveCount(1);
  await expect(page.getByRole("heading", { name: /Scope by stage/ })).toHaveCount(0);
  await expect(page.getByRole("region", { name: "Decide before issue" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Edit 1. Project", exact: true })).toHaveCount(0);
  const pageOne = page.locator(".rfp-page-one");
  for (const column of ["The building", "Site and planning", "Delivery"]) await expect(pageOne.getByRole("group", { name: column, exact: true })).toBeVisible();
  // Project conditions are set in Profile; only one that blocks issue stays here.
  await expect(pageOne.locator('.rfp-conditions [data-control^="fact:flag."]:not([data-blocked])')).toHaveCount(0);

  // A blocking check sits beside what it concerns; the Issue section counts them.
  const issue = page.getByRole("region", { name: "Issue", exact: true });
  await expect(issue.getByRole("button", { name: "Issue report", exact: true })).toBeDisabled();
  await expect(page.getByRole("group", { name: "Proposal dates" }).getByLabel("Release date", { exact: true })).toBeVisible();
  await expect(pageOne.locator('[data-control="fact:client"][data-blocked]')).toBeVisible();

  // A blank client quantity is marked under section 4 and opens its control.
  const allowances = page.locator(".report-section").filter({ has: page.getByRole("heading", { name: "4. Allowances", exact: true }) });
  await allowances.locator(".rfp-markers li").filter({ hasText: "Client presentations" }).getByRole("button", { name: "Show", exact: true }).click();
  await allowances.getByLabel("Client presentations", { exact: true }).fill("3");
  await allowances.getByLabel("Client presentations", { exact: true }).press("Enter");
  await expect(page.locator(".rfp-draft").getByRole("cell", { name: "Client presentations", exact: true })).toBeVisible();

  // Project facts are edited where they print on page one and saved to Profile.
  await pageOne.getByRole("button", { name: /^Client: .*Change$/ }).click();
  await pageOne.getByLabel("Client", { exact: true }).fill("Inner West Council");
  await pageOne.getByLabel("Client", { exact: true }).press("Enter");
  await expect(pageOne.locator(".rfp-source")).toHaveCount(0);
  await expect(pageOne.locator('[data-control="fact:client"]')).toContainText("Inner West Council");
  await expect(pageOne.locator('[data-control="fact:client"][data-blocked]')).toHaveCount(0);
  // An unset row waits behind "more", as on the Profile card.
  const site = pageOne.getByRole("group", { name: "Site and planning", exact: true });
  await expect(site.getByRole("button", { name: /^\+ \d+ more$/ })).toHaveCount(0);
  await site.getByRole("button", { name: "Zone: —. Change", exact: true }).click();
  await site.getByLabel("Zone", { exact: true }).fill("R2");
  await site.getByLabel("Zone", { exact: true }).press("Enter");
  await expect(pageOne.locator('[data-control="fact:zone"]')).toContainText("R2");
  // A condition that does not block issue is answered in Profile, not here.
  await expect(pageOne.locator('[data-control="fact:flag.country"]')).toHaveCount(0);
  await page.screenshot({ path: "../.tools/rfp-report-editing.png", fullPage: true });
  const saved = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  expect(saved.details.find((d: { key: string }) => d.key === "proj.zone").value).toBe("R2");
  expect(saved.details.find((d: { key: string }) => d.key === "proj.client").value).toBe("Inner West Council");

  const scope = page.locator('.report-section').filter({ has: page.getByRole('heading', { name: '2. Scope of services', exact: true }) });
  // Scope by stage: a service is included or excluded in its stage.
  const positions = scope.locator('.rfp-stage-scope');
  for (let i = 0; i < await positions.locator('.rfp-stage-toggle').count(); i++) { const t = positions.locator('.rfp-stage-toggle').nth(i); if (await t.getAttribute('aria-expanded') === 'false') await t.click(); }
  const out = positions.getByRole('checkbox', { name: 'Help selecting other consultants: excluded', exact: true });
  await out.check();
  await expect(out).toBeChecked();
  await refreshDraft(page);
  for (let i = 0; i < await positions.locator('.rfp-stage-toggle').count(); i++) { const t = positions.locator('.rfp-stage-toggle').nth(i); if (await t.getAttribute('aria-expanded') === 'false') await t.click(); }
  await expect(out).toBeChecked();
  await positions.getByRole('checkbox', { name: 'Help selecting other consultants: included', exact: true }).check();
  await expect(out).not.toBeChecked();

  await page.setViewportSize({ width: 1280, height: 2600 });
  await scope.screenshot({ path: '../.tools/rfp-scope-desktop.png' });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: "../.tools/rfp-report-mobile.png", fullPage: true });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
  await scope.screenshot({ path: '../.tools/rfp-scope-mobile.png' });
  await page.setViewportSize({ width: 1280, height: 720 });

  // Protected wording must never be replaced by tables of old generated scope.
  await scope.getByRole('button', { name: 'Edit wording:', exact: true }).last().click();
  await page.getByLabel('Report wording', { exact: true }).fill('Client-approved scope wording; retain this exactly.');
  await page.getByRole('button', { name: 'Save wording', exact: true }).click();
  await expect(scope.getByText('Client-approved scope wording; retain this exactly.', { exact: true })).toBeVisible();
  await expect(scope.getByRole('table')).toHaveCount(1);
  await refreshDraft(page);
  await expect(scope.getByText('Client-approved scope wording; retain this exactly.', { exact: true })).toBeVisible();
  await expect(scope.getByRole('table')).toHaveCount(1);

  // Wording edits where it reads (owner, 7 Nov 2026): a click opens the box
  // in the same place, as tall as its text, and clicking away saves.
  const wording = scope.locator('p.inline-text[data-editable]').first();
  const original = (await wording.textContent())!;
  await wording.scrollIntoViewIfNeeded();
  const before = (await wording.boundingBox())!;
  await wording.click();
  const box = page.getByLabel('Report wording', { exact: true });
  await expect(box).toBeFocused();
  const after = (await box.boundingBox())!;
  expect(Math.abs(after.y - before.y)).toBeLessThan(2);
  expect(Math.abs(after.x - before.x)).toBeLessThan(2);
  expect(await box.evaluate(el => el.scrollHeight <= el.clientHeight)).toBeTruthy();
  await page.keyboard.press('Control+End');
  await page.keyboard.type(' Edited in place.');
  await page.getByRole('heading', { name: '1. Project' }).click();
  await expect(box).toHaveCount(0);
  await expect(scope.getByText(`${original} Edited in place.`, { exact: true })).toBeVisible();
});

test("a package titled Civil is offered the civil RFP and switches only when confirmed", async ({ page }) => {
  const project = await signIn(page, "RFP discipline suggestion");
  const r = await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data: { kind: "services", title: "Civil" } });
  expect(r.ok()).toBeTruthy();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP for Civil", exact: true }).click();
  await expect(page.getByText("This package looks like civil engineering services", { exact: true })).toBeVisible();
  await expect(page.getByRole("region", { name: "Issue", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Use the civil engineering services RFP", exact: true }).click();
  await expect(page.getByRole("region", { name: "Issue", exact: true })).toBeVisible();
  await expect(page.getByText("This package looks like civil engineering services", { exact: true })).toHaveCount(0);
});

test("electrical keeps its allocation schedule beside grouped positions, with separate fire disciplines", async ({ page }) => {
  const project = await signIn(page, "RFP services allocation");
  const catalog = await (await page.request.get("/api/rfp/disciplines")).json();
  const ids = catalog.items.map((item: { id: string }) => item.id);
  expect(ids).toContain("fire-services");
  expect(ids).toContain("fire-engineering");
  expect(ids).toContain("electrical-services");
  for (const data of [
    { kind: "services", title: "Electrical", discipline_id: "consultant.electrical" },
    { kind: "services", title: "Fire services", discipline_id: "consultant.fire_services" },
    { kind: "services", title: "Fire engineering", discipline_id: "consultant.fire_engineer" },
  ]) {
    const response = await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data });
    expect(response.ok(), await response.text()).toBeTruthy();
  }
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP for Electrical", exact: true }).click();
  await expect(page.getByRole("button", { name: "Update", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Assemble draft", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Use last completed state", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Edit 4. Allowances", exact: true }).click();
  const allowances = page.getByRole("group", { name: "Edit allowances" });
  await expect(allowances.getByLabel("Design meetings", { exact: true })).toHaveValue("4");
  await expect(allowances.getByLabel("Design review rounds", { exact: true })).toHaveValue("1");
  await page.screenshot({ path: "../.tools/electrical-simplified-desktop.png", fullPage: true });
  const scope = page.locator(".report-section").filter({ has: page.getByRole("heading", { name: "2. Scope of services", exact: true }) });
  // Allocation notes sit with their service inside the opened stage.
  const allocation = scope.locator('.rfp-stage-scope');
  for (let i = 0; i < await allocation.locator('.rfp-stage-toggle').count(); i++) { const t = allocation.locator('.rfp-stage-toggle').nth(i); if (await t.getAttribute('aria-expanded') === 'false') await t.click(); }
  await expect(allocation).toContainText(/smoke alarms/i);
  await expect(allocation).toContainText(/power.*recovery/i);
  await expect(allocation).toHaveCount(1);
  await refreshDraft(page);
  for (let i = 0; i < await allocation.locator('.rfp-stage-toggle').count(); i++) { const t = allocation.locator('.rfp-stage-toggle').nth(i); if (await t.getAttribute('aria-expanded') === 'false') await t.click(); }
  await expect(allocation).toContainText(/smoke alarms/i);
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
  await scope.screenshot({ path: "../.tools/rfp-electrical-allocation-mobile.png" });
});


test("transmittal follows repository selection, saves snapshots and stays inside report budgets", async ({ page }) => {
  const project = await signIn(page, "Compact RFP transmittal");
  await page.setViewportSize({ width: 1900, height: 1000 });
  await page.getByTestId('file-input').setInputFiles([0,1,2].map(i => ({name: `transmittal-${i}.md`, mimeType: 'text/markdown', buffer: Buffer.from(`# Survey ${i}\nDocument number: T-10${i}\nRevision: A\n`)})));
  const repoRows = page.locator('tbody.reg-doc');
  await expect(repoRows).toHaveCount(3);
  await expect.poll(async () => (await (await page.request.get(`/api/projects/${project}/documents`)).json()).documents.every((d: {status: string}) => d.status === 'filed')).toBe(true);
  const docs = (await (await page.request.get(`/api/projects/${project}/documents`)).json()).documents;
  for (let i = 0; i < docs.length; i++) {
    for (const [field, value] of Object.entries({number:`T-10${i}`, title:`Survey ${i}`, revision:'B', date:'2026-10-28'})) {
      const r = await page.request.put(`/api/documents/${docs[i].id}/fields/${field}`, {headers:origin, data:{value}});
      expect(r.ok(), await r.text()).toBeTruthy();
    }
  }
  const pkg = await (await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data: {kind:'services', title:'Architect', discipline_id:'consultant.architect'} })).json();
  await page.getByRole('button', {name:'Procurement', exact:true}).click();
  await page.getByRole('button', {name:'Create RFP for Architect', exact:true}).click();
  const transmittal = page.getByRole('region', {name:'Transmittal', exact:true});
  await expect(transmittal.locator('tbody tr')).toHaveCount(3);
  await expect(transmittal).toContainText('2026-10-28');
  await expect(transmittal.locator('tbody tr td:nth-child(3)')).toHaveText(['B','B','B']);
  await expect(page.locator('.rfp-release-dates + .rfp-markers')).toHaveCount(0);
  await expect(page.locator('.report-toolbar')).not.toContainText('Draft — not for issue');
  await expect(page.locator('.rfp-grid .pf-tb-head')).toHaveText(['Site and planning', 'The building', 'Delivery']);
  await expect(page.locator('.rfp-page-one .rfp-source')).toHaveCount(0);
  // The button picks up the saved schedule (all three here), not the whole register.
  await transmittal.getByRole('button', {name:'Select transmittal documents', exact:true}).click();
  await expect(repoRows.locator('input[type=checkbox]:checked')).toHaveCount(3);
  await repoRows.nth(1).locator('.reg-title').click({modifiers:['Control']});
  await expect(repoRows.locator('input[type=checkbox]:checked')).toHaveCount(2);
  const selected = await repoRows.evaluateAll(els => els.filter(el => el.hasAttribute('data-selected')).map(el => el.id.slice(4)));
  await transmittal.getByRole('button', {name:'Update Transmittal', exact:true}).click();
  await expect(transmittal.locator('tbody tr')).toHaveCount(2);
  const reports = await (await page.request.get(`/api/projects/${project}/reports`)).json();
  const report = reports.find((r: {package_id:string}) => r.package_id === pkg.id);
  const base = `/api/reports/${report.id}`;
  const block = (view: any) => view.draft.sections.flatMap((s: any) => s.blocks).find((b: any) => b.id === 'rfp:project:information');
  let view = await (await page.request.get(base)).json();
  expect(block(view).basis.document_ids.sort()).toEqual(selected.sort());
  const frozenRows = block(view).table.rows;
  await refreshDraft(page);
  view = await (await page.request.get(base)).json();
  expect(block(view).table.rows).toEqual(frozenRows);
  await page.screenshot({path:'../.tools/rfp-transmittal-desktop.png'});
  await page.setViewportSize({width:390,height:844});
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
  expect((await transmittal.boundingBox())!.width).toBeLessThanOrEqual(390);
  await transmittal.screenshot({path:'../.tools/rfp-transmittal-mobile.png'});
  await page.setViewportSize({width:1900,height:1000});

  // Selecting again brings back the two saved documents only; Ctrl-click adds the third.
  await transmittal.getByRole('button', {name:'Select transmittal documents', exact:true}).click();
  await expect(repoRows.locator('input[type=checkbox]:checked')).toHaveCount(2);
  await repoRows.nth(1).locator('.reg-title').click({modifiers:['Control']});
  await expect(repoRows.locator('input[type=checkbox]:checked')).toHaveCount(3);

  // A failed write leaves both the previous saved schedule and the selection intact.
  await page.route('**/settings/transmittal', route => route.fulfill({status:503,body:'Unavailable'}));
  await transmittal.getByRole('button', {name:'Update Transmittal', exact:true}).click();
  await expect(page.getByRole('alert')).toContainText('Could not load or save');
  await expect(transmittal.locator('tbody tr')).toHaveCount(2);
  await expect(repoRows.locator('input[type=checkbox]:checked')).toHaveCount(3);
  await page.unroute('**/settings/transmittal');
  await transmittal.getByRole('button', {name:'Update Transmittal', exact:true}).click();
  await expect(transmittal.locator('tbody tr')).toHaveCount(3);
  view = await (await page.request.get(base)).json();
  for (const value of [[docs[0].id,docs[0].id], ['00000000-0000-0000-0000-000000000000'], {}, null]) {
    const bad = await page.request.put(`${base}/settings/transmittal`, {headers:origin,data:{value,version:view.draft.version}});
    expect(bad.status()).toBe(400);
  }
  const stale = await page.request.put(`${base}/settings/transmittal`, {headers:origin,data:{value:[],version:1}});
  expect(stale.status()).toBe(409);
  const timings: number[] = [], reads: number[] = [];
  for (let i = 0; i < 25; i++) {
    const started = performance.now();
    const saved = await page.request.put(`${base}/settings/transmittal`, {headers:origin,data:{value:i % 2 ? docs.map((d:any) => d.id) : [],version:view.draft.version}});
    expect(saved.ok(), await saved.text()).toBeTruthy();
    timings.push(performance.now() - started);
    view = await saved.json();
    const readStarted = performance.now();
    const read = await page.request.get(base);
    expect(read.ok()).toBeTruthy();
    reads.push(performance.now() - readStarted);
  }
  for (const [path, values] of [['report_setting', timings], ['report_read', reads]] as const) {
    values.sort((a,b) => a-b);
    console.log(`${path} p50=${values[12].toFixed(1)}ms p90=${values[22].toFixed(1)}ms; budget 100/250ms`);
    expect(values[12]).toBeLessThanOrEqual(100); expect(values[22]).toBeLessThanOrEqual(250);
  }
  expect(block(view).table.rows).toEqual([]);
  await page.reload();
  await page.getByRole('button', {name:'Procurement', exact:true}).click();
  const open = page.getByRole('button', {name:'View RFP for Architect', exact:true});
  if (await open.count()) await open.click();
  await expect(page.getByRole('region', {name:'Transmittal',exact:true})).toContainText('No documents in this transmittal.');
});
