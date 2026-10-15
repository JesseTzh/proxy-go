import { chromium, expect } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { layouts, pages } from './mock-page-catalog.mjs';

const root = resolve(process.env.CI_ARTIFACTS_DIR || 'mock-page-artifacts');
const baseURL = process.env.CI_BASE_URL || 'http://127.0.0.1:4173';
const browser = await chromium.launch({ headless: true });
try {
  for (const [layout, viewport] of Object.entries(layouts)) {
    const dir = resolve(root, layout); await mkdir(dir, { recursive: true });
    const context = await browser.newContext({ viewport, locale: 'zh-CN', timezoneId: 'Asia/Shanghai', reducedMotion: 'reduce' });
    const page = await context.newPage();
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    page.on('requestfailed', request => errors.push(`${request.url()}: ${request.failure()?.errorText}`));
    await page.goto(baseURL, { waitUntil: 'networkidle' });
    await expect(page.getByTestId('dashboard-page')).toBeVisible();
    const hideDebug = await page.addStyleTag({ content: '[data-testid="mock-debug-panel"]{display:none!important}' });
    const captured = [];
    async function capture(id, ready = page.locator('main')) { await expect(ready).toBeVisible(); await page.screenshot({ path: resolve(dir, `${id}.png`), fullPage: true }); captured.push(id); }
    await page.getByTestId('dashboard-logout-button').click();

    await capture('login', page.getByTestId('login-page'));
    await page.getByTestId('login-password-input').fill('mock-password');
    await page.getByTestId('login-submit-button').click();
    await capture('dashboard', page.getByTestId('dashboard-page'));
    await page.getByTestId('dashboard-process-sing-box-logs').click(); await capture('dashboard-logs', page.getByTestId('dashboard-logs-dialog')); await page.keyboard.press('Escape');
    await page.getByTestId('dashboard-process-nginx-config').click(); await capture('dashboard-config', page.getByTestId('dashboard-nginx-config-dialog')); await page.keyboard.press('Escape');
    const routes = [['domains', 'domains-page'], ['reverse', 'reverse-proxies-page'], ['vless', 'inbounds-page'], ['settings', 'settings-page'], ['audit', 'audit-page']];
    for (const [nav, ready] of routes) { await page.getByTestId(`nav-${nav}`).click(); await capture(nav, page.getByTestId(ready)); }
    await page.getByTestId('nav-domains').click(); await page.getByTestId('domain-create-input').fill('new.example.com'); await capture('domains-create'); await page.getByTestId('domain-create-button').click(); await expect(page.getByTestId('domain-row-20')).toBeVisible(); await page.getByTestId('domain-dns-1').click(); await capture('domains-dns');
    await page.getByTestId('nav-reverse').click(); await page.getByTestId('reverse-edit-1').click(); await capture('reverse-edit');
    await page.getByTestId('nav-vless').click(); await page.getByTestId('inbound-create-button').click(); await capture('vless-create', page.getByTestId('inbound-dialog')); await page.keyboard.press('Escape'); await page.getByTestId('inbound-config-1').click(); await capture('vless-config', page.getByTestId('inbound-config-dialog')); await page.keyboard.press('Escape');
    await page.getByTestId('inbound-show-qr-1').click(); await capture('vless-share', page.getByTestId('inbound-share-dialog')); await page.keyboard.press('Escape');
    await page.getByTestId('nav-settings').click(); await page.getByTestId('settings-runtime-tab').click(); await capture('settings-runtime');
    await page.getByTestId('nav-audit').click(); await page.getByTestId('audit-detail-1').click(); await capture('audit-detail', page.getByTestId('audit-detail-dialog')); await page.keyboard.press('Escape');
    await page.getByTestId('nav-dashboard').click(); await page.getByTestId('dashboard-logout-button').click(); await expect(page.getByTestId('login-page')).toBeVisible();
    await hideDebug.evaluate(element => element.remove());
    await page.getByTestId('mock-debug-open').click(); await capture('mock-debug', page.getByTestId('mock-debug-card'));
    if (errors.length) throw new Error(`Browser errors in ${layout}: ${errors.join('\n')}`);
    await writeFile(resolve(dir, 'manifest.json'), JSON.stringify({ layout, viewport, pages: captured }, null, 2));
    await writeFile(resolve(dir, 'browser.log'), 'No browser errors\n');
    await context.close();
  }
} finally { await browser.close(); }
