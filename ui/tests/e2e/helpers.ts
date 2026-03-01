import { type Page, expect } from "@playwright/test";

/** Log in via the login form and wait for the dashboard. */
export async function login(
  page: Page,
  username = "admin",
  password = "admin",
) {
  await page.goto("/ui/login");
  await page.getByTestId("login-username-input").fill(username);
  await page.getByTestId("login-password-input").fill(password);
  await page.getByTestId("login-submit-btn").click();
  await expect(page.getByTestId("dashboard-page")).toBeVisible({
    timeout: 10_000,
  });
}

/** Assert we've been redirected to the login page. */
export async function expectLoginPage(page: Page) {
  await expect(page.getByTestId("login-page")).toBeVisible({ timeout: 5_000 });
}
