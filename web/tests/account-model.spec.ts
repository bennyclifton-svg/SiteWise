import { expect, test } from "@playwright/test";
import { freshOwner } from "./tender-helpers";

// The org's one switch for sending tender pages to the model. Only the owner
// sees it; it is off until they turn it on, and it stays as they left it.
test("the owner switches the model on in Account and it stays on", async ({ page }) => {
  await freshOwner(page, "model");
  await page.goto("/account");
  const section = page.getByRole("region", { name: "Reading tenders with a model" });
  await expect(section).toContainText("Nothing is sent while this is off");
  const toggle = section.getByRole("switch", { name: "Send tender pages to the model" });
  await expect(toggle).not.toBeChecked();
  await toggle.check();
  await expect(section).toContainText("On since");
  await page.reload();
  await expect(page.getByRole("switch", { name: "Send tender pages to the model" })).toBeChecked();
  const account = await (await page.request.get("/api/account")).json();
  expect(account.model).toMatchObject({ available: true, enabled: true });
  await page.getByRole("switch", { name: "Send tender pages to the model" }).uncheck();
  await expect(page.getByRole("region", { name: "Reading tenders with a model" })).not.toContainText("On since");
});
