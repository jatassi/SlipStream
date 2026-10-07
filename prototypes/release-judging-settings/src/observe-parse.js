const E = require('./engine.js');
const fs = require('fs');
const names = fs.readFileSync(process.argv[2], 'utf8').split('\n').filter(Boolean);
for (const n of names) {
  const i = E.interpret(n, [E.BASE_PACK]);
  const b = k => i.beliefs[k] ? [...i.beliefs[k].values()].map(x => x.value + (x.conf < 0.9 ? '~' : '')).join('/') : '-';
  const q = E.qualityOf(i.rel.source, i.rel.res);
  console.log([q || 'Q?', b('hdr'), b('audio') + (i.beliefs.atmos ? '+atmos' : ''), b('channels'), b('codec'), b('service'), b('edition'), b('revision'), i.core.group || 'G?', i.hazards.map(h => '!' + h.label).join(','), '|', n].join('  '));
}
