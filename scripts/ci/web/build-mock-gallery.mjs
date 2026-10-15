import { copyFile, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { layouts, pages } from './mock-page-catalog.mjs';

const root = resolve(process.env.CI_ARTIFACTS_DIR || 'mock-page-artifacts');
const site = resolve(root, 'site');
const captures = pages.flatMap(page => page.captures.map(([id]) => id));

for (const [layout, viewport] of Object.entries(layouts)) {
  const manifest = JSON.parse(await readFile(resolve(root, layout, 'manifest.json'), 'utf8'));
  const complete = JSON.stringify(manifest.viewport) === JSON.stringify(viewport)
    && captures.every(id => manifest.pages.includes(id));
  if (!complete) throw new Error(`Incomplete ${layout} manifest`);
}

await rm(site, { recursive: true, force: true });
for (const layout of Object.keys(layouts)) {
  await mkdir(resolve(site, layout), { recursive: true });
  for (const id of captures) {
    await copyFile(resolve(root, layout, `${id}.png`), resolve(site, layout, `${id}.png`));
  }
}

const esc = value => String(value).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
const cards = pages.map((page, i) => {
  const shots = page.captures.map(([id, title]) => `<section class="capture"><h3>${esc(title)}</h3><div class="shots">${Object.entries(layouts).map(([layout, viewport]) => `<figure><figcaption>${layout === 'mobile' ? '手机' : '桌面'} · ${viewport.width}×${viewport.height}</figcaption><a href="${layout}/${id}.png"><img src="${layout}/${id}.png" loading="lazy" alt="${esc(page.title)}：${esc(title)}，${layout}"></a></figure>`).join('')}</div></section>`).join('');
  return `<article id="${page.id}" data-search="${esc([page.title, page.description, ...page.features].join(' '))}"><header><span>${String(i + 1).padStart(2, '0')}</span><div><small>${esc(page.group)} · ${esc(page.id)}</small><h2>${esc(page.title)}</h2></div></header><p>${esc(page.description)}</p><p><strong>进入：</strong>${esc(page.entry)}</p><ul>${page.features.map(x => `<li>${esc(x)}</li>`).join('')}</ul>${shots}</article>`;
}).join('');

const html = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Proxy-Go · Mock 功能与截图</title><style>
:root{font-family:Inter,system-ui,sans-serif;color:#171717;background:#f7f7f5}*{box-sizing:border-box}body{margin:0}aside{position:fixed;inset:0 auto 0 0;width:240px;padding:28px 20px;background:#111;color:white}aside h1{font-size:22px}aside p{color:#aaa;font-size:12px}aside nav a{display:block;color:#ddd;text-decoration:none;padding:6px 0;font-size:13px}main{margin-left:240px;max-width:1500px;padding:54px 52px}.hero{max-width:800px;padding-bottom:36px}.hero h1{font-size:clamp(34px,5vw,64px);line-height:1.1;margin:14px 0}.hero p{color:#555;line-height:1.8}.toolbar{padding:16px;background:white;border:1px solid #ddd;border-radius:8px;margin-bottom:24px}.toolbar input{padding:9px;width:min(100%,360px);border:1px solid #ccc;border-radius:6px}article{background:white;border:1px solid #ddd;border-radius:8px;padding:26px;margin:0 0 26px;scroll-margin-top:20px}article header{display:flex;align-items:center;gap:14px}article header>span{font-size:20px;color:#777}h2{margin:0;font-size:26px}small{color:#777}article>p{color:#555;line-height:1.7}li{margin:5px 0;color:#444}.capture{border-top:1px solid #e5e5e5;margin-top:24px;padding-top:18px}.capture h3{font-size:16px;margin:0 0 14px}.shots{display:grid;grid-template-columns:260px minmax(0,1fr);gap:22px}figure{margin:0}figcaption{font-size:12px;color:#666;margin-bottom:8px}figure a{display:block;border:1px solid #ddd;border-radius:6px;overflow:hidden;background:#f5f5f3}figure img{display:block;width:100%;height:auto}footer{color:#888;font-size:12px;padding:20px 0}@media(max-width:760px){aside{position:static;width:auto;padding:18px}aside nav{display:flex;gap:12px;overflow:auto}main{margin:0;padding:26px 16px}.shots{grid-template-columns:1fr}.shots figure:first-child{max-width:390px;margin:auto}article{padding:18px}.hero h1{font-size:40px}}
</style></head><body><aside><h1>Proxy-Go</h1><p>Mock 功能与 CI 截图</p><nav>${pages.map(page => `<a href="#${page.id}">${esc(page.title)}</a>`).join('')}</nav></aside><main><section class="hero"><small>Proxy-Go / Mock Debug Gallery</small><h1>代理基础设施管理</h1><p>浏览 Proxy-Go 各模块和关键操作的手机、桌面截图。CI 使用隔离的 Mock API 执行页面操作、检查浏览器错误并生成图库。</p><p><strong>部署方式：</strong>在 GitHub 仓库 Settings → Pages → Build and deployment 中将 Source 设为 GitHub Actions。此后每次推送 main，工作流会自动构建、截图并部署最新页面。</p></section><div class="toolbar"><label>查找模块 <input id="search" placeholder="名称或功能"></label> <span id="count">${pages.length} 个模块</span></div><section id="pages">${cards}</section><footer>截图版本：${esc((process.env.GALLERY_SHA || 'local').slice(0, 12))} · 由 GitHub Actions 生成</footer></main><script>const cards=[...document.querySelectorAll('article')],search=document.querySelector('#search'),count=document.querySelector('#count');search.addEventListener('input',()=>{const q=search.value.toLowerCase();let n=0;cards.forEach(card=>{card.hidden=!card.dataset.search.toLowerCase().includes(q);if(!card.hidden)n++});count.textContent=n+' 个模块'});</script></body></html>`;
await writeFile(resolve(site, 'index.html'), html, 'utf8');
await writeFile(resolve(site, '.nojekyll'), '');
console.log(`Generated gallery at ${site}`);
