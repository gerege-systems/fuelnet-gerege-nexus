/**
 * Builds the published documentation site from the Markdown already in this
 * repository.
 *
 *   node build.mjs        → docs/site/dist
 *
 * The documents are the source of truth and stay readable on GitHub; this
 * script only wraps them in a shell and rewrites the links. Nothing here should
 * ever require editing a document to keep the site working — a document that
 * renders on GitHub renders here.
 *
 * Link rewriting is the whole trick. A link in a Markdown file is relative to
 * that file, and the site is flat, so every href is resolved against its source
 * file and then re-pointed at one of three places:
 *
 *   - another published page  → its slug on this site
 *   - an asset under docs/    → the copied asset
 *   - anything else in the repo (LICENSE, .env.example, Go source) → GitHub
 *
 * That last case is why the site can link to code it does not publish.
 */
import {cpSync, mkdirSync, readFileSync, rmSync, writeFileSync} from "node:fs";
import {dirname, join, posix, relative, resolve} from "node:path";
import {fileURLToPath} from "node:url";

import {Marked} from "marked";

const HERE = dirname(fileURLToPath(import.meta.url));
const REPO = resolve(HERE, "..", "..");
const OUT = join(HERE, "dist");

const GITHUB = "https://github.com/gerege-systems/open-gerege-nexus";
const BLOB = `${GITHUB}/blob/main`;

/* ── What gets published ──────────────────────────────────────────────────── */

/**
 * Every page on the site, in sidebar order. `group` buckets them in the
 * sidebar; `lang` marks the seven translations of the overview so they can be
 * offered as a language row rather than seven sidebar entries.
 *
 * Deliberately absent: APPSTORE_PHASE2_PLAN, APPSTORE_SEPARATION_PLAN and
 * DOCUMENTS_WORKLOG. Those are working notes for a migration in progress, not
 * documentation of what the platform does, and publishing them would date the
 * site the moment the work lands.
 */
