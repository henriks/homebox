import type { Page } from "@playwright/test";
import { expect, test } from "@playwright/test";

const templatesURL = "**/api/v1/label-templates";

async function openEntity(page: Page, type: "item" | "location" = "item", withoutAssetId = false) {
  const response = await page.request.get(`/api/v1/entities?isLocation=${type === "location"}&pageSize=1`);
  expect(response.ok(), await response.text()).toBeTruthy();
  const { items } = await response.json();
  expect(items.length).toBeGreaterThan(0);
  if (withoutAssetId) {
    await page.route(`**/api/v1/entities/${items[0].id}`, async route => {
      const response = await route.fetch();
      await route.fulfill({ response, json: { ...(await response.json()), assetId: "" } });
    });
  }
  await page.goto(`/${type}/${items[0].id}`);
  await expect(page.getByRole("button", { name: "Print label", exact: true })).toBeVisible({ timeout: 15000 });
  return items[0] as { id: string; assetId: string };
}

test.beforeEach(async ({ page }) => {
  await page.route("**/api/v1/status", async route => {
    const response = await route.fetch();
    await route.fulfill({ response, json: { ...(await response.json()), labelPrinting: true } });
  });
  await page.route(templatesURL, route => route.fulfill({ json: [] }));
  await page.goto("/");
  await page.fill("input[type='text']", "demo@example.com");
  await page.fill("input[type='password']", process.env.E2E_PASSWORD || "demodemo");
  await page.click("button[type='submit']");
  await expect(page).toHaveURL("/home");
});

for (const type of ["item", "location"] as const) {
  test(`the print button uses the basic ${type} route without templates`, async ({ page }) => {
    await page.route("**/api/v1/labelmaker/**", route => route.fulfill({ body: "Printed!" }));
    const entity = await openEntity(page, type);
    const request = page.waitForRequest("**/api/v1/labelmaker/**");
    await page.getByRole("button", { name: "Print label", exact: true }).click();
    const url = new URL((await request).url());
    if (type === "item") expect(entity.assetId).toBeTruthy();
    expect(url.pathname).toBe(
      `/api/v1/labelmaker/${type === "location" ? `location/${entity.id}` : `asset/${entity.assetId}`}`
    );
    expect(url.searchParams.get("print")).toBe("true");
    expect(url.searchParams.get("tenant")).toBeTruthy();
    await expect(page.getByText("Label sent to the printer", { exact: true })).toBeVisible();
  });
}

test("basic printing stays available when a Homebox template exists", async ({ page }) => {
  await page.route(templatesURL, route =>
    route.fulfill({ json: [{ id: "test-template", name: "Existing template", width: 62, height: 29, dpi: 300 }] })
  );
  await page.route("**/api/v1/labelmaker/**", route => route.fulfill({ body: "Printed!" }));
  const entity = await openEntity(page);
  await page.getByRole("button", { name: "More print options" }).click();
  const request = page.waitForRequest("**/api/v1/labelmaker/**");
  await page.getByRole("menuitem", { name: "Basic label", exact: true }).click();
  expect(new URL((await request).url()).pathname).toBe(`/api/v1/labelmaker/asset/${entity.assetId}`);
  await expect(page.getByText("Label sent to the printer", { exact: true })).toBeVisible();
});

test("an item without an asset ID retains the entity label route", async ({ page }) => {
  await page.route("**/api/v1/labelmaker/**", route => route.fulfill({ body: "Printed!" }));
  const entity = await openEntity(page, "item", true);
  const request = page.waitForRequest("**/api/v1/labelmaker/**");
  await page.getByRole("button", { name: "Print label", exact: true }).click();
  expect(new URL((await request).url()).pathname).toBe(`/api/v1/labelmaker/entity/${entity.id}`);
  await expect(page.getByText("Label sent to the printer", { exact: true })).toBeVisible();
});

test("a failed basic print reports failure and releases the button", async ({ page }) => {
  await page.route("**/api/v1/labelmaker/**", route =>
    route.fulfill({ status: 500, body: "Label service unavailable" })
  );
  await openEntity(page);
  const button = page.getByRole("button", { name: "Print label", exact: true });
  await button.click();
  await expect(page.getByText("Failed to print label", { exact: true })).toBeVisible();
  await expect(page.getByText("Label service unavailable", { exact: true })).toBeVisible();
  await expect(button).toBeEnabled();
  await expect(page.getByText("Label sent to the printer", { exact: true })).toHaveCount(0);
});

test("a failed template load does not fall back to basic printing", async ({ page }) => {
  await page.route(templatesURL, route => route.fulfill({ status: 500, json: { message: "Unavailable" } }));
  const prints: string[] = [];
  await page.route("**/api/v1/labelmaker/**", route => {
    prints.push(route.request().url());
    return route.fulfill({ body: "Printed!" });
  });
  await openEntity(page);
  await page.getByRole("button", { name: "Print label", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  expect(prints).toHaveLength(0);
});

test("a loading template list does not fall back to basic printing", async ({ page }) => {
  let release = () => {};
  const ready = new Promise<void>(resolve => {
    release = resolve;
  });
  await page.route(templatesURL, async route => {
    await ready;
    await route.fulfill({ json: [] });
  });
  const prints: string[] = [];
  await page.route("**/api/v1/labelmaker/**", route => {
    prints.push(route.request().url());
    return route.fulfill({ body: "Printed!" });
  });
  try {
    await openEntity(page);
    await page.getByRole("button", { name: "Print label", exact: true }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    expect(prints).toHaveLength(0);
  } finally {
    release();
  }
});
