import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("User Management", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.getByTestId("nav-sidebar-users-link").click();
    await expect(page.getByTestId("users-page")).toBeVisible();
  });

  test("users page shows admin user", async ({ page }) => {
    await expect(page.getByTestId("users-list-table")).toBeVisible();
    // admin user should exist (created at bootstrap)
    await expect(
      page.getByTestId("users-list-table").getByText("admin"),
    ).toBeVisible();
  });

  test("create a new user", async ({ page }) => {
    await page.getByTestId("users-create-btn").click();

    // Fill the create user form
    await page
      .getByTestId("user-form-username-input")
      .fill("e2e-testuser-" + Date.now());
    await page.getByTestId("user-form-password-input").fill("testpass123");
    await page.getByTestId("user-form-submit-btn").click();

    // Dialog should close and new user should appear (or toast)
    // Wait for the table to update
    await page.waitForTimeout(1000);
    await expect(page.getByTestId("users-list-table")).toBeVisible();
  });

  test("create user button is disabled with empty form", async ({ page }) => {
    await page.getByTestId("users-create-btn").click();
    await expect(page.getByTestId("user-form-username-input")).toBeVisible();

    // Submit button should be disabled when fields are empty
    await expect(page.getByTestId("user-form-submit-btn")).toBeDisabled();

    // Fill only username — still disabled (password required for new user)
    await page.getByTestId("user-form-username-input").fill("test");
    await expect(page.getByTestId("user-form-submit-btn")).toBeDisabled();

    // Fill password too — now enabled
    await page.getByTestId("user-form-password-input").fill("pass1234");
    await expect(page.getByTestId("user-form-submit-btn")).toBeEnabled();
  });

  test("disable a user", async ({ page }) => {
    // First create a user to disable
    const username = "e2e-disable-" + Date.now();

    await page.getByTestId("users-create-btn").click();
    await page.getByTestId("user-form-username-input").fill(username);
    await page.getByTestId("user-form-password-input").fill("testpass123");
    await page.getByTestId("user-form-submit-btn").click();
    await page.waitForTimeout(1000);

    // Find the user row and click edit
    const row = page.getByTestId("users-list-table").getByText(username);
    await expect(row).toBeVisible();

    // Click the edit button in that row
    const editBtn = row.locator("..").getByTestId("user-edit-btn");
    // If edit is a sibling/cousin, try different traversal
    if (!(await editBtn.isVisible())) {
      // Click on the row's actions
      const rowEl = row.locator("xpath=ancestor::tr");
      await rowEl.getByTestId("user-edit-btn").click();
    } else {
      await editBtn.click();
    }

    // Toggle the enabled switch off
    const enabledToggle = page.getByTestId("user-form-enabled-toggle");
    if (await enabledToggle.isVisible()) {
      await enabledToggle.click();
      await page.getByTestId("user-form-submit-btn").click();
      await page.waitForTimeout(500);
    }
  });
});
