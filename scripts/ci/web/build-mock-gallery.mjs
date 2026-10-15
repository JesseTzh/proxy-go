import { copyFile, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { dirname, posix, resolve } from 'node:path';
import { Marked } from 'marked';
import sanitizeHtml from 'sanitize-html';
import { layouts, pages } from './mock-page-catalog.mjs';

const root = resolve(process.env.CI_ARTIFACTS_DIR || 'mock-page-artifacts');
const site = resolve(root, 'site');
const repository = process.env.GITHUB_REPOSITORY || 'JesseTzh/proxy-go';
const revision = process.env.GALLERY_SHA || 'main';
const sourceURL = `https://github.com/${repository}/blob/${encodeURIComponent(revision)}/`;
const documents = [
  { source: 'README.md', file: 'index.html', title: '项目介绍', description: 'README' },
  { source: 'docs/DEVELOPMENT.md', file: 'development.html', title: '开发说明' },
  { source: 'docs/RELEASE.md', file: 'release.html', title: '部署与发布' },
  { source: 'web/DESIGN.md', file: 'design.html', title: '前端设计' },
];
const chapters = pages.map(page => ({ ...page, file: `${page.id}.html` }));
const captures = pages.flatMap(page => page.captures.map(([id]) => id));
const esc = value => String(value).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);

for (const [layout, viewport] of Object.entries(layouts)) {
  const manifest = JSON.parse(await readFile(resolve(root, layout, 'manifest.json'), 'utf8'));
  if (JSON.stringify(manifest.viewport) !== JSON.stringify(viewport) || !captures.every(id => manifest.pages.includes(id))) {
    throw new Error(`Incomplete ${layout} manifest`);
  }
}
await rm(site, { recursive: true, force: true });
await mkdir(resolve(site, 'desktop'), { recursive: true });
for (const id of captures) await copyFile(resolve(root, 'desktop', `${id}.png`), resolve(site, 'desktop', `${id}.png`));
await copyFile(new URL('./docs.css', import.meta.url), resolve(site, 'docs.css'));

