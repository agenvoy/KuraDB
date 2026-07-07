const fs = require("fs");
const path = require("path");
const { marked } = require("marked");

const OUT_DIR = path.join(__dirname, "public/docs");
const TAGS_SRC = path.join(__dirname, "public/docs/tags");
const RELEASED_DIR = path.join(OUT_DIR, "released");

function semverSort(a, b) {
  const pa = a.replace(/^v/, "").split(".").map(Number);
  const pb = b.replace(/^v/, "").split(".").map(Number);
  for (let i = 0; i < 3; i++) {
    if ((pa[i] || 0) !== (pb[i] || 0)) return (pb[i] || 0) - (pa[i] || 0);
  }
  return 0;
}

function slugify(text) {
  return text.toLowerCase()
    .replace(/[^\w\s一-鿿㐀-䶿豈-﫿-]/g, "")
    .replace(/\s+/g, "-").replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function addHeadingIds(html) {
  return html.replace(/<h([1-4])>(.*?)<\/h\1>/g, (match, level, text) => {
    const id = slugify(text.replace(/<[^>]+>/g, ""));
    return `<h${level} id="${id}">${text}</h${level}>`;
  });
}

// wrap every table so it scrolls horizontally instead of overflowing the viewport
function wrapTables(html) {
  return html
    .replace(/<table>/g, '<div class="table-scroll"><table>')
    .replace(/<\/table>/g, "</table></div>");
}

function buildTOC(html) {
  const headings = [];
  const regex = /<h([23])[^>]*id="([^"]*)"[^>]*>(.*?)<\/h\1>/g;
  let m;
  while ((m = regex.exec(html)) !== null) {
    headings.push({ depth: parseInt(m[1]), id: m[2], text: m[3].replace(/<[^>]+>/g, "") });
  }
  if (!headings.length) return `<div class="toc-title">On this page</div>`;
  let toc = `<div class="toc-title">On this page</div>\n`;
  for (const h of headings) {
    const cls = h.depth === 3 ? " depth-3" : "";
    toc += `<a class="toc-link${cls}" href="#${h.id}">${h.text}</a>\n`;
  }
  return toc;
}

function buildVersionSidebar(activeTag, tags, dates) {
  let html = "";
  const groups = new Map();
  for (const t of tags) {
    const p = t.replace(/^v/, "").split(".");
    const key = `v${p[0]}.${p[1]}`;
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(t);
  }
  for (const [minor, versions] of groups) {
    html += `<div class="nav-section">${minor}</div>\n`;
    for (const v of versions) {
      const cls = v === activeTag ? " active" : "";
      const date = dates[v] ? `<span class="nav-date">${dates[v]}</span>` : "";
      html += `<a class="nav-item${cls}" href="/docs/released/${v}">${v}${date}</a>\n`;
    }
  }
  return html;
}

function renderPage(slug, title, description, keywords, sidebar, content, toc, latestVersion) {
  const canonical = `https://kuradb.agenvoy.com/docs/${slug}`;
  const fullTitle = `${title} - KuraDB`;
  const jsonLd = JSON.stringify({
    "@context": "https://schema.org",
    "@type": "TechArticle",
    "headline": fullTitle,
    "description": description,
    "url": canonical,
    "inLanguage": "en",
    "isPartOf": { "@type": "WebSite", "name": "KuraDB", "url": "https://kuradb.agenvoy.com/" },
    "publisher": { "@type": "Person", "name": "Pardn Chiu", "url": "https://pardn.io/" },
    "image": "https://kuradb.agenvoy.com/logo-min.svg",
    "dateModified": new Date().toISOString().split("T")[0],
  });
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="robots" content="index, follow" />
    <title>${fullTitle}</title>
    <meta name="title" content="${fullTitle}" />
    <meta name="description" content="${description}" />
    <meta name="keywords" content="${keywords}" />
    <meta name="author" content="Pardn Chiu" />
    <link rel="author" href="https://pardn.io/" />
    <link rel="icon" href="/logo-min.svg" type="image/svg+xml" />
    <link rel="canonical" href="${canonical}" />
    <meta property="og:title" content="${fullTitle}" />
    <meta property="og:description" content="${description}" />
    <meta property="og:image" content="https://kuradb.agenvoy.com/logo-min.svg" />
    <meta property="og:url" content="${canonical}" />
    <meta property="og:type" content="article" />
    <meta property="og:site_name" content="KuraDB" />
    <meta property="og:locale" content="en_US" />
    <meta name="twitter:card" content="summary" />
    <meta name="twitter:title" content="${fullTitle}" />
    <meta name="twitter:description" content="${description}" />
    <meta name="twitter:image" content="https://kuradb.agenvoy.com/logo-min.svg" />
    <script type="application/ld+json">${jsonLd}</script>
    <script async src="https://www.googletagmanager.com/gtag/js?id=G-L5VYEZPVXX"></script>
    <script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments)}gtag("js",new Date());gtag("config","G-L5VYEZPVXX");</script>
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet" />
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.2/css/all.min.css" referrerpolicy="no-referrer" />
    <link rel="stylesheet" href="/docs.css" />
  </head>
  <body>
    <header class="header">
      <button class="mobile-menu-btn" onclick="document.querySelector('.sidebar').classList.toggle('open')" aria-label="Menu"><i class="fa-solid fa-bars"></i></button>
      <a href="/" class="header-logo"><picture><source media="(max-width: 480px)" srcset="/logo-min.svg" /><img src="/logo-text.svg" alt="Agenvoy" /></picture><span class="header-badge">KuraDB</span></a>
      <span class="header-sep"></span>
      <span class="header-title">Documentation</span>
      ${latestVersion ? `<a class="header-version" href="https://github.com/agenvoy/kuradb/releases/tag/${latestVersion}" target="_blank" rel="noopener">${latestVersion}</a>` : ""}
      <div class="header-links">
        <a href="/">Home</a>
        <a href="https://github.com/agenvoy/kuradb" target="_blank" rel="noopener">GitHub</a>
      </div>
    </header>
    <div class="layout">
      <nav class="sidebar">${sidebar}</nav>
      <main class="content">${content}</main>
      <aside class="toc">${toc}</aside>
    </div>
    <script>
      document.querySelectorAll('.sidebar .nav-item').forEach(function(el){
        el.addEventListener('click',function(){document.querySelector('.sidebar').classList.remove('open')})
      });
      var tocObs=new IntersectionObserver(function(entries){
        entries.forEach(function(e){
          if(e.isIntersecting){
            document.querySelectorAll('.toc-link').forEach(function(l){
              l.classList.toggle('active',l.getAttribute('href')==='#'+e.target.id)
            })
          }
        })
      },{rootMargin:'-80px 0px -70% 0px'});
      document.querySelectorAll('.content h2,.content h3').forEach(function(h){tocObs.observe(h)});
    </script>
  </body>
