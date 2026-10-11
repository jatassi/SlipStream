// PROTOTYPE boundary check, after Q5 of The Content Type contract: the Work page renderers never name a concrete Content Type.
import { readFileSync, readdirSync } from 'node:fs';
const dir = new URL('.', import.meta.url).pathname;
const files = ['head.js', 'model.js', ...readdirSync(`${dir}variants`).map((f) => `variants/${f}`)];
const bad = files.flatMap((f) => readFileSync(dir + f, 'utf8').split('\n').map((l, i) => [f, i + 1, l]).filter(([, , l]) => /['"`](movie|series|season|episode)['"`]|contentType\s*===|\.type\s*===/.test(l)));
bad.forEach(([f, n, l]) => console.log(`${f}:${n}: ${l.trim()}`));
if (bad.length) { console.log(`${bad.length} line(s) name a concrete Content Type`); process.exit(1); }
console.log(`ok: ${files.length} renderer files name no Content Type`);