// Resolve README/document links relative to their source, including Pages project subpaths.
function rewriteURL(href, source, image = false) {
  if (!href || /^(?:[a-z][a-z\d+.-]*:|\/\/|#)/i.test(href)) return href;
  const match = href.match(/^([^?#]*)(.*)$/);
  const path = posix.normalize(posix.join(dirname(source), decodeURI(match[1])));
  const doc = documents.find(document => document.source === path);
  if (doc && !image) return doc.file + match[2];
  return (image ? `https://raw.githubusercontent.com/${repository}/${encodeURIComponent(revision)}/` : sourceURL) + path + match[2];
}
function renderMarkdown(markdown, source) {
  const headings = [];
  const slugs = new Map();
  const parser = new Marked({
    gfm: true,
    walkTokens(token) {
      if (token.type === 'link' || token.type === 'image') token.href = rewriteURL(token.href, source, token.type === 'image');
    },
    renderer: {
      heading({ tokens, depth }) {
        const title = this.parser.parseInline(tokens);
        const plain = sanitizeHtml(title, { allowedTags: [], allowedAttributes: {} });
        const slug = plain.toLowerCase().replace(/<[^>]*>/g, '').replace(/[^\p{L}\p{N}\p{M}_\-\s]/gu, '').replace(/\s/g, '-');
        const n = slugs.get(slug) || 0;
        slugs.set(slug, n + 1);
        const id = n ? `${slug}-${n}` : slug;
        if (depth <= 3) headings.push({ id, title: plain, depth });
        return `<h${depth} id="${esc(id)}">${title}</h${depth}>`;
      },
    },
  });
  return {
    html: sanitizeHtml(parser.parse(markdown), {
      allowedTags: sanitizeHtml.defaults.allowedTags.concat(['img', 'details', 'summary', 'input']),
      allowedAttributes: { ...sanitizeHtml.defaults.allowedAttributes, '*': ['id'], code: ['class'], img: ['src', 'alt', 'title', 'width', 'height'], input: ['type', 'checked', 'disabled'] },
    }),
    headings,
  };
}
const link = (item, current) => `<a href="${item.file}"${item.file === current.file ? ' aria-current="page"' : ''}>${esc(item.title)}</a>`;
function shell(current, content, headings, sequence = []) {
  const index = sequence.findIndex(item => item.file === current.file);
  const previous = sequence[index - 1];
  const next = sequence[index + 1];
  const html = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${esc(current.title)} · Proxy-Go 文档</title><link rel="stylesheet" href="docs.css"></head><body>
<a class="skip" href="#content">跳到正文</a>
<aside class="sidebar"><a class="brand" href="index.html">Proxy-Go <span>文档</span></a><p class="tagline">代理网关 · 部署与使用指南</p><nav aria-label="文档导航"><div class="nav-label">开始</div>${link(documents[0], current)}<div class="nav-label">使用指南</div>${chapters.filter(page => page.id !== 'mock-debug').map(page => link(page, current)).join('')}<div class="nav-label">开发与部署</div>${documents.slice(1).map(doc => link(doc, current)).join('')}${link(chapters.find(page => page.id === 'mock-debug'), current)}</nav><a class="repository" href="https://github.com/${repository}">GitHub 仓库 ↗</a></aside>
<div class="workspace"><header class="topbar"><span>Proxy-Go / ${esc(current.title)}</span><a href="${sourceURL}${current.source || 'web/src/app/navigation.ts'}">查看源文件 ↗</a></header><div class="document-layout"><main id="content" tabindex="-1"><article class="prose">${content}</article>${previous || next ? `<nav class="pagination" aria-label="章节翻页">${previous ? `<a href="${previous.file}"><small>上一章</small>${esc(previous.title)} ←</a>` : '<span></span>'}${next ? `<a href="${next.file}"><small>下一章</small>${esc(next.title)} →</a>` : ''}</nav>` : ''}<footer>基于当前仓库文档与桌面演示截图构建 · <code>${esc(revision.slice(0, 12))}</code></footer></main><aside class="toc"><nav aria-label="本页目录"><div class="nav-label">本页内容</div>${headings.map(h => `<a class="depth-${h.depth}" href="#${esc(h.id)}">${esc(h.title)}</a>`).join('')}</nav></aside></div></div></body></html>`;
  // Include rendered Markdown elements in the same stable locator convention as the app.
  let element = 0;
  return html.replace(/<([a-z][a-z\d-]*)(?=[\s>])/gi, (tag, name) => `${tag} data-testid="docs-${current.file.replace('.html', '')}-${name}-${++element}"`);
}
const sequence = [documents[0], ...chapters.filter(page => page.id !== 'mock-debug')];
for (const doc of documents) {
  const markdown = await readFile(resolve(doc.source), 'utf8');
  const { html, headings } = renderMarkdown(markdown, doc.source);
  await writeFile(resolve(site, doc.file), shell(doc, html, headings, sequence));
}
for (const page of chapters) {
  const content = `<p class="eyebrow">${esc(page.group)} / 使用指南</p><h1>${esc(page.title)}</h1><p class="lead">${esc(page.description)}</p><h2 id="entry">进入页面</h2><p>${esc(page.entry)}</p><h2 id="features">功能与操作</h2><ol>${page.features.map(feature => `<li>${esc(feature)}</li>`).join('')}</ol><h2 id="screenshots">界面与操作示例</h2><p class="note">以下为桌面演示环境截图，使用示例数据。点击图片查看原图。</p>${page.captures.map(([id, title]) => `<section class="capture"><h3 id="${id}">${esc(title)}</h3><figure><a href="desktop/${id}.png"><img src="desktop/${id}.png" loading="lazy" width="1440" alt="${esc(page.title)} — ${esc(title)}"></a><figcaption>${esc(title)}</figcaption></figure></section>`).join('')}`;
  const headings = [{ id: 'entry', title: '进入页面', depth: 2 }, { id: 'features', title: '功能与操作', depth: 2 }, { id: 'screenshots', title: '界面与操作示例', depth: 2 }, ...page.captures.map(([id, title]) => ({ id, title, depth: 3 }))];
  await writeFile(resolve(site, page.file), shell(page, content, headings, sequence));
}
await writeFile(resolve(site, '.nojekyll'), '');
console.log(`Generated project documentation at ${site}`);
