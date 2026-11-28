import { expect, test } from "@playwright/test";
import {readFileSync} from "node:fs";
import {resolve} from "node:path";
import type { Page } from "@playwright/test";

// Add item, Import and Preview / issue sit behind the heading's three dots
// (PanelHead, owner 25 November 2026).
async function specAction(page:Page,name:string){await page.getByRole("button",{name:"Specification actions",exact:true}).click();await page.getByRole("menuitem",{name,exact:true}).click();}
async function specItem(page:Page,name:string){await page.getByRole("button",{name:"Specification actions",exact:true}).click();const item=page.getByRole("menuitem",{name,exact:true});return item;}

test("a user authors, reviews and issues a standalone specification", async ({page}) => {
  const {token}=await(await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Synthetic specification authoring");
  await page.getByRole("button",{name:"Create project",exact:true}).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}/);await page.goto(`${new URL(page.url()).pathname}?view=specifications`);
  await expect(page.getByRole("columnheader",{name:"Type / specification",exact:true})).toBeVisible();
  await expect(page.getByRole("button",{name:"Specify basin",exact:true})).toBeVisible();
  await page.setViewportSize({width:1360,height:1000});
  await page.getByRole("button",{name:"Add item to Interiors and finishes",exact:true}).click();
  const newLine=page.getByRole("form",{name:"New item in Interiors and finishes",exact:true});
  await newLine.getByLabel("Item name").fill("Bathroom mirror");
  await newLine.getByLabel("Code",{exact:true}).fill("MR1");
  await newLine.getByRole("button",{name:"Add line item",exact:true}).click();
  await expect(page.getByRole("button",{name:"Edit Item name for Bathroom mirror",exact:true})).toBeVisible();
  await page.getByRole("button",{name:"Update specification",exact:true}).click();
  // The page has other live regions (the project name's saving line since
  // 4 November 2026, the repository's live state), so the specification's
  // own message is read where it shows.
  await expect(page.locator(".spec-feedback")).toContainText("Current documents are already read or queued");
  await page.screenshot({path:"../tmp/specification-framework-desktop.png"});
  await page.setViewportSize({width:390,height:844});
  await page.screenshot({path:"../tmp/specification-framework-mobile.png"});
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBeTruthy();
  await page.setViewportSize({width:1360,height:1000});
  await specAction(page,"Add item");
  await page.getByLabel("Item name",{exact:true}).fill("Bathroom tile");
  await page.getByLabel("Code",{exact:true}).fill("TIL1");
  await page.getByRole("combobox",{name:"Building system",exact:true}).selectOption("interiors.tiling");
  await page.getByRole("button",{name:"Save item",exact:true}).click();
  await expect(page.getByRole("button",{name:"Edit item",exact:true})).toBeVisible();
  for(const location of ["Ground floor bathroom","Upstairs bathroom"]){
    await page.getByRole("button",{name:"Add location",exact:true}).click();
    await page.getByLabel("Location",{exact:true}).fill(location);
    await page.getByLabel("Surface",{exact:true}).fill("floor");
    await page.getByRole("combobox",{name:"Choice",exact:true}).selectOption("selected");
    await page.getByRole("button",{name:"Save location",exact:true}).click();
    await expect(page.getByRole("button",{name:`Edit ${location}`,exact:true})).toBeVisible();
  }
  await page.getByRole("button",{name:"Add a detail or requirement",exact:true}).click();
  await page.getByRole("combobox",{name:"Attribute",exact:true}).selectOption("grout");
  await page.getByRole("combobox",{name:"Value state",exact:true}).selectOption("unknown");
  await page.getByRole("button",{name:"Save for review",exact:true}).click();
  await page.getByRole("button",{name:"Accept value",exact:true}).click();
  await expect(page.getByText("All applications · accepted · User input",{exact:true})).toBeVisible();
  await page.getByRole("button",{name:"Close new item",exact:true}).click();
  await page.getByRole("button",{name:"Edit Material / product for Bathroom tile",exact:true}).click();
  await page.getByRole("textbox",{name:"Material / product for Bathroom tile",exact:true}).fill("Porcelain tile");
  await page.getByRole("button",{name:"Save",exact:true}).click();
  await expect(page.getByRole("button",{name:"Edit Material / product for Bathroom tile",exact:true})).toHaveText("Porcelain tile");
  const tileRow=page.locator("tr[data-spec-item]").filter({hasText:"Bathroom tile"});
  await tileRow.scrollIntoViewIfNeeded();
  await page.screenshot({path:"../tmp/specification-filled-desktop.png"});
  await page.setViewportSize({width:390,height:844});
  await tileRow.scrollIntoViewIfNeeded();
  await page.screenshot({path:"../tmp/specification-filled-mobile.png"});
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBeTruthy();
  await page.setViewportSize({width:1360,height:1000});
  await specAction(page,"Preview / issue");
  await expect(page.getByText("Ground floor bathroom · floor · selected · unknown",{exact:true})).toBeVisible();
  await expect(page.getByText("Upstairs bathroom · floor · selected · unknown",{exact:true})).toBeVisible();
  await expect(page.getByText("(selection): unknown",{exact:false})).toHaveCount(2);
  await page.getByRole("button",{name:"Issue schedule",exact:true}).click();
  await expect(page.getByRole("link",{name:"Issue 1 CSV",exact:true})).toBeVisible();
  const download=page.waitForEvent("download");
  await page.getByRole("link",{name:"Issue 1 CSV",exact:true}).click();
  expect((await download).suggestedFilename()).toBe("specification-issue-1.csv");
  const pdfDownload=page.waitForEvent("download");
  await page.getByRole("link",{name:"Issue 1: Materials and finishes schedule · PDF",exact:true}).click();
  expect((await pdfDownload).suggestedFilename()).toBe("specification-issue-1.pdf");
  await page.setViewportSize({width:1360,height:1000});
  await page.screenshot({path:"../tmp/specifications-desktop.png",fullPage:true});
  await page.setViewportSize({width:390,height:844});
  await page.screenshot({path:"../tmp/specifications-mobile.png",fullPage:true});
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBeTruthy();
  await specAction(page,"Add item");
  const heading=page.getByRole("heading",{name:"Add a specification item",exact:true});
  await expect(heading).toBeFocused();
  expect((await heading.boundingBox())!.y).toBeLessThan(250);
  await page.screenshot({path:"../tmp/specifications-mobile-form.png"});
  await page.getByRole("button",{name:"Cancel",exact:true}).click();
  await specAction(page,"Preview / issue");
  await expect(page.getByRole("heading",{name:"Materials and finishes schedule",exact:true})).toBeVisible();
  await page.screenshot({path:"../tmp/specifications-mobile-reader.png"});
});

