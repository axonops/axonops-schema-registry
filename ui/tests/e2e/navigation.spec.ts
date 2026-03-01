import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("sidebar is visible with all nav groups", async ({ page }) => {
    await expect(page.getByTestId("app-sidebar")).toBeVisible();
    // Check key nav sections exist
    await expect(
      page.getByTestId("nav-sidebar-dashboard-link"),
    ).toBeVisible();
    await expect(
      page.getByTestId("nav-sidebar-subjects-link"),
    ).toBeVisible();
    await expect(
      page.getByTestId("nav-sidebar-register-link"),
    ).toBeVisible();
    await expect(page.getByTestId("nav-sidebar-users-link")).toBeVisible();
    await expect(
      page.getByTestId("nav-sidebar-api-docs-link"),
    ).toBeVisible();
  });

  const navTests = [
    { testId: "nav-sidebar-subjects-link", pageTestId: "subjects-list-page" },
    { testId: "nav-sidebar-users-link", pageTestId: "users-page" },
  ];

  for (const { testId, pageTestId } of navTests) {
    test(`clicking ${testId} navigates to page`, async ({ page }) => {
      await page.getByTestId(testId).click();
      await expect(page.getByTestId(pageTestId)).toBeVisible({
        timeout: 5_000,
      });
    });
  }

  test("quick action buttons navigate correctly", async ({ page }) => {
    // Click "Browse Subjects" quick action
    await page.getByTestId("dashboard-quick-actions").getByText("Browse Subjects").click();
    await expect(page.getByTestId("subjects-list-page")).toBeVisible({
      timeout: 5_000,
    });
  });

  test("breadcrumb shows on nested pages", async ({ page }) => {
    await page.getByTestId("nav-sidebar-subjects-link").click();
    await expect(page.getByRole("navigation", { name: "breadcrumb" })).toBeVisible();
  });

  test("SPA routing works for deep links", async ({ page }) => {
    // Navigate directly to a deep route — SPA should handle it
    await page.goto("/ui/subjects");
    // Should either show subjects page or redirect to login
    const subjectsPage = page.getByTestId("subjects-list-page");
    const loginPage = page.getByTestId("login-page");

    // One of these should be visible (depends on session cookie)
    await expect(subjectsPage.or(loginPage)).toBeVisible({ timeout: 5_000 });
  });
});
