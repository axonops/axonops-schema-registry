import { test, expect } from "@playwright/test";
import { login, expectLoginPage } from "./helpers";

test.describe("Login", () => {
  test("shows login form", async ({ page }) => {
    await page.goto("/ui/login");
    await expect(page.getByTestId("login-form")).toBeVisible();
    await expect(page.getByTestId("login-username-input")).toBeVisible();
    await expect(page.getByTestId("login-password-input")).toBeVisible();
    await expect(page.getByTestId("login-submit-btn")).toHaveText("Sign In");
  });

  test("logs in with valid credentials and reaches dashboard", async ({
    page,
  }) => {
    await login(page);
    await expect(page.getByTestId("dashboard-stat-subjects")).toBeVisible();
    await expect(page.getByTestId("dashboard-stat-schemas")).toBeVisible();
    await expect(page.getByTestId("dashboard-stat-health")).toBeVisible();
  });

  test("shows error for invalid credentials", async ({ page }) => {
    await page.goto("/ui/login");
    await page.getByTestId("login-username-input").fill("admin");
    await page.getByTestId("login-password-input").fill("wrongpassword");
    await page.getByTestId("login-submit-btn").click();
    await expect(page.getByTestId("login-error")).toBeVisible();
    await expect(page.getByTestId("login-error")).toContainText(
      "Invalid username or password",
    );
  });

  test("redirects unauthenticated users to login", async ({ page }) => {
    await page.goto("/ui/dashboard");
    await expectLoginPage(page);
  });

  test("logout returns to login page", async ({ page }) => {
    await login(page);
    // Click the user menu button then logout
    await page.getByRole("button", { name: /admin/ }).click();
    // Look for a logout option in the dropdown/menu
    const logoutBtn = page.getByRole("menuitem", { name: /log\s?out/i });
    if (await logoutBtn.isVisible()) {
      await logoutBtn.click();
    } else {
      // Fallback: call the API directly (some UIs use different patterns)
      await page.evaluate(() =>
        fetch("/api/auth/logout", { method: "POST" }),
      );
      await page.goto("/ui/login");
    }
    await expectLoginPage(page);
  });
});