test("specification load failures preserve recovery and do not look empty",async({page})=>{
  const {token}=await(await page.request.post("/__e2e/invite?org=a")).json();await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Synthetic specification recovery");await page.getByRole("button",{name:"Create project",exact:true}).click();
  let broken=true;await page.route("**/api/projects/*/specifications",r=>broken?r.fulfill({status:500,body:"unavailable"}):r.continue());
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}/);await page.goto(`${new URL(page.url()).pathname}?view=specifications`);
  await expect(page.getByRole("alert")).toContainText("Could not load specifications");
  await expect(await specItem(page,"Add item")).toBeDisabled();
  await page.keyboard.press("Escape");
  broken=false;await page.getByRole("button",{name:"Reload specifications",exact:true}).click();
  await expect(await specItem(page,"Add item")).toBeEnabled();
});

test("a consultant reference can cover a section without copying its specification",async({page})=>{
  const {token}=await(await page.request.post("/__e2e/invite?org=a")).json();await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Synthetic consultant coverage");await page.getByRole("button",{name:"Create project",exact:true}).click();
  await expect(page).toHaveURL(/projects\//);
  const project=new URL(page.url()).pathname.split("/")[2];
  const bytes=readFileSync(resolve(import.meta.dirname,"../../testdata/identity/identity-page.pdf"));
  const upload=await page.request.post(`/api/projects/${project}/files?name=consultant-materials.pdf`,{data:bytes,headers:{"Content-Type":"application/pdf",Origin:"http://127.0.0.1:4173"}});
  expect(upload.ok()).toBeTruthy();
  const doc=await upload.json();await page.reload();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}/);await page.goto(`${new URL(page.url()).pathname}?view=specifications`);
  const section=page.locator("tr.spec-section").filter({has:page.getByRole("button",{name:"Interiors and finishes",exact:true})});
  await section.getByRole("button",{name:"Needs attention",exact:true}).click();
  const form=page.getByRole("form",{name:"Specification coverage",exact:true});
  await expect(form).toBeVisible();
  await form.getByRole("combobox",{name:"Coverage",exact:true}).selectOption("external");
  await form.getByLabel("Covered scope / extent").fill("All interior materials and finishes");
  await form.getByLabel("Specification maintained by").fill("Architect");
  await form.getByLabel("Review coordinated by").fill("Project manager");
  await form.getByRole("combobox",{name:"Reference document",exact:true}).selectOption(doc.id);
  await form.getByRole("button",{name:"Add reference",exact:true}).click();
  await form.getByLabel("Page / schedule / section").fill("Schedule A, pages 1–4");
  await form.getByRole("button",{name:"Confirm coverage and revisions",exact:true}).click();
  await expect(section.getByRole("button",{name:"Covered by document",exact:true})).toBeVisible();
  await expect(page.getByRole("button",{name:"Specify floor tile",exact:true})).toHaveCount(0);
  await specAction(page,"Preview / issue");
  await expect(page.getByRole("heading",{name:"Consultant references and coverage",exact:true})).toBeVisible();
  await expect(page.getByText("Maintained by: Architect · Review coordinated by: Project manager",{exact:true})).toBeVisible();
  await page.getByRole("button",{name:"Issue schedule",exact:true}).click();
  await expect(page.getByRole("link",{name:"Issue 1 CSV",exact:true})).toBeVisible();
  await page.setViewportSize({width:1360,height:1000});
  await page.screenshot({path:"../tmp/specification-coverage-desktop.png"});
});
