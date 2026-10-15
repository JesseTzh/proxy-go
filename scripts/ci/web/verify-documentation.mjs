import { chromium, expect } from '@playwright/test';
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { extname, resolve, sep } from 'node:path';
import { pages } from './mock-page-catalog.mjs';

const root = resolve(process.env.CI_ARTIFACTS_DIR || 'mock-page-artifacts');
const site = resolve(root, 'site');
const server = createServer(async (request, response) => {
  try {
    const pathname = decodeURI(new URL(request.url, 'http://localhost').pathname);
    if (!pathname.startsWith('/proxy-go/')) throw new Error('Outside project');
    const path = resolve(site, '.' + pathname.slice('/proxy-go'.length).replace(/\/$/, '/index.html'));
    if (!path.startsWith(site + sep)) throw new Error('Outside site');
    const data = await readFile(path);
    response.writeHead(200, { 'Content-Type': ({ '.html': 'text/html; charset=utf-8', '.css': 'text/css', '.png': 'image/png' })[extname(path)] || 'application/octet-stream' });
    response.end(data);
  } catch { response.writeHead(404); response.end('Not found'); }
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
const base = `${origin}/proxy-go`;
let browser;
try {
  browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1440, height: 1080 } });
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => { if (response.status() >= 400) errors.push(`${response.status()} ${response.url()}`); });
  const files = ['index.html', 'development.html', 'release.html', 'design.html', ...pages.map(page => `${page.id}.html`)];
  const anchors = new Map();
  const targets = [];
  const assets = new Set();
  for (const file of files) {
    await page.goto(`${base}/${file}`, { waitUntil: 'networkidle' });
    await expect(page.locator('main h1')).toHaveCount(1);
    await expect(page.locator('nav[aria-label="文档导航"] a[aria-current="page"]')).toHaveAttribute('href', file);
    const state = await page.evaluate(() => ({
      ids: [...document.querySelectorAll('[id]')].map(el => el.id),
      links: [...document.querySelectorAll('a[href]')].map(el => el.href),
      images: [...document.images].map(el => el.src),
      missingTestIds: [...document.body.querySelectorAll('*')].filter(el => !el.hasAttribute('data-testid')).map(el => el.tagName),
    }));
    expect(state.missingTestIds).toEqual([]);
    expect(new Set(state.ids).size).toBe(state.ids.length);
    anchors.set(file, state.ids);
    for (const href of state.links) {
      const url = new URL(href);
      if (url.origin !== origin) continue;
      if (url.pathname.endsWith('.html')) targets.push(url);
      else assets.add(url.href);
    }
    for (const src of state.images) { expect(src).not.toContain('/mobile/'); if (src.startsWith(base)) assets.add(src); }
    if (file === 'index.html') {
      const readme = await readFile('README.md', 'utf8');
      await expect(page.locator('main h1')).toHaveText(readme.match(/^#\s+(.+)$/m)[1]);
      await expect(page.locator('main .capture')).toHaveCount(0);
      await expect(page.locator('main pre code')).not.toHaveCount(0);
      await expect(page.locator('main a[href="development.html"]')).toHaveCount(1);
      await expect(page.locator('.pagination a')).toHaveAttribute('href', 'login.html');
      await page.screenshot({ path: resolve(root, 'documentation-home.png'), fullPage: true });
    }
    if (file === 'login.html') {
      await expect(page.locator('main img')).toHaveCount(1);
      await expect(page.locator('.pagination a').last()).toHaveAttribute('href', 'dashboard.html');
      await page.screenshot({ path: resolve(root, 'documentation-login.png'), fullPage: true });
    }
    await page.setViewportSize({ width: 390, height: 844 });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.setViewportSize({ width: 1440, height: 1080 });
  }
  for (const url of targets) {
    const file = url.pathname.slice('/proxy-go/'.length);
    expect(anchors.has(file), `Missing page ${url}`).toBe(true);
    if (url.hash) expect(anchors.get(file), `Missing anchor ${url}`).toContain(decodeURIComponent(url.hash.slice(1)));
  }
  for (const url of assets) expect((await page.request.get(url)).ok(), `Missing asset ${url}`).toBe(true);
  expect(errors).toEqual([]);
  console.log(`Verified ${files.length} documentation pages, navigation, anchors, desktop images and responsive layout`);
} finally {
  if (browser) await browser.close();
  await new Promise(resolve => server.close(resolve));
}
