import { test, expect } from '@playwright/test';
import { UI_DEFINITIONS } from './definitions';

test.describe('Trade Recorder E2E Tests', () => {
  
  test('Verify Homepage Text and Elements', async ({ page }) => {
    await page.goto(UI_DEFINITIONS.HOME.URL);
    console.log('Current URL:', page.url());
    console.log('Page Title:', await page.title());
    console.log('Body HTML snippet:', (await page.content()).substring(0, 1000));
    
    // Wait for the main action button to appear
    await page.waitForSelector(UI_DEFINITIONS.HOME.ADD_PLAN_BTN.SELECTOR, { timeout: 10000 });

    // Verify Brand / Version
    const brand = page.locator(UI_DEFINITIONS.HOME.BRAND.SELECTOR);
    await expect(brand).toBeVisible();
    await expect(brand).toHaveText(UI_DEFINITIONS.HOME.BRAND.TEXT);

    // Verify Dashboard Link
    const dashboardLink = page.locator(UI_DEFINITIONS.HOME.DASHBOARD_LINK.SELECTOR);
    await expect(dashboardLink).toBeVisible();
    await expect(dashboardLink).toHaveText(UI_DEFINITIONS.HOME.DASHBOARD_LINK.TEXT);

    // Verify Buttons
    const addPlanBtn = page.locator(UI_DEFINITIONS.HOME.ADD_PLAN_BTN.SELECTOR);
    await expect(addPlanBtn).toContainText(UI_DEFINITIONS.HOME.ADD_PLAN_BTN.TEXT);

    const addTradeBtn = page.locator(UI_DEFINITIONS.HOME.ADD_TRADE_BTN.SELECTOR);
    await expect(addTradeBtn).toContainText(UI_DEFINITIONS.HOME.ADD_TRADE_BTN.TEXT);
  });

  test('Verify Accounts Page Text and Elements', async ({ page }) => {
    await page.goto(UI_DEFINITIONS.ACCOUNTS.URL);

    // Verify Header
    const header = page.locator(UI_DEFINITIONS.ACCOUNTS.HEADER.SELECTOR);
    await expect(header).toBeVisible();
    await expect(header).toHaveText(UI_DEFINITIONS.ACCOUNTS.HEADER.TEXT);

    // Verify Action Buttons
    const addAccBtn = page.locator(UI_DEFINITIONS.ACCOUNTS.ADD_ACCOUNT_BTN.SELECTOR);
    await expect(addAccBtn).toBeVisible();
    await expect(addAccBtn).toContainText(UI_DEFINITIONS.ACCOUNTS.ADD_ACCOUNT_BTN.TEXT);

    const importCsvBtn = page.locator(UI_DEFINITIONS.ACCOUNTS.IMPORT_CSV_BTN.SELECTOR);
    await expect(importCsvBtn).toContainText(UI_DEFINITIONS.ACCOUNTS.IMPORT_CSV_BTN.TEXT);
  });

  test('Navigate from Home to Accounts', async ({ page }) => {
    await page.goto(UI_DEFINITIONS.HOME.URL);
    
    // Click Accounts Link
    await page.click(UI_DEFINITIONS.HOME.ACCOUNTS_LINK.SELECTOR);
    
    // Check if we are on accounts page
    await expect(page).toHaveURL(UI_DEFINITIONS.ACCOUNTS.URL);
    await expect(page.locator(UI_DEFINITIONS.ACCOUNTS.HEADER.SELECTOR)).toHaveText(UI_DEFINITIONS.ACCOUNTS.HEADER.TEXT);
  });

  test('Functionality: Toggle Add Account Modal', async ({ page }) => {
    await page.goto(UI_DEFINITIONS.ACCOUNTS.URL);
    
    // Click Add Account Button
    await page.click(UI_DEFINITIONS.ACCOUNTS.ADD_ACCOUNT_BTN.SELECTOR);
    
    // Verify a modal or form appeared
    const modalTitle = page.locator(UI_DEFINITIONS.ACCOUNTS.MODAL_TITLE.SELECTOR, { hasText: UI_DEFINITIONS.ACCOUNTS.MODAL_TITLE.TEXT });
    await expect(modalTitle).toBeVisible();
    
    // Close modal (assuming there's a cancel button or '✕')
    await page.click('button:has-text("取消")');
    await expect(modalTitle).not.toBeVisible();
  });
});