const PAGES = [
  // The overview is the site's front door. There is no separate showcase page:
  // the product's own landing page makes that argument, and saying it twice
  // meant two places to keep true.
  {src: "README.md", slug: "index", title: "Тойм", group: "Танилцуулга", lang: "mn"},
  {src: "docs/README_EN.md", slug: "overview-en", title: "Overview", group: "Танилцуулга", lang: "en"},
  {src: "docs/README_AR.md", slug: "overview-ar", title: "نظرة عامة", group: "Танилцуулга", lang: "ar", rtl: true},
  {src: "docs/README_ZH.md", slug: "overview-zh", title: "概览", group: "Танилцуулга", lang: "zh"},
  {src: "docs/README_FR.md", slug: "overview-fr", title: "Aperçu", group: "Танилцуулга", lang: "fr"},
  {src: "docs/README_RU.md", slug: "overview-ru", title: "Обзор", group: "Танилцуулга", lang: "ru"},
  {src: "docs/README_ES.md", slug: "overview-es", title: "Resumen", group: "Танилцуулга", lang: "es"},

  {src: "docs/README.md", slug: "documents", title: "Баримтын индекс", group: "Архитектур"},
  {src: "docs/ARCHITECTURE_SPECIFICATION.md", slug: "architecture", title: "Архитектурын тодорхойлолт", group: "Архитектур"},
  {src: "docs/ARCHITECTURE_SPECIFICATION_EN.md", slug: "architecture-en", title: "Architecture specification (EN)", group: "Архитектур"},

  {src: "docs/SSO_FEDERATION.md", slug: "sso-federation", title: "SSO холбоос", group: "Архитектур"},

  // The strategy and the decision record. They are published because the ADRs
  // are what a reader is sent to when they ask why the code is shaped this way,
  // and because both link to the strategy — a published page linking to an
  // unpublished one leaves the site.
  {src: "docs/ECOSYSTEM_GIT_STRATEGY.md", slug: "ecosystem-git-strategy", title: "Экосистемийн git стратеги", group: "Архитектур"},
  {src: "docs/adr/0001-domain-first.md", slug: "adr-0001-domain-first", title: "ADR 0001 — Домэйн эхэнд", group: "Архитектур"},
  {src: "docs/adr/0002-one-signing-rail.md", slug: "adr-0002-one-signing-rail", title: "ADR 0002 — Гарын үсгийн зам нэг", group: "Архитектур"},
  {src: "docs/adr/0003-a-document-carries-what-is-signed.md", slug: "adr-0003-document-carries", title: "ADR 0003 — Баримт файлаа авч явна", group: "Архитектур"},

  {src: "docs/MODULE_AUTHORING_GUIDE.md", slug: "module-authoring", title: "Модуль хөгжүүлэх заавар", group: "Хөгжүүлэлт"},
  {src: "docs/TRANSLATION_GUIDE.md", slug: "translation", title: "Орчуулгын гарын авлага", group: "Хөгжүүлэлт"},
  {src: "docs/APPSTORE_OPERATIONS.md", slug: "appstore-operations", title: "Апп сторын ажиллагаа", group: "Хөгжүүлэлт"},

  {src: "docs/GOV_SERVICES_WORKFLOW.md", slug: "gov-services", title: "Төрийн үйлчилгээний урсгал", group: "Модулиуд"},
  {src: "docs/DOCUMENTS_SIGNING.md", slug: "documents-signing", title: "Цахим гарын үсэг", group: "Модулиуд"},
  {src: "docs/REPORTS.md", slug: "reports", title: "Тайлангийн хөдөлгүүр", group: "Модулиуд"},
  {src: "docs/REPORT_SHARING.md", slug: "report-sharing", title: "Тенант дамнасан тайлан", group: "Модулиуд"},

  // Operations. MONITORING and RUNBOOKS are read at different moments — one
  // when setting the stack up, the other at three in the morning — so they are
  // two entries rather than one long page. The proposal is here because both
  // link to it for the reasoning behind what they describe, and a link from a
  // published page to an unpublished one leaves the site.
  {src: "docs/MONITORING.md", slug: "monitoring", title: "Мониторинг", group: "Ажиллагаа"},
  {src: "docs/RUNBOOKS.md", slug: "runbooks", title: "Runbook-ууд", group: "Ажиллагаа"},
  {src: "docs/MONITORING_AND_REPORTING_PROPOSAL.md", slug: "monitoring-proposal", title: "Ажиглалт ба тайлангийн санал", group: "Ажиллагаа"},

  {src: "docs/SHELL_CONTRACT.md", slug: "shell-contract", title: "Bridge гэрээ", group: "Native клиентүүд"},
  {src: "docs/NATIVE_LOGIN_SPEC.md", slug: "native-login", title: "Native нэвтрэлт", group: "Native клиентүүд"},
  {src: "docs/NATIVE_SETTINGS_SPEC.md", slug: "native-settings", title: "Native тохиргоо", group: "Native клиентүүд"},

  {src: "CONTRIBUTING.md", slug: "contributing", title: "Хувь нэмэр оруулах", group: "Төслийн журам"},
  {src: "docs/CONTRIBUTING_EN.md", slug: "contributing-en", title: "Contributing (EN)", group: "Төслийн журам"},
  {src: "SECURITY.md", slug: "security", title: "Аюулгүй байдал", group: "Төслийн журам"},
  {src: "docs/SECURITY_EN.md", slug: "security-en", title: "Security policy (EN)", group: "Төслийн журам"},
  {src: "CODE_OF_CONDUCT.md", slug: "code-of-conduct", title: "Ёс зүйн дүрэм", group: "Төслийн журам"},
  {src: "docs/CODE_OF_CONDUCT_EN.md", slug: "code-of-conduct-en", title: "Code of conduct (EN)", group: "Төслийн журам"},
  {src: "CHANGELOG.md", slug: "changelog", title: "Өөрчлөлтийн түүх", group: "Төслийн журам"},
];

const bySrc = new Map(PAGES.map((p) => [p.src, p]));
const LANGS = [
  {lang: "mn", label: "Монгол", flag: "flag-mn.png"},
  {lang: "ar", label: "العربية", flag: "flag-ar.png"},
  {lang: "zh", label: "中文", flag: "flag-zh.png"},
  {lang: "en", label: "English", flag: "flag-en.png"},
  {lang: "fr", label: "Français", flag: "flag-fr.png"},
  {lang: "ru", label: "Русский", flag: "flag-ru.png"},
  {lang: "es", label: "Español", flag: "flag-es.png"},
];

