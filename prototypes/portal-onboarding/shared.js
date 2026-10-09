// PROTOTYPE, throwaway. What every variant shares: the in-memory session, the dead-link card, the PIN field,
// the simulated OS passkey sheet, the ambient tint and the portal a Member lands in. Layout of the flows lives in a.js, b.js, c.js.

import * as D from './data.js';

export const esc = (s) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
export const I = (n, c = '') => `<i data-lucide="${n}" class="ic ${c}" aria-hidden="true"></i>`;
export const icons = () => window.lucide && window.lucide.createIcons();
export const W = (id) => D.works[id];
export const meta = (w) => (w.type === 'series' ? `Series · ${w.network}` : `Movie · ${w.year}`);
export const MARK = `<svg class="mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M2.5 6.5c6 0 9.5 0 12.5 3.2"/><path d="M2.5 12h9.5"/><path d="M2.5 17.5c6 0 9.5 0 12.5-3.2"/><circle cx="18.6" cy="12" r="2.6" fill="currentColor" stroke="none"/></svg>`;
export const wordmark = () => `<span class="wordmark">${MARK}SlipStream</span>`;

export const day = (d) => d.toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short' });
export const clock = (d) => d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
export const expiryShort = () => `${day(D.link.expiresAt)} at ${clock(D.link.expiresAt)}`;

export function art(w, cls = '') {
  return `<span class="art ${cls}"><img src="${w.art.poster}" alt="" decoding="async" onerror="this.parentNode.classList.add('art-x');this.remove()"><span class="art-fb" aria-hidden="true">${esc(w.title)}</span></span>`;
}

// ───────────── session: everything a run of the flow changes, in memory only

export function session(cfg) {
  const signedIn = cfg.state === 'signedin';
  return {
    cfg,
    account: signedIn ? { ...D.existingAccount } : null,
    linkUsed: false,
    mine: signedIn ? D.existingAccount.requests.map((work) => ({ work, state: 'coming' })) : [],
    notifyEmail: true,
  };
}

export const host = (cfg) => (cfg.https ? D.server.https : D.server.http).replace(/^https?:\/\//, '');

// The link carries Member settings; using it gives them to the Account that signs in through it.
export function useLink(s) {
  s.linkUsed = true;
  s.account.autoApprove = D.grants.autoApprove;
  s.account.quota = D.grants.quota;
}

export function createAccount(s, { name, email }) {
  s.account = { name, first: name.split(/\s+/)[0], email, initials: name.split(/\s+/).slice(0, 2).map((p) => p[0]).join('').toUpperCase(), passkey: false };
  useLink(s);
}

export function signIn(s, { viaLink = true } = {}) {
  s.account = { ...D.existingAccount };
  s.mine = D.existingAccount.requests.map((work) => ({ work, state: 'coming' }));
  if (viaLink) useLink(s);
}

export function request(s, id) {
  if (s.mine.some((m) => m.work === id)) return;
  s.mine.unshift({ work: id, state: 'requested', fresh: true });
}

export function stateOf(s, id) {
  const m = s.mine.find((x) => x.work === id);
  if (m) return m.state;
  return D.availability[id] || 'none';
}

// ───────────── form checks shared by every sign-up and sign-in form

export function readForm(form) {
  const v = Object.fromEntries(new FormData(form).entries());
  for (const k of Object.keys(v)) v[k] = String(v[k]).trim();
  return v;
}

export function checkSignUp(s, v) {
  const errs = {};
  if (!v.name) errs.name = 'Enter your name.';
  if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(v.email || '')) errs.email = 'Enter an email address.';
  if (!/^\d{4}$/.test(v.pin || '')) errs.pin = 'Enter four digits.';
  if (!errs.email && s.cfg.state === 'existing' && v.email.toLowerCase() === D.existingAccount.email) errs.form = 'exists';
  return errs;
}

export function checkSignIn(v) {
  const errs = {};
  if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(v.email || '')) errs.email = 'Enter an email address.';
  if (!/^\d{4}$/.test(v.pin || '')) errs.pin = 'Enter four digits.';
  return errs;
}

