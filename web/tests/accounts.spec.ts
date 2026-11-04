import { expect, test } from "@playwright/test";

// Anyone can create an account and come back with email and password.
test("sign up, then sign in again from a fresh browser", async ({ browser }) => {
  const email = `e2e-${Date.now()}@example.com`;
  const password = "e2e password 1";

  const first = await (await browser.newContext()).newPage();
  await first.goto("/");
  await expect(first.getByRole("heading", { name: "Sign in" })).toBeVisible();
  await first.getByRole("button", { name: "Create an account" }).click();
  await first.getByLabel("Email").fill(email);
  await first.getByLabel("New password").fill(password);
  await first.getByRole("button", { name: "Create account" }).click();
  await expect(first.getByRole("heading", { name: "Projects" })).toBeVisible();

  const second = await (await browser.newContext()).newPage();
  await second.goto("/");
  await second.getByLabel("Email").fill(email);
  await second.getByLabel("Password").fill("wrong password");
  await second.getByRole("button", { name: "Sign in" }).click();
  await expect(second.getByRole("alert")).toHaveText("Wrong email or password.");
  await second.getByLabel("Password").fill(password);
  await second.getByRole("button", { name: "Sign in" }).click();
  await expect(second.getByRole("heading", { name: "Projects" })).toBeVisible();
});

test("forgot password never says whether an account exists", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Forgot password?" }).click();
  await page.getByLabel("Email").fill(`nobody-${Date.now()}@example.com`);
  await page.getByRole("button", { name: "Send reset link" }).click();
  await expect(page.getByRole("heading", { name: "Check your email" })).toBeVisible();
});
