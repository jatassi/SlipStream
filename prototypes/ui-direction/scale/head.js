// PROTOTYPE. The Work page's head, shared by every matrix variant so only the matrix differs between them.
// Rendered from the descriptor's display fields; no per-type code.

import { esc, I, art, dot } from '../variants/lit-console/variant.js';
import { cells, counts, summaryLine } from './model.js';

function metaLine(work, desc) {
  return desc.display.map((f) => {
    const v = f.value ? f.value(work) : work[f.key];
    if (v == null) return '';
    const text = Array.isArray(v) ? v.join(f.join) : String(v);
    if (f.badge) return `<span class="rate-b">${esc(text)}</span>`;
    return f.mono ? `<span class="num">${esc(text)}</span>` : esc(text);
  }).filter(Boolean).join(' · ');
}

function statePills(work, desc) {
  const c = counts(cells(work, desc));
  const [met, ...rest] = summaryLine(c);
  const live = c.downloading + c.converting;
  return `<span class="pill-met">${I('check')}${esc(met)}</span>${c.failed ? `<span class="chip-fail">${I('circle-x')}${c.failed} failed</span>` : ''}${live ? `<span class="pill-live">${dot('draft')}${live} in flight</span>` : ''}<span class="t2 wk-state-rest">${esc(rest.filter((r) => !/failed|in flight/.test(r)).join(' · '))}</span>`;
}

const logo = (w) => `<h1 class="wk-logo"><img src="${w.art.logo}" alt="${esc(w.title)}" onerror="this.replaceWith(Object.assign(document.createElement('span'),{className:'wk-logo-t',textContent:this.alt}))"></h1>`;
const monitorSwitch = (compact) => `<button class="btn tog${compact ? ' tog-c' : ''}" data-monitor role="switch" aria-checked="true"${compact ? ' aria-label="Monitored"' : ''}>${compact ? I('eye') : `${I('eye')}<span>Monitored</span>`}<span class="sw" aria-hidden="true"><span></span></span></button>`;

export function headWide(work, desc) {
  return `<div class="wk-head">
    <div class="wk-bd" aria-hidden="true"><img src="${work.art.backdrop}" alt="" onerror="this.remove()"></div>
    ${art(work, 'poster-lg')}
    <div class="wk-id">
      ${logo(work)}
      <p class="meta t2">${metaLine(work, desc)}</p>
      <p class="wk-state">${statePills(work, desc)}</p>
      <p class="overview t2">${esc(work.overview)}</p>
      <div class="wk-act">
        <button class="btn pri" data-toast="Searching every wanted cell is not in this prototype">${I('search')}Search</button>
        <button class="btn" data-toast="Manual search is not in this prototype">${I('list-filter')}Manual search</button>
        <button class="btn" data-toast="Editing Editions is not in this prototype">${I('layers')}Editions</button>
        ${monitorSwitch(false)}
        <button class="btn ib-c" data-menu="work" aria-haspopup="menu" aria-label="More actions for ${esc(work.title)}">${I('ellipsis')}</button>
      </div>
    </div>
  </div>`;
}

export function headPhone(work, desc) {
  return `<div class="statusbar" aria-hidden="true"></div>
  <header class="ph-nav"><button class="back tap" data-go="home">${I('chevron-left')}Home</button><span class="grow"></span><button class="ib tap" data-menu="work" aria-haspopup="dialog" aria-label="More actions for ${esc(work.title)}">${I('ellipsis')}</button></header>
  <div class="pw-top">
    <div class="pw-bd" aria-hidden="true"><img src="${work.art.backdrop}" alt="" onerror="this.remove()"></div>
    <div class="pw-id">${art(work, 'poster-md')}<div>${logo(work)}<p class="meta t2">${metaLine(work, desc)}</p></div></div>
  </div>
  <p class="wk-state ph-pad">${statePills(work, desc)}</p>
  <div class="pw-act ph-pad"><button class="btn pri" data-toast="Searching every wanted cell is not in this prototype">${I('search')}Search</button><button class="btn" data-toast="Manual search is not in this prototype">${I('list-filter')}Manual</button>${monitorSwitch(true)}</div>`;
}
