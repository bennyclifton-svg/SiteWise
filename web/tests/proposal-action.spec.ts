import { expect, test } from "@playwright/test";

// Implications is off the left nav. Proposal records stay; this screen is not opened.
test.beforeEach(() => { test.skip(true, "Implications is off the left nav"); });

test("physical make-good implication requires an explicit action", async ({ page }) => {
 const {token}=await(await page.request.post("/__e2e/invite?org=a")).json();
 await page.goto(`/#token=${token}`);await page.getByLabel("New project").fill("Explicit make-good action");await page.getByRole("button",{name:"Create project",exact:true}).click();
 await expect(page.getByRole("button",{name:"Implications",exact:true})).toBeVisible();
 const project=new URL(page.url()).pathname.split("/").pop();const headers={Origin:"http://127.0.0.1:4173"};const profile=await(await page.request.get(`/api/projects/${project}/profile`)).json();
 const created=await page.request.post(`/api/projects/${project}/works`,{headers,data:{part_id:profile.parts[0].id,system_id:"mechanical.local-exhaust",action:"alter",title:"Alter exhaust"}});expect(created.ok()).toBeTruthy();
 const proposals=(await(await page.request.get(`/api/projects/${project}/proposals?show=all`)).json()).items;
 const proposal=proposals.find((p:{record_id:string;target_system_id?:string;target_part_id?:string})=>p.record_id==="ic.penetrates-make-good"&&p.target_system_id&&p.target_part_id);expect(proposal).toBeTruthy();
 await page.getByRole("button",{name:"Implications",exact:true}).click();
 const row=page.locator(`tr.implication-row[data-key="${proposal.key}"]`);await row.getByRole("button",{name:/^Accept /}).click();
 await expect(page.getByRole("button",{name:"Accept for planning",exact:true})).toBeDisabled();await page.getByLabel("Action for the new work item",{exact:true}).selectOption("repair");await page.getByRole("button",{name:"Accept for planning",exact:true}).click();await expect(page.getByRole("region",{name:"Saved decision",exact:true})).toBeVisible();
 const items=(await(await page.request.get(`/api/projects/${project}/works`)).json()).items;expect(items.some((w:{source_proposal_key:string;action:string})=>w.source_proposal_key===proposal.key&&w.action==="repair")).toBeTruthy();
});
