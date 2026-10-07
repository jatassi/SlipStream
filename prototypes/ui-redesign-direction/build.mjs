// PROTOTYPE build: inline the stage, the shared core and every variant into one
// double-clickable file. Usage: node build.mjs
import { readFileSync, writeFileSync, existsSync, readdirSync, mkdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import vm from 'node:vm';
import { createHash } from 'node:crypto';

const root = dirname(fileURLToPath(import.meta.url));
const read = (p) => readFileSync(join(root, p), 'utf8');

mkdirSync(join(root, 'dist'), { recursive: true });
const LUCIDE = 'tools/lucide-1.52.0.min.js';
if (!existsSync(join(root, LUCIDE))) {
  const res = await fetch('https://cdn.jsdelivr.net/npm/lucide@1.52.0/dist/umd/lucide.min.js');
  writeFileSync(join(root, LUCIDE), await res.text());
}

const variantDirs = readdirSync(join(root, 'src/variants')).filter((d) => existsSync(join(root, 'src/variants', d, 'meta.json'))).sort();
const variants = variantDirs.map((dir) => {
  const meta = JSON.parse(read(`src/variants/${dir}/meta.json`));
  const files = readdirSync(join(root, 'src/variants', dir)).sort();
  return {
    ...meta,
    dir,
    css: files.filter((f) => f.endsWith('.css')).map((f) => read(`src/variants/${dir}/${f}`)).join('\n'),
    js: (meta.scripts || files.filter((f) => f.endsWith('.js'))).map((f) => read(`src/variants/${dir}/${f}`)).join('\n;\n'),
  };
});

const core = ['src/core/data.js', 'src/core/sim.js', 'src/core/fmt.js', 'src/core/props.js', 'src/core/icons.js'].map(read);
const stageJs = read('src/proto/stage.js');
const stageCss = read('src/proto/stage.css');

// Bundle only the Lucide icons the sources name.
const sources = [...core, stageJs, ...variants.map((v) => v.js)].join('\n');
const names = new Set();
for (const m of sources.matchAll(/icon\(\s*['"]([a-z0-9-]+)['"]/g)) names.add(m[1]);
for (const m of sources.matchAll(/icons:\s*([a-z0-9- ]+)$/gm)) m[1].trim().split(/\s+/).forEach((n) => names.add(n));
for (const v of variants) (v.icons || []).forEach((n) => names.add(n));
const ctx = { console };
ctx.window = ctx; ctx.self = ctx; ctx.globalThis = ctx;
vm.createContext(ctx);
vm.runInContext(read(LUCIDE), ctx);
const pascal = (k) => k.split('-').map((p) => p[0].toUpperCase() + p.slice(1)).join('');
const icons = {};
const missing = [];
for (const n of [...names].sort()) {
  const node = ctx.lucide[pascal(n)];
  if (node) icons[n] = node; else missing.push(n);
}
if (missing.length) console.warn('Unknown Lucide icons:', missing.join(', '));

const posterFonts = ['Bebas+Neue', 'Gloock', 'Italiana', 'Bodoni+Moda:wght@500', 'Unbounded:wght@300', 'Big+Shoulders+Display:wght@800', 'Limelight', 'Archivo+Black', 'Poiret+One', 'Oswald:wght@300;400;500'];
const fontFamilies = [...new Set([...posterFonts, ...variants.flatMap((v) => v.fonts || [])])];
const fontsHref = `https://fonts.googleapis.com/css2?${fontFamilies.map((f) => `family=${f}`).join('&')}&display=swap`;
const fontCss = await selfHostFonts(fontsHref);

const script = (s) => `<script>${s.replace(/<\/script/gi, '<\\/script')}</script>`;
const html = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="theme-color" content="#050506">
<title>SlipStream v2 Directions</title>
<style>${fontCss}</style>
<style>${stageCss}</style>
${variants.map((v) => `<style id="css-${v.key}">${v.css}</style>`).join('\n')}
</head>
<body>
<div id="boot"><div class="boot__inner">Preparing the prop department<div class="boot__track"><div class="boot__fill"></div></div></div></div>
<div id="stage" class="stage"></div>
<nav id="protobar" aria-label="Prototype switcher"></nav>
${core.map(script).join('\n')}
${script(`window.SS.ICONS=${JSON.stringify(icons)};`)}
${script(stageJs)}
${variants.map((v) => script(v.js)).join('\n')}
</body>
</html>
`;
writeFileSync(join(root, 'dist/index.html'), html);
// The Artifact host wraps the page in its own skeleton, so its copy starts at <title>.
const body = html.split('<body>')[1].split('</body>')[0];
const head = html.split('<head>')[1].split('</head>')[0].replace(/<meta[^>]*>\n?/g, '');
writeFileSync(join(root, 'dist/artifact.html'), head.replace(/^\s+/, '') + body);
console.log(`dist/index.html ${(html.length / 1024).toFixed(0)} KB · ${variants.map((v) => v.key).join(' ') || 'no variants'} · ${Object.keys(icons).length} icons`);

// Inline every face's latin subset as a data URI so the single file keeps its type offline.
// Responses are cached in tools/font-cache, so a rebuild needs the network only for new faces.
async function selfHostFonts(href) {
  const cacheDir = join(root, 'tools/font-cache');
  mkdirSync(cacheDir, { recursive: true });
  const key = (s) => createHash('sha1').update(s).digest('hex').slice(0, 16);
  const cached = async (url, binary) => {
    const file = join(cacheDir, key(url) + (binary ? '.woff2' : '.css'));
    if (existsSync(file)) return readFileSync(file, binary ? undefined : 'utf8');
    const res = await fetch(url, { headers: { 'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36' } });
    if (!res.ok) throw new Error(`${res.status} ${url}`);
    const body = binary ? Buffer.from(await res.arrayBuffer()) : await res.text();
    writeFileSync(file, body);
    return body;
  };
  const css = await cached(href, false);
  const blocks = [...css.matchAll(/\/\* ([a-z-]+) \*\/\s*(@font-face \{[^}]+\})/g)].filter((m) => m[1] === 'latin').map((m) => m[2]);
  const out = [];
  for (const block of blocks) {
    const url = block.match(/url\((https:[^)]+\.woff2)\)/)[1];
    const data = (await cached(url, true)).toString('base64');
    out.push(block.replace(url, `data:font/woff2;base64,${data}`));
  }
  return out.join('\n');
}