export function showErrors(form, errs) {
  form.querySelectorAll('.err, .formerr').forEach((e) => e.remove());
  form.querySelectorAll('.fld.bad').forEach((e) => e.classList.remove('bad'));
  for (const [k, msg] of Object.entries(errs)) {
    if (k === 'form') continue;
    const f = form.querySelector(`[data-f="${k}"]`);
    if (!f) continue;
    f.classList.add('bad');
    f.insertAdjacentHTML('beforeend', `<span class="err">${I('circle-alert')}${esc(msg)}</span>`);
  }
  if (errs.form === 'exists') {
    form.insertAdjacentHTML('afterbegin', `<div class="formerr">${I('circle-alert')}<p>That email already has an Account here. <button type="button" data-act="to-signin">Sign in instead</button></p></div>`);
  }
  icons();
  const first = form.querySelector('.fld.bad input');
  if (first) first.focus();
  return Object.keys(errs).length === 0;
}

export const field = (k, label, input, caption = '') => `<label class="fld" data-f="${k}"><span class="fld-l">${label}</span>${input}${caption ? `<span class="fld-c">${caption}</span>` : ''}</label>`;
export const nameField = (caption = `${D.admin.name} sees this beside your requests.`) => field('name', 'Your name', '<input class="in" name="name" autocomplete="name" autocapitalize="words" enterkeyhint="next">', caption);
export const emailField = (value, caption = 'You sign in with this.') => field('email', 'Email', `<input class="in" name="email" type="email" inputmode="email" autocomplete="email" autocapitalize="off" spellcheck="false" value="${esc(value)}" enterkeyhint="next">`, caption);

// PIN: one real numeric input under four boxes, so the phone shows its number pad and autofill still works.
export const pinField = (caption = 'Four digits. You can type it on any screen, even a TV.', label = 'PIN') => `
  <div class="fld" data-f="pin"><span class="fld-l" id="pin-l">${label}</span>
    <div class="pin-row"><div class="pin" data-pin>
      <input class="pin-in" name="pin" inputmode="numeric" pattern="[0-9]*" maxlength="4" autocomplete="off" aria-labelledby="pin-l" enterkeyhint="go">
      <span class="pin-b"></span><span class="pin-b"></span><span class="pin-b"></span><span class="pin-b"></span>
    </div><button type="button" class="peek" data-act="peek" aria-label="Show PIN" aria-pressed="false">${I('eye')}</button></div>
    ${caption ? `<span class="fld-c">${caption}</span>` : ''}
  </div>`;

export function wirePins(el) {
  el.querySelectorAll('[data-pin]').forEach((pin) => {
    const inp = pin.querySelector('input');
    const boxes = [...pin.querySelectorAll('.pin-b')];
    const peekBtn = pin.parentNode.querySelector('[data-act="peek"]');
    const paint = () => {
      const v = inp.value;
      const shown = peekBtn && peekBtn.getAttribute('aria-pressed') === 'true';
      boxes.forEach((b, i) => {
        b.textContent = shown && v[i] ? v[i] : '';
        b.classList.toggle('dot', !shown && !!v[i]);
        b.classList.toggle('cur', i === Math.min(v.length, 3));
      });
    };
    inp.addEventListener('input', () => { inp.value = inp.value.replace(/\D/g, '').slice(0, 4); paint(); });
    inp.addEventListener('focus', () => { pin.classList.add('focus'); paint(); });
    inp.addEventListener('blur', () => pin.classList.remove('focus'));
    if (peekBtn) peekBtn.addEventListener('click', () => {
      const on = peekBtn.getAttribute('aria-pressed') !== 'true';
      peekBtn.setAttribute('aria-pressed', on);
      peekBtn.setAttribute('aria-label', on ? 'Hide PIN' : 'Show PIN');
      peekBtn.innerHTML = I(on ? 'eye-off' : 'eye');
      icons();
      paint();
    });
    paint();
  });
}

export function busy(btn, ms = 650) {
  btn.classList.add('busy');
  btn.insertAdjacentHTML('afterbegin', '<i class="spin"></i>');
  return new Promise((r) => setTimeout(r, ms));
}

