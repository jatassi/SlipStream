// PROTOTYPE, throwaway. Variant B · Guided steps: one question per screen, phone-native, a big keypad for the PIN, ending on a first request.

import * as D from './data.js';
import { AMBIENT, I, Portal, busy, createAccount, deadCard, esc, expiryShort, icons, passkeySystem, request, resetAmbient, session, setAmbient, signIn, statusbar, toast, useLink, W } from './shared.js';

const PICKS = ['the-matrix', 'the-dark-knight', 'hotd', 'eeaao', 'the-boys', 'got'];

export function mount({ root, cfg }) {
  const s = session(cfg);
  root.className = `lc lc-${cfg.device}`;
  root.innerHTML = AMBIENT + statusbar() + '<div data-view></div>';
  const view = root.querySelector('[data-view]');
  setAmbient(root, 'dune-2');

  // The flow is a list of steps; the welcome screen decides which list runs.
  const FLOWS = {
    new: ['welcome', 'name', 'pin', ...(cfg.https ? ['passkey'] : []), 'notify', 'first'],
    existing: ['welcome', 'signin', ...(cfg.https ? ['passkey'] : [])],
    signedin: ['welcome'],
  };
  let flow = FLOWS[cfg.state];
  let at = 0;
  let link = cfg.link;
  let email = D.invitee.email;
  let editingEmail = false;
  let name = '';
  let pin = '', first = '', pinMsg = '', pinBad = false;
  let pick = null;

  const step = () => flow[at];
  let plain = false; // signing in after a used link: nothing gained
  // Existing Accounts finish on the portal with a note of what the link gave them.
  const next = () => { at += 1; if (at >= flow.length) return done(flow === FLOWS.new || plain ? '' : `Your Account now has ${D.grantsShort}.`); draw(); };
  const back = () => { at = Math.max(0, at - 1); pin = first = pinMsg = ''; draw(); };

  const progress = () => {
    const n = flow.length - 1;
    return `<header class="b-bar"><button class="ib" data-act="back" aria-label="Back">${I('chevron-left')}</button>
      <div class="b-prog" role="progressbar" aria-label="Step ${at} of ${n}" aria-valuemin="0" aria-valuemax="${n}" aria-valuenow="${at}">${Array.from({ length: n }, (_, i) => `<span class="${i < at ? 'done' : ''}"></span>`).join('')}</div></header>`;
  };

  const wallCol = (ids) => `<div class="b-col">${[...ids, ...ids].map((id) => `<img src="${W(id).art.poster}" alt="" onerror="this.style.visibility='hidden'">`).join('')}</div>`;

  const keypad = (fn = '') => `<div class="keys" role="group" aria-label="Keypad">${[1, 2, 3, 4, 5, 6, 7, 8, 9].map((d) => `<button class="key" data-key="${d}" aria-label="${d}">${d}</button>`).join('')}
    ${fn || '<span></span>'}<button class="key" data-key="0" aria-label="0">0</button><button class="key fn" data-key="del" aria-label="Delete">${I('delete')}</button></div>`;
  const dots = () => `<div class="dots ${pinBad ? 'shake' : ''}" aria-hidden="true">${[0, 1, 2, 3].map((i) => `<span class="${i < pin.length ? 'on' : ''}"></span>`).join('')}</div>
    <p class="dots-msg ${pinBad ? 'bad' : ''}" role="status">${esc(pinMsg) || `${pin.length} of 4`}</p>`;

  const STEPS = {
    welcome: () => {
      const signedIn = cfg.state === 'signedin' && s.account;
      const existing = flow === FLOWS.existing;
      const n = cfg.device === 'wide' ? 6 : 3;
      const cols = Array.from({ length: n }, (_, i) => D.wall.filter((__, j) => j % n === i));
      return `<div class="b b-welcome">
        <div class="b-wall" aria-hidden="true" style="grid-template-columns:repeat(${n},1fr)">${cols.map(wallCol).join('')}</div>
        <div class="b-w-body">
          <p class="b-w-from"><span class="avatar">${D.admin.initials}</span><span><b>${D.admin.name}</b> invited you to</span></p>
          <h1>${D.server.name}</h1>
          ${signedIn
            ? `<p class="lede">This link adds ${D.grantsShort} to your Account.</p>
               <div class="who"><span class="avatar lg">${esc(s.account.initials)}</span><div><b>${esc(s.account.name)}</b><span>${esc(s.account.email)}</span></div></div>
               <button class="btn pri block" data-act="confirm"><span>Add to my Account</span></button>
               <button class="btn ghost block" data-act="sign-out">Not you? Sign out</button>`
            : existing
              ? `<p class="lede">Sign in, and your Account gets ${D.grantsShort}.</p>
                 <button class="btn pri block" data-act="next"><span>Sign in</span></button>
                 <button class="btn ghost block" data-act="to-new">Create an Account instead</button>`
              : `<p class="lede">Ask for any movie or series, and hear when it is ready to watch. ${D.grantsLine}</p>
                 <p class="b-as">${editingEmail
                   ? `<label class="sr" for="b-email">Email</label><input class="in" id="b-email" type="email" inputmode="email" autocomplete="email" value="${esc(email)}"><button data-act="email-done">Done</button>`
                   : `Signing up as <b>${esc(email)}</b><button data-act="email-edit">Change</button>`}</p>
                 <button class="btn pri block" data-act="next"><span>Get started</span></button>
                 <button class="btn ghost block" data-act="to-existing">I already have an Account</button>`}
          <p class="a-fine">This link works once, until ${expiryShort()}.</p>
        </div></div>`;
    },
    name: () => `${progress()}<form class="b-step" data-form="name" novalidate>
        <h1>What should we call you?</h1><p class="lede">${D.admin.name} sees this beside your requests.</p>
        <label class="sr" for="b-name">Your name</label><input class="big-in" id="b-name" name="name" placeholder="Your name" autocomplete="name" autocapitalize="words" enterkeyhint="next" value="${esc(name)}">
        <p class="err" data-name-err hidden>${I('circle-alert')}Enter your name.</p>
        <div class="b-fill"></div><div class="b-foot"><button class="btn pri block" type="submit"><span>Continue</span></button></div></form>`,
    pin: () => `${progress()}<div class="b-step">
        <h1>${first ? 'Enter it again' : 'Choose a 4-digit PIN'}</h1>
        <p class="lede">${first ? 'Type the same four digits to check them.' : 'You sign in with it on any screen, even a TV remote.'}</p>
        ${dots()}${keypad()}</div>`,
    signin: () => `${progress()}<div class="b-step">
        <h1>Welcome back</h1>
        <p class="b-mail" style="margin-top:12px">${esc(email)}</p>
        ${dots()}${keypad(cfg.https ? `<button class="key fn" data-act="passkey-in" aria-label="Use a passkey">${I('scan-face')}</button>` : '')}</div>`,
    passkey: () => `${progress()}<div class="b-step"><div class="b-fill"></div><div class="pk">
        <span class="pk-ic">${I('scan-face')}</span><h2>Sign in with your face next time</h2>
        <p>A passkey lets this phone sign you in with Face ID. Your PIN still works on every screen.</p></div><div class="b-fill"></div>
        <div class="b-foot"><button class="btn pri block" data-act="passkey-add">${I('key-round')}<span>Add a passkey</span></button><button class="btn ghost block" data-act="next">Not now</button></div></div>`,
    notify: () => `${progress()}<div class="b-step">
        <h1>How should we tell you a request is ready?</h1><p class="lede">You can change this later in your Account.</p>
        <div style="margin-top:26px">
          <button class="opt" role="switch" aria-checked="${s.notifyEmail}" data-act="toggle-email"><span class="opt-ic">${I('mail')}</span><span class="opt-t"><b>Email me</b><span>${esc(s.account ? s.account.email : email)}</span></span><span class="sw"><span></span></span></button>
          <div class="opt" role="switch" aria-checked="true" aria-disabled="true"><span class="opt-ic">${I('bell')}</span><span class="opt-t"><b>In SlipStream</b><span>Always on, in your inbox here</span></span><span class="sw"><span></span></span></div>
        </div>
        <div class="b-fill"></div><div class="b-foot"><button class="btn pri block" data-act="next"><span>Continue</span></button></div></div>`,
    first: () => `${progress()}<div class="b-step">
        <h1>What do you want to watch first?</h1><p class="lede">Pick one and we will ask for it. Popular this week:</p>
        <div class="pick">${PICKS.map((id) => `<button class="pick-i" data-pick="${id}" aria-pressed="${pick === id}" aria-label="${esc(W(id).title)}">${`<span class="art"><img src="${W(id).art.poster}" alt="" onerror="this.parentNode.classList.add('art-x');this.remove()"><span class="art-fb">${esc(W(id).title)}</span></span>`}<span class="chk">${I('check')}</span></button>`).join('')}</div>
        <div class="b-fill" style="min-height:20px"></div>
        <div class="b-foot"><button class="btn pri block" data-act="request" ${pick ? '' : 'disabled'}><span>${pick ? `Request ${esc(W(pick).title)}` : 'Pick one to request'}</span></button><button class="btn ghost block" data-act="skip">Skip, I will look around</button></div></div>`,
  };

  function draw() {
    if (link !== 'live') { view.innerHTML = deadCard(link); icons(); return; }
    view.innerHTML = STEPS[step()]();
    icons();
    root.scrollTop = 0;
    const f = view.querySelector('#b-name, #b-email');
    if (f) f.focus();
    pinBad = false;
  }

  function done(note) {
    resetAmbient();
    const portal = Portal(view, root, s, {
      homeWork: pick || 'dune-2',
      onRequest: (id, btn, p) => { request(s, id); p.refresh(); toast(root, `Requested ${W(id).title}.`); },
    });
    portal.render();
    if (note) toast(root, note, 'badge-check');
  }

  async function onDigit(k) {
    if (k === 'del') { pin = pin.slice(0, -1); pinMsg = ''; return redrawPin(); }
    if (pin.length >= 4) return;
    pin += k;
    pinMsg = '';
    redrawPin();
    if (pin.length < 4) return;
    await new Promise((r) => setTimeout(r, 220));
    if (step() === 'signin') {
      pinMsg = 'Signing in';
      redrawPin();
      await new Promise((r) => setTimeout(r, 500));
      signIn(s, { viaLink: !plain });
      return next();
    }
    if (!first) { first = pin; pin = ''; return draw(); }
    if (first !== pin) { first = ''; pin = ''; pinMsg = 'Those did not match, so choose a PIN again.'; pinBad = true; draw(); return; }
    pinMsg = 'Creating your Account';
    redrawPin();
    await new Promise((r) => setTimeout(r, 550));
    createAccount(s, { name, email });
    pin = first = '';
    next();
  }
  function redrawPin() {
    const d = view.querySelector('.dots');
    if (!d) return;
    d.querySelectorAll('span').forEach((sp, i) => sp.classList.toggle('on', i < pin.length));
    const m = view.querySelector('.dots-msg');
    m.textContent = pinMsg || `${pin.length} of 4`;
    m.classList.toggle('bad', false);
  }

  view.addEventListener('submit', (e) => {
    e.preventDefault();
    const v = view.querySelector('#b-name').value.trim();
    if (!v) { view.querySelector('[data-name-err]').hidden = false; view.querySelector('#b-name').focus(); return; }
    name = v;
    next();
  });

  view.addEventListener('click', async (e) => {
    const key = e.target.closest('[data-key]');
    if (key) { key.classList.add('hit'); setTimeout(() => key.classList.remove('hit'), 120); return onDigit(key.dataset.key); }
    const p = e.target.closest('[data-pick]');
    if (p) { pick = pick === p.dataset.pick ? null : p.dataset.pick; if (pick) setAmbient(root, pick); draw(); return; }
    const b = e.target.closest('[data-act]');
    if (!b || !view.contains(b)) return;
    const act = b.dataset.act;
    if (act === 'next') next();
    else if (act === 'back') back();
    else if (act === 'to-existing') { flow = FLOWS.existing; at = 1; draw(); }
    else if (act === 'to-new') { flow = FLOWS.new; at = 0; draw(); }
    else if (act === 'email-edit') { editingEmail = true; draw(); }
    else if (act === 'email-done') { const v = view.querySelector('#b-email').value.trim(); if (v) email = v; editingEmail = false; draw(); }
    else if (act === 'sign-out') { s.account = null; s.mine = []; flow = FLOWS.new; at = 0; draw(); }
    else if (act === 'confirm') { await busy(b); useLink(s); done(`Your Account now has ${D.grantsShort}.`); }
    else if (act === 'passkey-add') { if (await passkeySystem(root, cfg, 'create', s.account.email)) { s.account.passkey = true; toast(root, 'Passkey added.', 'key-round'); next(); } }
    else if (act === 'passkey-in') { if (await passkeySystem(root, cfg, 'get', email)) { signIn(s, { viaLink: !plain }); s.account.passkey = true; flow = flow.filter((x) => x !== 'passkey'); next(); } }
    else if (act === 'toggle-email') { s.notifyEmail = !s.notifyEmail; b.setAttribute('aria-checked', s.notifyEmail); }
    else if (act === 'request') { await busy(b, 450); request(s, pick); done(); toast(root, `Requested ${W(pick).title}.`); }
    else if (act === 'skip') done();
    else if (act === 'retry') { await busy(b, 900); link = 'live'; draw(); }
    else if (act === 'go-portal') { signIn(s, { viaLink: false }); done(); }
    else if (act === 'sign-in') { link = 'live'; plain = true; flow = ['welcome', 'signin']; at = 1; draw(); }
  });

  addEventListener('keydown', (e) => {
    if (!['pin', 'signin'].includes(step()) || link !== 'live' || !view.querySelector('.keys')) return;
    if (/^\d$/.test(e.key)) onDigit(e.key);
    else if (e.key === 'Backspace') onDigit('del');
  });

  draw();
}