/* ── Markdown → HTML ──────────────────────────────────────────────────────── */

const slugged = new Map();

/** A stable, collision-free id for a heading, so the sidebar can link into it. */
function headingId(text) {
  const base =
    text
      .toLowerCase()
      .replace(/<[^>]+>/g, "")
      .replace(/[^\p{L}\p{N}]+/gu, "-")
      .replace(/^-+|-+$/g, "") || "section";
  const seen = slugged.get(base) ?? 0;
  slugged.set(base, seen + 1);
  return seen ? `${base}-${seen}` : base;
}

/**
 * Re-points one href from "relative to this Markdown file" to "correct on this
 * site". Anchors, mailto: and absolute URLs are already right and pass through.
 */
function rewrite(href, srcPath) {
  if (!href || /^(https?:|mailto:|#|data:)/.test(href)) return href;

  const [pathPart, hash = ""] = href.split("#");
  if (!pathPart) return href;

  const repoRel = posix.normalize(posix.join(posix.dirname(srcPath), pathPart)).replace(/^\.\//, "");
  const anchor = hash ? `#${hash}` : "";

  const page = bySrc.get(repoRel);
  if (page) return `${page.slug}.html${anchor}`;
  // Assets are copied and served; a Markdown file sitting among them is prose,
  // not an asset, and a browser would download it rather than render it — so it
  // goes to GitHub with the rest of the unpublished files.
  if (repoRel.startsWith("docs/assets/") && !repoRel.endsWith(".md")) {
    return repoRel.slice("docs/".length) + anchor;
  }
  return `${BLOB}/${repoRel}${anchor}`;
}

function render(markdown, srcPath) {
  slugged.clear();
  const headings = [];
  const md = new Marked({gfm: true, breaks: false});

  md.use({
    renderer: {
      heading({tokens, depth}) {
        const text = this.parser.parseInline(tokens);
        const id = headingId(text);
        if (depth === 2 || depth === 3) headings.push({id, text, depth});
        return `<h${depth} id="${id}"><a class="anchor" href="#${id}" aria-hidden="true"></a>${text}</h${depth}>\n`;
      },
      link({href, title, tokens}) {
        const text = this.parser.parseInline(tokens);
        const target = rewrite(href, srcPath);
        const external = /^https?:/.test(target);
        return `<a href="${target}"${title ? ` title="${title}"` : ""}${
          external ? ' target="_blank" rel="noopener"' : ""
        }>${text}</a>`;
      },
      image({href, title, text}) {
        return `<img src="${rewrite(href, srcPath)}" alt="${text ?? ""}"${title ? ` title="${title}"` : ""}>`;
      },
      table(token) {
        // Wrapped so a wide table scrolls inside the column instead of pushing
        // the whole page sideways on a phone.
        const rendered = md.Renderer.prototype.table.call(this, token);
        return `<div class="table-scroll">${rendered}</div>`;
      },
    },
  });

  // Raw HTML in the source (the flag rows) carries hrefs marked has no reason
  // to touch, so they are rewritten here.
  let html = md.parse(markdown);
  html = html.replace(/(<a\s[^>]*href=")([^"]+)(")/g, (m, a, href, b) => a + rewrite(href, srcPath) + b);
  html = html.replace(/(<img\s[^>]*src=")([^"]+)(")/g, (m, a, src, b) => a + rewrite(src, srcPath) + b);

  return {html, headings};
}

/* ── Page shell ───────────────────────────────────────────────────────────── */

const esc = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");

function sidebar(current) {
  const groups = new Map();
  for (const p of PAGES) {
    // The seven overview translations collapse to one entry; the rest of the
    // languages are reachable from the language row on the page itself.
    if (p.lang && p.lang !== "mn") continue;
    if (!groups.has(p.group)) groups.set(p.group, []);
    groups.get(p.group).push(p);
  }
  const sections = [...groups]
    .map(([group, pages]) => {
      const items = pages
        .map((p) => {
          const active = p.slug === current || (p.lang && PAGES.find((q) => q.slug === current)?.lang && p.lang === "mn");
          return `<li><a href="${p.slug}.html"${active ? ' class="active" aria-current="page"' : ""}>${esc(p.title)}</a></li>`;
        })
        .join("");
      return `<div class="nav-group"><h4>${esc(group)}</h4><ul>${items}</ul></div>`;
    })
    .join("");
  return `<nav class="sidebar" aria-label="Баримтын цэс">${sections}</nav>`;
}

function languageRow(page) {
  if (!page?.lang) return "";
  const links = LANGS.map(({lang, label, flag}) => {
    const target = PAGES.find((p) => p.lang === lang);
    if (!target) return "";
    const img = `<img src="assets/icons/${flag}" width="18" height="18" alt="">`;
    return lang === page.lang
      ? `<b>${img} ${esc(label)}</b>`
      : `<a href="${target.slug}.html">${img} ${esc(label)}</a>`;
  }).join("");
  return `<div class="lang-row">${links}</div>`;
}

function shell({title, slug, body, toc = "", page}) {
  const rtl = page?.rtl ? ' dir="rtl"' : "";
  return `<!doctype html>
<html lang="${page?.lang ?? "mn"}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(title)} · Gerege Nexus</title>
<meta name="description" content="Gerege Nexus — үйлчилгээ, үйл ажиллагаа, системийн нэгдсэн нээлттэй эхийн платформ.">
<link rel="icon" href="assets/icons/flag-mn.png">
<link rel="stylesheet" href="assets/theme.css">
</head>
<body>
<a class="skip" href="#content">Агуулга руу шилжих</a>
<header class="topbar">
  <a class="brand" href="index.html"><span class="mark">GN</span> Gerege Nexus</a>
  <nav class="topnav">
    <a href="index.html">Тойм</a>
    <a href="architecture.html">Архитектур</a>
    <a href="module-authoring.html">Хөгжүүлэлт</a>
    <a href="documents.html">Баримтын индекс</a>
  </nav>
  <div class="topactions">
    <a class="ghost" href="${GITHUB}" target="_blank" rel="noopener">GitHub</a>
    <a class="gold" href="https://nexus.gerege.mn" target="_blank" rel="noopener">Нэвтрэх</a>
  </div>
</header>
<div class="layout">
${sidebar(slug)}
<main id="content"${rtl}>
${page ? languageRow(page) : ""}
${body}
</main>
${toc}
</div>
<footer class="sitefoot">
  <span>© 2026 Gerege Systems · Apache 2.0</span>
  <span><a href="${GITHUB}" target="_blank" rel="noopener">Эх код</a> · <a href="changelog.html">Өөрчлөлтийн түүх</a> · <a href="security.html">Аюулгүй байдал</a></span>
</footer>
</body>
</html>
`;
}

function tocFor(headings) {
  if (headings.length < 3) return "";
  const items = headings
    .map((h) => `<li class="d${h.depth}"><a href="#${h.id}">${h.text}</a></li>`)
    .join("");
  return `<aside class="toc" aria-label="Энэ хуудсанд"><h4>Энэ хуудсанд</h4><ul>${items}</ul></aside>`;
}

/* ── Build ────────────────────────────────────────────────────────────────── */

rmSync(OUT, {recursive: true, force: true});
mkdirSync(OUT, {recursive: true});

for (const page of PAGES) {
  const markdown = readFileSync(join(REPO, page.src), "utf8");
  const {html, headings} = render(markdown, page.src);
  writeFileSync(
    join(OUT, `${page.slug}.html`),
    shell({title: page.title, slug: page.slug, body: html, toc: tocFor(headings), page}),
  );
}

mkdirSync(join(OUT, "assets"), {recursive: true});
cpSync(join(REPO, "docs/assets"), join(OUT, "assets"), {
  recursive: true,
  filter: (src) => !src.endsWith(".md"),
});
cpSync(join(HERE, "theme.css"), join(OUT, "assets/theme.css"));
// Tells GitHub Pages not to run the output through Jekyll, which would drop
// any file or directory whose name begins with an underscore.
writeFileSync(join(OUT, ".nojekyll"), "");

console.log(`built ${PAGES.length} pages → ${relative(process.cwd(), OUT)}`);