// ───────────── the OS passkey sheet, simulated. It belongs to the phone, so it never takes the Work's tint.

export function passkeySystem(root, cfg, mode, email) {
  return new Promise((resolve) => {
    const wrap = document.createElement('div');
    wrap.className = 'sys';
    const title = mode === 'create' ? `Save a passkey for ${host(cfg)}?` : `Sign in to ${host(cfg)}?`;
    const body = mode === 'create'
      ? `<p class="sys-p">This phone keeps a passkey for ${esc(email)} and signs you in with Face ID.</p>`
      : `<div class="sys-acct">${I('key-round')}<div>${esc(email)}<small>Passkey from this phone</small></div></div>`;
    wrap.innerHTML = `<div class="sys-scrim"></div><div class="sys-card" role="dialog" aria-modal="true" aria-label="Passkey">
      <div class="sys-top"><span>Passkey</span><button class="sys-x" aria-label="Cancel" data-x>×</button></div>
      <div class="sys-key">${I('key-round')}</div>
      <h2 class="sys-t">${title}</h2>${body}
      <button class="sys-go" data-go>Continue</button>
      <span class="sys-note">Simulated system sheet</span></div>`;
    root.append(wrap);
    icons();
    const done = (ok) => { wrap.remove(); resolve(ok); };
    wrap.querySelector('[data-x]').onclick = () => done(false);
    wrap.querySelector('.sys-scrim').onclick = () => done(false);
    wrap.querySelector('[data-go]').onclick = () => {
      const card = wrap.querySelector('.sys-card');
      card.innerHTML = `<svg class="sys-face" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><path d="M3 7V5a2 2 0 0 1 2-2h2"/><path d="M17 3h2a2 2 0 0 1 2 2v2"/><path d="M21 17v2a2 2 0 0 1-2 2h-2"/><path d="M7 21H5a2 2 0 0 1-2-2v-2"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><path d="M9 9h.01"/><path d="M15 9h.01"/></svg><p class="sys-p">Face ID</p>`;
      setTimeout(() => {
        card.innerHTML = `<svg class="sys-face ok" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><path d="m8 12 3 3 5-6"/></svg><p class="sys-p">Done</p>`;
        setTimeout(() => done(true), 520);
      }, 760);
    };
  });
}

// ───────────── toast and sheet

export function toast(root, msg, icon = 'circle-check') {
  let t = root.querySelector(':scope > .toast');
  if (!t) { t = document.createElement('div'); t.className = 'toast'; t.setAttribute('role', 'status'); root.append(t); }
  t.innerHTML = `${I(icon)}<span>${esc(msg)}</span>`;
  icons();
  requestAnimationFrame(() => t.classList.add('on'));
  clearTimeout(t._h);
  t._h = setTimeout(() => t.classList.remove('on'), 3200);
}

export function openSheet(root, html, { label = 'Sheet', onScrim } = {}) {
  const wrap = document.createElement('div');
  wrap.className = 'sheet-wrap';
  wrap.innerHTML = `<div class="sheet-scrim"></div><div class="sheet" role="dialog" aria-modal="true" aria-label="${esc(label)}"><span class="grab" aria-hidden="true"></span>${html}</div>`;
  root.append(wrap);
  icons();
  const sheet = wrap.querySelector('.sheet');
  const api = {
    el: sheet,
    set(inner) { sheet.innerHTML = `<span class="grab" aria-hidden="true"></span>${inner}`; icons(); },
    close() { return new Promise((r) => { sheet.classList.add('out'); wrap.querySelector('.sheet-scrim').style.transition = 'opacity .26s'; wrap.querySelector('.sheet-scrim').style.opacity = 0; setTimeout(() => { wrap.remove(); r(); }, 260); }); },
  };
  wrap.querySelector('.sheet-scrim').onclick = () => onScrim && onScrim(api);
  return api;
}

// ───────────── ambient: the page takes its colour from the Work in focus (sampler copied from lit-console)

