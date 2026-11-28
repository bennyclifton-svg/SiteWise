import { expect, test } from "@playwright/test";

// The public front door: a signed-out visitor at "/" sees the sample project
// being built, and its one action leads to the create-account panel.

const HEADLINE = "Files your documents in about a second.";
const STEPS = ["Builds your project profile.", "Builds your cost plan.", "Builds your program.", "Sets up and evaluates your tenders."];

test("a signed-out visitor sees the landing page and can start an account", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1, name: HEADLINE })).toBeVisible();
  // The profile plays first, as an Update: the strip shows documents being read.
  await expect(page.getByRole("tab", { name: STEPS[0] })).toHaveAttribute("aria-selected", "true");
  await expect(page.locator(".lp-strip[data-mode=run]")).toBeVisible({ timeout: 5000 });
  // Each step can be chosen; Pause holds the stage where it is.
  await page.getByRole("tab", { name: STEPS[3] }).click();
  await expect(page.getByRole("tab", { name: STEPS[3] })).toHaveAttribute("aria-selected", "true");
  await page.getByRole("button", { name: "Pause" }).click();
  await expect(page.getByRole("button", { name: "Play" })).toBeVisible();
  await page.getByRole("button", { name: "Start a project free" }).first().click();
  await expect(page).toHaveURL(/\/signup$/);
  await expect(page.getByRole("heading", { name: "Create your account" })).toBeVisible();
  // Back returns to the landing page; Sign in opens the sign-in panel.
  await page.goBack();
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(page).toHaveURL(/\/signin$/);
  await expect(page.getByRole("heading", { name: "Sign in", exact: true })).toBeVisible();
});

test("the landing page fits a phone and reads without motion", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/welcome");
  await expect(page.getByRole("heading", { level: 1, name: HEADLINE })).toBeVisible();
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(overflow).toBeLessThanOrEqual(0);
  // With reduced motion nothing is covered and nothing moves by itself: the
  // profile shows finished, and the steps still switch the screen.
  await expect(page.locator(".shot-layer")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Pause" })).toHaveCount(0);
  await expect(page.locator(".lp-strip[data-mode=done]")).toBeVisible();
  await page.getByRole("tab", { name: STEPS[1] }).click();
  await expect(page.getByRole("tab", { name: STEPS[1] })).toHaveAttribute("aria-selected", "true");
});

test("a link from an email opens the sign-in panel, not the landing page", async ({ page }) => {
  await page.goto("/#reset=0123456789abcdef");
  await expect(page.getByRole("heading", { name: "Choose a new password" })).toBeVisible();
});
