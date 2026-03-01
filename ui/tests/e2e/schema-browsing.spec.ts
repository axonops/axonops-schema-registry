import { test, expect } from "@playwright/test";
import { login } from "./helpers";

test.describe("Schema Browsing", () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test("dashboard shows subject and schema counts", async ({ page }) => {
    // Stats should be populated (demo seed provides data)
    const subjectsStat = page.getByTestId("dashboard-stat-subjects");
    await expect(subjectsStat).toBeVisible();
    // The count should be > 0 (exact number depends on seed data)
    await expect(subjectsStat).not.toContainText("0");

    const schemasStat = page.getByTestId("dashboard-stat-schemas");
    await expect(schemasStat).toBeVisible();
    await expect(schemasStat).not.toContainText("0");
  });

  test("dashboard shows recent schemas table", async ({ page }) => {
    await expect(page.getByTestId("dashboard-recent-schemas")).toBeVisible();
    // Table should have at least one row
    const rows = page
      .getByTestId("dashboard-recent-schemas")
      .locator("tbody tr");
    await expect(rows.first()).toBeVisible();
  });

  test("navigate to subjects list via sidebar", async ({ page }) => {
    await page.getByTestId("nav-sidebar-subjects-link").click();
    await expect(page.getByTestId("subjects-list-page")).toBeVisible();
    await expect(page.getByTestId("subjects-list-table")).toBeVisible();
  });

  test("subjects list shows seeded subjects", async ({ page }) => {
    await page.getByTestId("nav-sidebar-subjects-link").click();
    await expect(page.getByTestId("subjects-list-table")).toBeVisible();
    // Should have multiple rows from seed data
    const rows = page.getByTestId("subjects-list-table").locator("tbody tr");
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
  });

  test("subjects search filters results", async ({ page }) => {
    await page.getByTestId("nav-sidebar-subjects-link").click();
    await expect(page.getByTestId("subjects-list-table")).toBeVisible();

    const allRows = page
      .getByTestId("subjects-list-table")
      .locator("tbody tr");
    const totalBefore = await allRows.count();

    // Search for a specific subject
    await page.getByTestId("subjects-search-input").fill("user");
    // Wait for filtering
    await page.waitForTimeout(500);

    const filteredRows = page
      .getByTestId("subjects-list-table")
      .locator("tbody tr");
    const totalAfter = await filteredRows.count();
    expect(totalAfter).toBeLessThanOrEqual(totalBefore);
    expect(totalAfter).toBeGreaterThan(0);
  });

  test("clicking a subject row navigates to detail", async ({ page }) => {
    await page.getByTestId("nav-sidebar-subjects-link").click();
    await expect(page.getByTestId("subjects-list-table")).toBeVisible();

    // Click the first subject row
    const firstRow = page
      .getByTestId("subjects-list-table")
      .locator("tbody tr")
      .first();
    await firstRow.click();

    // Should navigate to subject detail (URL contains /subjects/)
    await page.waitForURL(/\/ui\/subjects\/.+/);
  });

  test("health status shows healthy", async ({ page }) => {
    const healthStat = page.getByTestId("dashboard-stat-health");
    await expect(healthStat).toBeVisible();
    await expect(healthStat).toContainText("Healthy");
  });
});