const tintCache = new Map();
function sampleTint(src) {
  if (tintCache.has(src)) return tintCache.get(src);
  const p = new Promise((res) => {
    const im = new Image();
    im.onload = () => {
      try {
        const c = document.createElement('canvas');
        c.width = 24; c.height = 36;
        const g = c.getContext('2d', { willReadFrequently: true });
        g.drawImage(im, 0, 0, 24, 36);
        const d = g.getImageData(0, 0, 24, 36).data;
        let x = 0, y = 0, ws = 0, ss = 0;
        for (let i = 0; i < d.length; i += 4) {
          const r = d[i] / 255, gg = d[i + 1] / 255, b = d[i + 2] / 255;
          const mx = Math.max(r, gg, b), mn = Math.min(r, gg, b), l = (mx + mn) / 2, dd = mx - mn;
          if (dd < 0.04) continue;
          const sat = dd / (1 - Math.abs(2 * l - 1));
          let h = mx === r ? ((gg - b) / dd) % 6 : mx === gg ? (b - r) / dd + 2 : (r - gg) / dd + 4;
          h = (h * 60 + 360) % 360;
          const w = sat * sat * (1 - Math.abs(l - 0.5) * 1.6);
          if (w <= 0) continue;
          x += Math.cos((h * Math.PI) / 180) * w; y += Math.sin((h * Math.PI) / 180) * w; ws += w; ss += sat * w;
        }
        if (!ws) return res({ h: 220, s: 12 });
        res({ h: ((Math.atan2(y, x) * 180) / Math.PI + 360) % 360, s: Math.round(Math.min(70, Math.max(18, (ss / ws) * 80))) });
      } catch { res(null); }
    };
    im.onerror = () => res(null);
    im.src = src;
  });
  tintCache.set(src, p);
  return p;
}

export const AMBIENT = '<div class="amb" aria-hidden="true"><div class="amb-i"></div><div class="amb-i"></div><div class="amb-tint"></div></div>';

let ambOn = 0, ambWork = null;
export function setAmbient(root, id) {
  if (ambWork === id) return;
  ambWork = id;
  const w = W(id);
  const layers = [...root.querySelectorAll('.amb-i')];
  if (layers.length === 2) {
    const next = layers[1 - ambOn];
    next.style.backgroundImage = `url('${w.art.backdrop}')`;
    next.classList.add('on');
    layers[ambOn].classList.remove('on');
    ambOn = 1 - ambOn;
  }
  sampleTint(w.art.poster).then((t) => {
    if (!t || ambWork !== id) return;
    root.style.setProperty('--h', t.h.toFixed(1));
    root.style.setProperty('--s', `${t.s}%`);
  });
}
export function resetAmbient() { ambWork = null; ambOn = 0; }

export const statusbar = () => `<div class="statusbar" aria-hidden="true"><span>21:40</span><span class="sb-r">${I('signal')}${I('wifi')}${I('battery-full')}</span></div>`;

// ───────────── the dead-link card: one card, one sentence per reason

export const DEAD = {
  unknown: { icon: 'link-2-off', text: 'This link does not work, so check that you have all of it or ask the person who sent it for a new one.' },
  withdrawn: { icon: 'link-2-off', text: `This link was replaced or cancelled, so ask ${D.admin.name} whether a new one is on its way.` },
  expired: { icon: 'clock-alert', text: `This link expired today at 19:12, seven days after it was sent, so ask ${D.admin.name} for a new one.` },
  used: { icon: 'circle-check', text: `This link was already used, and each link works once, so sign in or ask ${D.admin.name} for a new one.`, act: ['sign-in', 'Sign in'] },
  held: { icon: 'circle-check', text: 'Your Account already has everything this link gives, so nothing changed and the link is still unused.', act: ['go-portal', 'Go to SlipStream'] },
  closed: { icon: 'pause', text: `Requests are turned off on this server for now, so its links do not work until ${D.admin.name} turns them back on.` },
  unreached: { icon: 'wifi-off', text: 'The server did not answer, so check your connection and try again.', act: ['retry', 'Try again'], warn: true },
};

