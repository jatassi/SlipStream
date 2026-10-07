
const fs = require('fs');
const items = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const out = items.map(src => {
  const r = {};
  for (const flags of ['i', 'iu']) {
    try { new RegExp(src, flags); r[flags] = null; } catch (e) { r[flags] = String(e.message); }
  }
  return r;
});
fs.writeFileSync(process.argv[3], JSON.stringify(out));
