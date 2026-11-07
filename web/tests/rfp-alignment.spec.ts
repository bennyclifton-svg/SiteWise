import { expect, test } from "@playwright/test";
const origin = { Origin: "http://127.0.0.1:4173" };
const disciplines = [{"id": "access", "discipline": "consultant.access", "positions": 11}, {"id": "arborist", "discipline": "consultant.arborist", "positions": 12}, {"id": "architecture", "discipline": "consultant.architect", "positions": 35}, {"id": "bca-certification", "discipline": "consultant.bca", "positions": 12}, {"id": "civil", "discipline": "consultant.civil", "positions": 31}, {"id": "cost-planning", "discipline": "consultant.quantity_surveyor", "positions": 13}, {"id": "electrical-services", "discipline": "consultant.electrical", "positions": 19}, {"id": "fire-engineering", "discipline": "consultant.fire_engineer", "positions": 15}, {"id": "fire-services", "discipline": "consultant.fire_services", "positions": 17}, {"id": "geotechnical", "discipline": "consultant.geotechnical", "positions": 13}, {"id": "hydraulic-services", "discipline": "consultant.hydraulic", "positions": 24}, {"id": "land-surveying", "discipline": "consultant.surveyor", "positions": 13}, {"id": "landscape-architecture", "discipline": "consultant.landscape", "positions": 12}, {"id": "mechanical-services", "discipline": "consultant.mechanical", "positions": 18}, {"id": "structural", "discipline": "consultant.structural", "positions": 20}, {"id": "town-planning", "discipline": "consultant.town_planner", "positions": 12}, {"id": "traffic", "discipline": "consultant.traffic", "positions": 11}, {"id": "waste-management", "discipline": "consultant.waste", "positions": 13}];
for (const d of disciplines) test(`${d.id}: complete unified scope, inputs and stage pricing`, async ({ page }) => {
 const {token} = await (await page.request.post("/__e2e/invite?org=a")).json();
 await page.goto(`/#token=${token}`);
 await page.getByLabel("New project").fill(`Alignment ${d.id}`);
 await page.getByRole("button",{name:"Create project",exact:true}).click();
 await expect(page.getByRole("button",{name:"Procurement",exact:true})).toBeVisible();
 const project = new URL(page.url()).pathname.split("/").pop()!;
 const pkg=await page.request.post(`/api/projects/${project}/packages`,{headers:origin,data:{kind:"services",title:d.id,discipline_id:d.discipline}});
 expect(pkg.ok(),await pkg.text()).toBeTruthy();
 await page.getByRole("button",{name:"Procurement",exact:true}).click();
 await page.getByRole("button",{name:/^Create RFP for /}).click();
 await expect(page.getByRole("heading",{name:"2. Scope of services",exact:true})).toBeVisible();
 // Scope by stage (3 November 2026): one stage table under section 2.
 await expect(page.getByRole("heading",{name:"3. Transmittal",exact:true})).toBeVisible();
 await expect(page.getByRole("heading",{name:/Scope by stage|Service positions|Your role|Evaluation|Scope position/})).toHaveCount(0);
 await expect(page.getByText("This appointment buys",{exact:true})).toHaveCount(0);
 await expect(page.getByRole("group",{name:"Proposal dates",exact:true}).locator('input')).toHaveCount(2);
 const reports=await (await page.request.get(`/api/projects/${project}/reports`)).json();
 const reportId=reports[0].id;
 const view=await (await page.request.get(`/api/reports/${reportId}`)).json();
 const block=view.draft.sections.flatMap((s:any)=>s.blocks).find((b:any)=>b.id==="rfp:stages");
 expect(block.basis.layout).toBe("stage_scope");
 const table=page.locator('.rfp-stage-scope');
 await expect(table).toHaveCount(1);
 await expect(table.getByRole('radio')).toHaveCount(0);
 await expect(table.locator('select')).toHaveCount(0);
 const applies=block.basis.stages.filter((s:any)=>!(s.state==="exclude"&&s.source==="rule"));
 await expect(table.locator('.rfp-stage-row')).toHaveCount(applies.length+1);
 // Open every stage: each shows its lines, its elements and its services.
 for (let i = 0; i < await table.locator('.rfp-stage-toggle').count(); i++) { const t = table.locator('.rfp-stage-toggle').nth(i); if (await t.getAttribute('aria-expanded') === 'false') await t.click(); }
 const shown=(key:string)=>block.basis.services.filter((v:any)=>(v.stage??"")===key&&!v.inactive).length;
 let expected=shown("");
 for (const st of applies) if (st.state!=="exclude") expected+=(st.lines??[]).length+(st.by_element?block.basis.elements.length:0)+shown(st.key);
 await expect(table.locator('.rfp-stage-line')).toHaveCount(expected);
 const inputGroup=view.rfp.groups.find((g:any)=>g.id==='appointment');
 for (const input of inputGroup.controls) await expect(page.locator(`.rfp-scope-inputs [data-control="${input.key}"]`)).toHaveCount(1);
 // Exclude one service, keep it after a reload, then include it again.
 const service=block.basis.services.find((v:any)=>!v.inactive&&v.value==="I"&&applies.some((s:any)=>s.key===v.stage&&s.state!=="exclude"));
 const out=table.getByRole('checkbox',{name:`${service.label}: excluded`,exact:true});
 // Wait for the save: the tick shows at once, before the draft is written.
 await Promise.all([page.waitForResponse(r=>r.request().method()==="PUT"&&decodeURIComponent(r.url()).includes(`/settings/position:${service.key}`)),out.check()]);
 await expect(out).toBeChecked();
 const saved=await (await page.request.get(`/api/reports/${reportId}`)).json();
 expect(saved.draft.sections.flatMap((s:any)=>s.blocks).find((b:any)=>b.id==="rfp:stages").basis.services.find((v:any)=>v.key===service.key).value).toBe("X");
 await page.reload();
 await page.getByRole("button",{name:"Procurement",exact:true}).click();
 await page.getByRole("button",{name:/^View RFP for /}).click();
 await table.getByRole('button',{name:applies.find((s:any)=>s.key===service.stage).name,exact:true}).click();
 await expect(out).toBeChecked();
 await table.getByRole('checkbox',{name:`${service.label}: included`,exact:true}).check();
 await expect(table.getByRole('checkbox',{name:`${service.label}: included`,exact:true})).toBeChecked();
 if(d.id==='mechanical-services') {await page.screenshot({path:'../.tools/rfp-aligned-desktop.png',fullPage:true});await page.setViewportSize({width:390,height:844});await table.scrollIntoViewIfNeeded();await page.screenshot({path:'../.tools/rfp-aligned-mobile.png'});}
 if(d.id==='civil') {
  // Elements are lines inside one stage fee.
  const elements=table.locator('tr[data-control^="element:"]');
  expect(await elements.count()).toBeGreaterThanOrEqual(4);
  const first=elements.first();
  await Promise.all([page.waitForResponse(r=>r.request().method()==="PUT"&&decodeURIComponent(r.url()).includes("/settings/element:")),first.getByRole('checkbox',{name:/: excluded$/}).check()]);
  await expect(first.getByRole('checkbox',{name:/: excluded$/})).toBeChecked();
  const after=await (await page.request.get(`/api/reports/${reportId}`)).json();
  expect(after.draft.sections.flatMap((s:any)=>s.blocks).find((b:any)=>b.id==="rfp:stages").basis.elements.some((e:any)=>e.state==="exclude")).toBeTruthy();
 }
});