export function deadCard(reason) {
  const d = DEAD[reason];
  return `<section class="dead" aria-labelledby="dead-h">
    <div class="dead-brand">${wordmark()}</div>
    <div class="dead-card">
      <span class="dead-ic ${d.warn ? 'warn' : ''}">${I(d.icon)}</span>
      <h1 class="dead-h" id="dead-h">Signup link</h1>
      <p class="dead-p">${esc(d.text)}</p>
      ${d.act ? `<div class="dead-foot"><button class="btn ${d.act[0] === 'retry' ? '' : 'pri'}" data-act="${d.act[0]}">${d.act[0] === 'retry' ? I('rotate-cw') : ''}${d.act[1]}</button></div>` : ''}
    </div></section>`;
}

// ───────────── portal: where every flow ends. Members see outcomes only: requested, coming, ready.

const RIB = { ready: ['Ready', 'circle-check'], coming: ['Coming', 'clock'], requested: ['Requested', 'hourglass'] };
export const ribbon = (state, cls = '') => (RIB[state] ? `<span class="rib ${state} ${cls}">${I(RIB[state][1])}${RIB[state][0]}</span>` : '');

function control(s, id, size = 'sm') {
  const st = stateOf(s, id);
  if (st === 'none') return `<button class="btn ${size} pri pc-req" data-act="request" data-id="${id}">${I('plus')}<span>Request</span></button>`;
  return ribbon(st);
}

function mineLine(s, m) {
  if (m.state === 'coming') return 'New episodes are on the way.';
  const how = s.notifyEmail ? 'We will email you when it is ready.' : 'We will tell you here when it is ready.';
  return s.account && s.account.autoApprove ? `Approved automatically. ${how}` : `Waiting for ${D.admin.name} to approve it.`;
}

