// PROTOTYPE, throwaway. Variant A · One card: everything on one card over the artwork, then a passkey sheet, then the portal with search focused.

import * as D from './data.js';
import { AMBIENT, I, Portal, busy, checkSignIn, checkSignUp, createAccount, deadCard, emailField, esc, expiryShort, host, icons, nameField, openSheet, passkeySystem, pinField, readForm, request, session, setAmbient, showErrors, signIn, statusbar, toast, useLink, W, wirePins, wordmark } from './shared.js';

export function mount({ root, cfg }) {
  const s = session(cfg);
  root.className = `lc lc-${cfg.device}`;
  root.innerHTML = AMBIENT + statusbar() + '<div data-view></div>';
  const view = root.querySelector('[data-view]');
  setAmbient(root, 'dune-2');
  let mode = cfg.state === 'signedin' ? 'confirm' : cfg.state === 'existing' ? 'signin' : 'signup';
  let link = cfg.link;
  let portal = null;
  let plain = false; // signing in after a used link: no link, nothing gained

  const gifts = `<div class="a-gives"><span class="gift">${I('badge-check')}Auto-approved requests</span><span class="gift">${I('calendar-range')}Up to ${D.grants.quota} a week</span></div>`;
  const from = `<p class="a-from"><span class="avatar">${D.admin.initials}</span><span><b>${D.admin.name}</b> invited you</span></p>`;

  const cards = {
    signup: () => `${from}
      <h1>Join ${D.server.name}</h1>
      <p class="a-lede">Ask for movies and series, and hear when they are ready to watch.</p>${gifts}
      <form class="a-form" novalidate data-form="signup">
        ${nameField()}${emailField(D.invitee.email)}${pinField()}
        <button class="btn pri block" type="submit"><span>Create Account</span></button>
      </form>
      <div class="a-alt"><button class="btn ghost block" data-act="to-signin">I already have an Account</button></div>`,
    signin: () => `${plain ? `<h1>Sign in</h1>` : `${from}
      <h1>Sign in to add this to your Account</h1>
      <p class="a-lede">This link gives your Account:</p>${gifts}`}
      <form class="a-form" novalidate data-form="signin">
        ${emailField(D.invitee.email, '')}${pinField('', 'PIN')}
        <button class="btn pri block" type="submit"><span>Sign in</span></button>
      </form>
      ${cfg.https ? `<p class="or">or</p><button class="btn block" data-act="passkey-in">${I('key-round')}<span>Sign in with a passkey</span></button>` : ''}
      ${plain ? '' : '<div class="a-alt"><button class="btn ghost block" data-act="to-signup">Create an Account instead</button></div>'}`,
    confirm: () => `<div class="who"><span class="avatar lg">${esc(s.account.initials)}</span><div><b>${esc(s.account.name)}</b><span>${esc(s.account.email)}</span></div></div>
      <h1>Add this to your Account?</h1>
      <p class="a-lede">${D.admin.name} sent you a link that gives your Account:</p>${gifts}
      <button class="btn pri block" data-act="confirm"><span>Add to my Account</span></button>
      <div class="a-alt"><button class="btn ghost block" data-act="sign-out">Not you? Sign out</button></div>`,
  };

  function draw() {
    if (link !== 'live') { view.innerHTML = deadCard(link); icons(); return; }
    view.innerHTML = `<div class="a">
      <div class="a-bd" aria-hidden="true"><img src="${W('dune-2').art.backdrop}" alt=""></div>
      <header class="a-top">${wordmark()}<span class="a-host">${cfg.https ? I('lock') : I('globe')}${host(cfg)}</span></header>
      <section class="a-card" aria-label="Signup link">${cards[mode]()}
        ${plain ? '' : `<p class="a-fine">This link works once, until ${expiryShort()}.</p>`}
      </section></div>`;
    icons();
    wirePins(view);
  }

  const hint = () => (s.mine.some((m) => m.fresh) ? '' : `Search for any movie or series and press Request. We will ${s.notifyEmail ? 'email you' : 'tell you'} when it is ready.`);

  function goPortal({ askPasskey, note } = {}) {
    root.scrollTop = 0;
    portal = Portal(view, root, s, {
      hint,
      onRequest: (id, btn, p) => {
        request(s, id);
        p.refresh();
        toast(root, `Requested ${W(id).title}.`);
        const fresh = root.querySelector('[data-fresh]');
        if (fresh && !p.work) fresh.scrollIntoView({ behavior: 'smooth', block: 'center' });
      },
    });
    portal.render();
    if (note) toast(root, note, 'badge-check');
    if (!askPasskey) { portal.render({ focus: true }); return; }
    const sheet = openSheet(root, `<div class="pk"><span class="pk-ic">${I('scan-face')}</span>
        <h2>Sign in with your face next time</h2>
        <p>A passkey lets this phone sign you in with Face ID. Your PIN still works on every screen.</p>
        <div class="pk-act"><button class="btn pri block" data-pk="add">${I('key-round')}<span>Add a passkey</span></button><button class="btn ghost block" data-pk="skip">Not now</button></div></div>`, { label: 'Add a passkey' });
    const finish = async (msg) => { await sheet.close(); if (msg) toast(root, msg, 'key-round'); portal.render({ focus: true }); };
    sheet.el.addEventListener('click', async (e) => {
      const b = e.target.closest('[data-pk]');
      if (!b) return;
      if (b.dataset.pk === 'skip') return finish();
      const ok = await passkeySystem(root, cfg, 'create', s.account.email);
      if (ok) { s.account.passkey = true; finish('Passkey added. This phone signs you in with Face ID.'); }
    });
  }

  view.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;
    const v = readForm(form);
    const btn = form.querySelector('[type=submit]');
    if (form.dataset.form === 'signup') {
      if (!showErrors(form, checkSignUp(s, v))) return;
      await busy(btn);
      createAccount(s, v);
      goPortal({ askPasskey: cfg.https });
    } else {
      if (!showErrors(form, checkSignIn(v))) return;
      await busy(btn);
      signIn(s, { viaLink: !plain });
      goPortal({ askPasskey: cfg.https && !s.account.passkey, note: plain ? '' : `Your Account now has ${D.grantsShort}.` });
    }
  });

  view.addEventListener('click', async (e) => {
    const b = e.target.closest('[data-act]');
    if (!b || portal) return;
    const act = b.dataset.act;
    if (act === 'to-signin') { mode = 'signin'; draw(); }
    else if (act === 'to-signup') { mode = 'signup'; draw(); }
    else if (act === 'sign-out') { s.account = null; s.mine = []; mode = 'signup'; draw(); }
    else if (act === 'confirm') { await busy(b); useLink(s); goPortal({ note: `Your Account now has ${D.grantsShort}.` }); }
    else if (act === 'passkey-in') { if (await passkeySystem(root, cfg, 'get', D.invitee.email)) { signIn(s, { viaLink: !plain }); s.account.passkey = true; goPortal({ note: plain ? '' : `Your Account now has ${D.grantsShort}.` }); } }
    else if (act === 'retry') { await busy(b, 900); link = 'live'; draw(); }
    else if (act === 'go-portal') { s.account = { ...D.existingAccount }; s.mine = D.existingAccount.requests.map((work) => ({ work, state: 'coming' })); goPortal(); }
    else if (act === 'sign-in') { link = 'live'; mode = 'signin'; plain = true; draw(); }
  });

  draw();
}