</html>`;
}

marked.setOptions({ gfm: true, breaks: false });

let releaseTags = [];

if (fs.existsSync(TAGS_SRC)) {
  const tagFiles = fs.readdirSync(TAGS_SRC).filter(f => f.endsWith(".md"));
  const tags = tagFiles.map(f => f.replace(".md", "")).sort(semverSort);
  const manifestPath = path.join(TAGS_SRC, "manifest.json");
  const dates = fs.existsSync(manifestPath) ? JSON.parse(fs.readFileSync(manifestPath, "utf-8")) : {};

  if (tags.length) {
    fs.mkdirSync(RELEASED_DIR, { recursive: true });
    const latestVersion = tags[0];

    for (const tag of tags) {
      const md = fs.readFileSync(path.join(TAGS_SRC, `${tag}.md`), "utf-8");
      let html = marked.parse(md);
      html = wrapTables(addHeadingIds(html));
      const sidebar = buildVersionSidebar(tag, tags, dates);
      const toc = buildTOC(html);
      const desc = `KuraDB ${tag} release notes — changelog, new features, and fixes.`;
      const kw = `kuradb, release notes, changelog, ${tag}`;
      const page = renderPage(`released/${tag}`, `${tag} Release Notes`, desc, kw, sidebar, html, toc, latestVersion);
      fs.writeFileSync(path.join(RELEASED_DIR, `${tag}.html`), page);
      releaseTags.push(tag);
    }

    // Index page
    let listHtml = "<h1>Release Notes</h1>\n<p>All KuraDB releases.</p>\n";
    const groups = new Map();
    for (const t of tags) {
      const p = t.replace(/^v/, "").split(".");
      const key = `v${p[0]}.${p[1]}`;
      if (!groups.has(key)) groups.set(key, []);
      groups.get(key).push(t);
    }
    for (const [minor, versions] of groups) {
      listHtml += `<h2>${minor}</h2>\n<ul>\n`;
      for (const v of versions) {
        const date = dates[v] ? ` <span style="color:var(--muted);font-size:13px">${dates[v]}</span>` : "";
        listHtml += `<li><a href="/docs/released/${v}">${v}</a>${date}</li>\n`;
      }
      listHtml += "</ul>\n";
    }
    listHtml = addHeadingIds(listHtml);
    const indexSidebar = buildVersionSidebar("", tags, dates);
    const indexToc = buildTOC(listHtml);
    const indexPage = renderPage("released", "Release Notes", "All KuraDB release notes — changelogs, features, and fixes by version.", "kuradb, releases, changelog, version history", indexSidebar, listHtml, indexToc, latestVersion);
    fs.writeFileSync(path.join(RELEASED_DIR, "index.html"), indexPage);

    console.log(`OK: ${releaseTags.length} release pages + index`);
  }
} else {
  console.warn(`SKIP: ${TAGS_SRC} not found — run "npm run sync-tags" first`);
}

// Generate sitemap.xml
const today = new Date().toISOString().split("T")[0];
let sitemap = `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n`;
sitemap += `  <url><loc>https://kuradb.agenvoy.com/</loc><changefreq>weekly</changefreq><priority>1.0</priority><lastmod>${today}</lastmod></url>\n`;
sitemap += `  <url><loc>https://kuradb.agenvoy.com/zh/</loc><changefreq>weekly</changefreq><priority>0.9</priority><lastmod>${today}</lastmod></url>\n`;
if (releaseTags.length) {
  sitemap += `  <url><loc>https://kuradb.agenvoy.com/docs/released/</loc><changefreq>weekly</changefreq><priority>0.6</priority><lastmod>${today}</lastmod></url>\n`;
  for (let i = 0; i < releaseTags.length; i++) {
    const pri = i < 5 ? 0.5 : 0.3;
    sitemap += `  <url><loc>https://kuradb.agenvoy.com/docs/released/${releaseTags[i]}</loc><changefreq>yearly</changefreq><priority>${pri}</priority><lastmod>${today}</lastmod></url>\n`;
  }
}
sitemap += `</urlset>\n`;
fs.writeFileSync(path.join(__dirname, "public/sitemap.xml"), sitemap);
console.log(`OK: sitemap.xml (${2 + (releaseTags.length ? releaseTags.length + 1 : 0)} URLs)`);