export function Portal(el, root, s, o = {}) {
  let q = '', work = null;
  const homeWork = o.homeWork || 'dune-2';

  const results = () => {
    if (!q) return '';
    const hits = Object.values(D.works).filter((w) => w.title.toLowerCase().includes(q.toLowerCase())).slice(0, 6);
    if (!hits.length) return `<div class="p-res"><p class="p-res-empty">Nothing matches "${esc(q)}". Try another spelling.</p></div>`;
    return `<ul class="p-res" aria-label="Results">${hits.map((w) => `<li class="p-res-i"><button data-act="open" data-id="${w.id}" aria-label="${esc(w.title)}">${art(w)}</button><button class="p-res-t" data-act="open" data-id="${w.id}"><b>${esc(w.title)}</b><span>${meta(w)}</span></button>${control(s, w.id)}</li>`).join('')}</ul>`;
  };

  const card = (id, sub) => {
    const w = W(id);
    const st = stateOf(s, id);
    return `<li class="pc"><button class="pc-art" data-act="open" data-id="${id}" aria-label="${esc(w.title)}">${art(w)}${st !== 'none' ? ribbon(st) : ''}</button><p class="pc-t">${esc(w.title)}</p><p class="pc-s">${sub || meta(w)}</p>${st === 'none' ? control(s, id) : ''}</li>`;
  };

  const mine = () => (s.mine.length ? `<section class="p-sec" aria-labelledby="mine-h"><div class="p-sec-h"><h2 id="mine-h">Your requests</h2><span class="n">${s.mine.length}</span></div>
    <ul class="mine">${s.mine.map((m) => { const w = W(m.work); return `<li class="mine-i" ${m.fresh ? 'data-fresh' : ''}><button data-act="open" data-id="${w.id}" aria-label="${esc(w.title)}">${art(w)}</button><div class="mine-t">${ribbon(m.state, m.fresh ? 'new' : '')}<b>${esc(w.title)}</b><p>${mineLine(s, m)}</p></div></li>`; }).join('')}</ul></section>` : '');

  const home = () => {
    const name = s.account ? s.account.first : null;
    return `
    <header class="p-top">${wordmark()}${s.account ? `<button class="ib" aria-label="Inbox">${I('bell')}</button><span class="avatar" aria-label="${esc(s.account.name)}">${esc(s.account.initials)}</span>` : ''}</header>
    ${o.banner ? o.banner() : ''}
    <section class="p-hero" aria-labelledby="ph-t">
      <div class="p-wall" aria-hidden="true">${D.wall.slice(0, 16).map((id) => `<img src="${W(id).art.poster}" alt="" onerror="this.style.visibility='hidden'">`).join('')}</div>
      <h1 id="ph-t">${name ? `What are we watching, ${esc(name)}?` : 'What do you want to watch?'}</h1>
      <label class="p-search"><span class="sr">Search movies and series</span>${I('search')}<input type="search" data-search placeholder="Search movies and series" autocomplete="off" enterkeyhint="search" value="${esc(q)}"></label>
      ${o.hint && o.hint() ? `<p class="p-hint">${I('sparkles')}<span>${o.hint()}</span></p>` : ''}
    </section>
    <div data-results>${results()}</div>
    ${mine()}
    <section class="p-sec" aria-labelledby="pop-h"><div class="p-sec-h"><h2 id="pop-h">Popular this week</h2></div><ul class="strip">${D.trending.map((id) => card(id)).join('')}</ul></section>
    <section class="p-sec" aria-labelledby="new-h"><div class="p-sec-h"><h2 id="new-h">New in the library</h2></div><ul class="strip">${D.newInLibrary.map((id) => card(id)).join('')}</ul></section>
    <nav class="tabs" aria-label="Portal"><button class="tab on">${I('house')}Home</button><button class="tab">${I('search')}Search</button><button class="tab">${I('bookmark')}Requests</button><button class="tab">${I('circle-user')}Account</button></nav>`;
  };

  const workPage = () => {
    const w = W(work);
    const st = stateOf(s, work);
    const act = st === 'none'
      ? `<button class="btn pri block" data-act="request" data-id="${w.id}">${I('plus')}<span>Request</span></button>`
      : `<div class="wk-state">${ribbon(st)}<span>${st === 'ready' ? 'In the library. Watch it on Plex.' : st === 'coming' ? 'On its way to the library.' : mineLine(s, s.mine.find((m) => m.work === work))}</span></div>`;
    return `<article class="wk">
      <nav class="wk-nav"><button class="back" data-act="back">${I('chevron-left')}Back</button></nav>
      <div class="wk-bd" aria-hidden="true"><img src="${w.art.backdrop}" alt="" onerror="this.remove()"></div>
      <div class="wk-body">
        <h1 class="wk-logo"><img src="${w.art.logo}" alt="${esc(w.title)}" onerror="this.outerHTML='<span class=wk-logo-t>${esc(w.title).replace(/'/g, '')}</span>'"></h1>
        <p class="wk-meta">${meta(w)}</p>
        <p class="wk-ov">${esc(w.overview)}</p>
        <div class="wk-act" data-wk-act>${act}</div>
      </div></article>`;
  };

  const api = {
    render({ focus } = {}) {
      const top = el.scrollTop;
      el.innerHTML = work ? workPage() : home();
      icons();
      const latest = s.mine.find((m) => m.fresh);
      setAmbient(root, work || (latest && latest.work) || homeWork);
      if (focus) { const i = el.querySelector('[data-search]'); if (i) i.focus(); }
      if (o.keepScroll) root.scrollTop = top;
      if (o.after) o.after(api);
    },
    open(id) { work = id; root.scrollTop = 0; api.render(); },
    home() { work = null; api.render(); },
    get work() { return work; },
    refresh() { const y = root.scrollTop; api.render(); root.scrollTop = y; },
  };

  el.addEventListener('click', (e) => {
    const b = e.target.closest('[data-act]');
    if (!b || !el.contains(b)) return;
    const act = b.dataset.act;
    if (act === 'open') api.open(b.dataset.id);
    else if (act === 'back') { api.home(); root.scrollTop = 0; }
    else if (act === 'request') o.onRequest(b.dataset.id, b, api);
  });
  el.addEventListener('input', (e) => {
    if (!e.target.matches('[data-search]')) return;
    q = e.target.value.trim();
    el.querySelector('[data-results]').innerHTML = results();
    icons();
  });
  return api;
}
