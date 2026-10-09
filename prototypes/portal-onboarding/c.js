// PROTOTYPE, throwaway. Variant C · Request first: the link opens the portal itself. The Account is made only when they press Request,
// and that is the moment the link is used.

import * as D from './data.js';
import { AMBIENT, I, Portal, art, busy, checkSignIn, checkSignUp, createAccount, deadCard, emailField, esc, expiryShort, icons, nameField, openSheet, passkeySystem, pinField, readForm, request, session, setAmbient, showErrors, signIn, statusbar, toast, useLink, W, wirePins } from './shared.js';

export function mount({ root, cfg }) {
  const s = session(cfg);
  root.className = `lc lc-${cfg.device}`;
  root.innerHTML = AMBIENT + statusbar() + '<div data-view></div>';
  const view = root.querySelector('[data-view]');
  let link = cfg.link;
  let plain = false; // arrived from a used link: no banner, nothing to gain
  let portal = null;

  const banner = () => {
    if (s.linkUsed || plain) return '';
    if (s.account) return `<div class="banner" role="region" aria-label="Signup link"><span class="avatar">${D.admin.initials}</span><p><b>${D.admin.name}</b> sent you a link that adds ${D.grantsShort}.</p><button class="btn sm pri" data-act="confirm"><span>Add</span></button></div>`;
    if (cfg.state === 'existing') return `<div class="banner" role="region" aria-label="Signup link"><span class="avatar">${D.admin.initials}</span><p><b>${D.admin.name}</b> sent you a link. Sign in when you request, and your Account gets ${D.grantsShort}.</p></div>`;
    return `<div class="banner" role="region" aria-label="Signup link"><span class="avatar">${D.admin.initials}</span><p><b>${D.admin.name} invited you.</b> Look around, and request anything to create your Account.</p></div>`;
  };

  const sheetBody = (mode, id) => {
    const w = W(id);
    const head = (p) => `<button class="c-x" data-c="close" aria-label="Close">${I('x')}</button><div class="c-head">${art(w)}<div><h2>Request ${esc(w.title)}</h2><p>${p}</p></div></div>`;
    if (mode === 'signup') return `${head(`Create your Account to send it. ${D.grantsLine}`)}
      <form class="a-form" novalidate data-form="signup">${nameField()}${emailField(D.invitee.email)}${pinField()}
        <button class="btn pri block" type="submit"><span>Create Account and request</span></button></form>
      <button class="btn ghost block" data-c="to-signin">I already have an Account</button>
      <p class="a-fine">This link works once, until ${expiryShort()}.</p>`;
    return `${head(plain ? 'Sign in to send it.' : `Sign in to send it, and your Account gets ${D.grantsShort}.`)}
      <form class="a-form" novalidate data-form="signin">${emailField(D.invitee.email, '')}${pinField('', 'PIN')}
        <button class="btn pri block" type="submit"><span>Sign in and request</span></button></form>
      ${cfg.https ? `<p class="or">or</p><button class="btn block" data-c="passkey-in">${I('key-round')}<span>Sign in with a passkey</span></button>` : ''}
      ${plain ? '' : '<button class="btn ghost block" data-c="to-signup">Create an Account instead</button>'}`;
  };

  async function land(sheet, id, { created }) {
    request(s, id);
    const fromLink = !plain;
    if (cfg.https && !s.account.passkey) {
      sheet.set(`<div class="pk"><span class="pk-ic">${I('check')}</span>
        <h2>${created ? 'Your Account is ready' : `Welcome back, ${esc(s.account.first)}`}</h2>
        <p>${esc(W(id).title)} is requested. Add a passkey so this phone signs you in with Face ID next time.</p>
        <div class="pk-act"><button class="btn pri block" data-c="pk-add">${I('key-round')}<span>Add a passkey</span></button><button class="btn ghost block" data-c="pk-skip">Not now</button></div></div>`);
      return;
    }
    await sheet.close();
    finish(id, fromLink && !created);
  }

  function finish(id, gained) {
    portal.refresh();
    toast(root, gained ? `Requested ${W(id).title}. Your Account now has ${D.grantsShort}.` : `Requested ${W(id).title}.`);
  }

  function askAccount(id) {
    let mode = cfg.state === 'existing' || plain ? 'signin' : 'signup';
    const sheet = openSheet(root, sheetBody(mode, id), { label: `Request ${W(id).title}`, onScrim: (sh) => sh.close() });
    wirePins(sheet.el);
    const redraw = () => { sheet.set(sheetBody(mode, id)); wirePins(sheet.el); };
    sheet.el.addEventListener('submit', async (e) => {
      e.preventDefault();
      const form = e.target;
      const v = readForm(form);
      const btn = form.querySelector('[type=submit]');
      if (form.dataset.form === 'signup') {
        if (!showErrors(form, checkSignUp(s, v))) return;
        await busy(btn);
        createAccount(s, v);
        land(sheet, id, { created: true });
      } else {
        if (!showErrors(form, checkSignIn(v))) return;
        await busy(btn);
        signIn(s, { viaLink: !plain });
        land(sheet, id, { created: false });
      }
    });
    sheet.el.addEventListener('click', async (e) => {
      const b = e.target.closest('[data-c], [data-act="to-signin"]');
      if (!b) return;
      const c = b.dataset.c || 'to-signin';
      if (c === 'close') sheet.close();
      else if (c === 'to-signin') { mode = 'signin'; redraw(); }
      else if (c === 'to-signup') { mode = 'signup'; redraw(); }
      else if (c === 'passkey-in') { if (await passkeySystem(root, cfg, 'get', D.invitee.email)) { signIn(s, { viaLink: !plain }); s.account.passkey = true; land(sheet, id, { created: false }); } }
      else if (c === 'pk-add') { if (await passkeySystem(root, cfg, 'create', s.account.email)) { s.account.passkey = true; await sheet.close(); finish(id, !plain && cfg.state !== 'new'); toast(root, 'Passkey added. You are in.', 'key-round'); } }
      else if (c === 'pk-skip') { await sheet.close(); finish(id, !plain && cfg.state !== 'new'); }
    });
  }

  function openPortal() {
    portal = Portal(view, root, s, {
      banner,
      onRequest: (id, btn, p) => {
        if (!s.account) return askAccount(id);
        request(s, id);
        p.refresh();
        toast(root, `Requested ${W(id).title}.`);
      },
    });
    portal.render();
  }

  view.addEventListener('click', async (e) => {
    const b = e.target.closest('[data-act]');
    if (!b) return;
    const act = b.dataset.act;
    if (act === 'confirm') { await busy(b, 500); useLink(s); portal.refresh(); toast(root, `Your Account now has ${D.grantsShort}.`, 'badge-check'); }
    else if (act === 'retry') { await busy(b, 900); link = 'live'; openPortal(); }
    else if (act === 'go-portal') { signIn(s, { viaLink: false }); plain = true; openPortal(); }
    else if (act === 'sign-in') { plain = true; openPortal(); }
  });

  if (link !== 'live') { setAmbient(root, 'dune-2'); view.innerHTML = deadCard(link); icons(); return; }
  openPortal();
}
