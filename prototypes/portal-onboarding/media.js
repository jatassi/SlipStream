// PROTOTYPE, throwaway. Round 2: the Media Server pieces variant B uses.
// A fake plex.tv window (labelled as fake), the "How to watch" card, and the portal's Media Server card.
// Everything here is simulated in memory; no request leaves the page.

import * as D from './data.js';
import { I, esc, icons, toast } from './shared.js';

export const serverOf = (cfg) => (cfg.server === 'none' ? null : D.mediaServers[cfg.server]);

// What the Member's side knows about their Media Server access. stage: unset | waiting | granted.
export function mediaState(cfg) {
  return { stage: 'unset', account: null, dismissed: false, ms: cfg.ms };
}

export const msIcon = (kind) => `<span class="ms-ic ${kind.toLowerCase()}" aria-hidden="true">${I(kind === 'Plex' ? 'play' : 'tv')}</span>`;

// ───────────── fake plex.tv: a browser tab (ms=1, sign up on your own) or Plex's sign-in window (ms=2, 3)

// On a phone it is a full-page trip to plex.tv and back (Plex's forwarding flow); on a wide screen, a popup window.
export function plexWindow(root, mode, { email = D.invitee.email } = {}) {
  return new Promise((resolve) => {
    const wrap = document.createElement('div');
    wrap.className = `px ${root.classList.contains('lc-phone') ? 'px-page' : ''}`;
    const tab = mode === 'tab';
    wrap.innerHTML = `<div class="px-scrim"></div>
      <div class="px-win" role="dialog" aria-modal="true" aria-label="${tab ? 'plex.tv in a new tab' : 'Plex sign-in window'}, simulated">
        <div class="px-chrome">
          ${tab ? `<div class="px-tabs"><span class="px-tab">${I('layout-grid')}SlipStream</span><span class="px-tab on">${I('play')}Plex</span></div>` : ''}
          <div class="px-url">${I('lock')}<span data-px-url>${tab ? 'plex.tv/sign-up' : 'app.plex.tv/auth'}</span><button class="px-x" data-px-act="close" aria-label="Close">${I('x')}</button></div>
        </div>
        <p class="px-fake">${I('flask-conical')}Fake plex.tv, simulated for this prototype</p>
        <div class="px-body" data-px></div>
      </div>`;
    root.append(wrap);
    const body = wrap.querySelector('[data-px]');
    const url = wrap.querySelector('[data-px-url]');
    let made = false;
    const sso = `<button class="px-sso" data-px-act="sso">${I('globe')}Continue with Google</button><button class="px-sso" data-px-act="sso">${I('apple')}Continue with Apple</button><p class="px-or"><span>or use email</span></p>`;
    const SCREENS = {
      signin: () => `<p class="px-brand">Plex</p><h2>Sign in</h2>${tab ? '' : `<p class="px-sub">to continue to ${esc(D.server.name)}</p>`}${sso}
        <label class="px-f"><span>Email or username</span><input value="${esc(email)}" autocomplete="off"></label>
        <label class="px-f"><span>Password</span><input type="password" data-px-pw autocomplete="off"></label>
        <button class="px-go" data-px-act="signin">Sign in</button>
        <p class="px-alt">New to Plex? <button data-px-act="to-signup">Create a free account</button></p>`,
      signup: () => `<p class="px-brand">Plex</p><h2>Create a free account</h2>${tab ? '' : `<p class="px-sub">then continue to ${esc(D.server.name)}</p>`}${sso}
        <label class="px-f"><span>Email</span><input value="${esc(email)}" autocomplete="off"></label>
        <label class="px-f"><span>Password</span><input type="password" data-px-pw autocomplete="off"></label>
        <button class="px-go" data-px-act="signup">Create account</button>${tab ? '' : '<p class="px-inf">Making a new account inside Plex\'s sign-in is inferred, not yet tested.</p>'}
        <p class="px-alt">Have an account? <button data-px-act="to-signin">Sign in</button></p>`,
      made: () => `<div class="px-done">${I('circle-check')}<h2>You're all set</h2>
        <p>Your Plex username is <b>${esc(D.plexAccount.username)}</b>. You can also sign in with ${esc(email)}.</p></div>
        <button class="px-go" data-px-act="back">Back to the SlipStream tab</button>
        <p class="px-alt">Plex may suggest getting the app now. You can do that later.</p>`,
      allow: () => `<p class="px-brand">Plex</p><h2>Allow access?</h2>
        <p class="px-sub">${esc(D.server.name)} wants to use your Plex account.</p>
        <div class="px-acct"><span class="px-av">${esc(D.plexAccount.username[0].toUpperCase())}</span><div><b>${esc(D.plexAccount.username)}</b><span>${esc(D.plexAccount.email)}${made ? ' · new account' : ''}</span></div></div>
        <p class="px-fine">It sees your Plex username and email, so ${D.admin.name} can share his Plex with you.</p>
        <button class="px-go" data-px-act="allow">Allow</button><button class="px-no" data-px-act="close">Cancel</button>`,
    };
    const show = (k) => {
      body.innerHTML = SCREENS[k]();
      url.textContent = { signin: tab ? 'plex.tv/sign-in' : 'app.plex.tv/auth', signup: tab ? 'plex.tv/sign-up' : 'app.plex.tv/auth#sign-up', made: 'app.plex.tv/desktop', allow: 'app.plex.tv/auth#allow' }[k];
      icons();
    };
    const done = (v) => { wrap.classList.add('out'); setTimeout(() => { wrap.remove(); resolve(v); }, 240); };
    const busy = (b, ms) => { b.classList.add('busy'); b.textContent = 'One moment'; return new Promise((r) => setTimeout(r, ms)); };
    wrap.addEventListener('click', async (e) => {
      if (e.target.classList.contains('px-scrim')) return done(null);
      const b = e.target.closest('[data-px-act]');
      if (!b) return;
      const act = b.dataset.pxAct;
      if (act === 'close') done(null);
      else if (act === 'to-signup') show('signup');
      else if (act === 'to-signin') show('signin');
      else if (act === 'signup') { await busy(b, 700); made = true; show(tab ? 'made' : 'allow'); }
      else if (act === 'signin' || act === 'sso') { await busy(b, 600); show(tab ? 'made' : 'allow'); }
      else if (act === 'back') done({ ...D.plexAccount });
      else if (act === 'allow') { await busy(b, 500); done({ ...D.plexAccount, verified: true, made }); }
    });
    show(tab ? 'signup' : 'signin');
  });
}

// ───────────── How to watch: one card, after access is granted

export function howToWatch(server, account, { jfUser } = {}) {
  if (server.kind === 'Plex') {
    return `<div class="hw">
      <ol class="hw-steps">
        <li><span class="hw-n">1</span><div><b>Get Plex</b><span>The Plex app on your TV, phone or tablet, or app.plex.tv in a browser.</span>
          <span class="hw-dev">${I('tv')}TV${I('smartphone')}Phone${I('monitor')}Browser</span></div></li>
        <li><span class="hw-n">2</span><div><b>Sign in as ${esc(account.username)}</b><span>The Plex account ${D.admin.name} added. Your SlipStream PIN does not work in Plex.</span></div></li>
        <li><span class="hw-n">3</span><div><b>Pick ${esc(server.plexName)}</b><span>${D.admin.name}'s server sits in Plex's list of servers, beside Plex's own free movies. Its libraries are what you request here.</span></div></li>
      </ol>
      <p class="hw-away">${I('house')}<span>Away from home, Plex asks for Plex Pass on ${D.admin.name}'s server, or your own Plex Pass or Remote Watch Pass.</span><span class="fc">Unverified</span></p>
      <button class="btn pri block" data-act="open-ms">${I('external-link')}<span>Open Plex</span></button></div>`;
  }
  const host = server.publicUrl.replace(/^https?:\/\//, '');
  return `<div class="hw">
      <ol class="hw-steps">
        <li><span class="hw-n">1</span><div><b>Get Jellyfin</b><span>The Jellyfin app on your TV or phone, or a browser.</span>
          <span class="hw-dev">${I('tv')}TV${I('smartphone')}Phone${I('monitor')}Browser</span></div></li>
        <li><span class="hw-n">2</span><div><b>Add the server</b><span class="mono hw-copy">${esc(host)}</span></div></li>
        <li><span class="hw-n">3</span><div><b>Sign in as ${esc(jfUser || 'rosa')}</b><span>With the Jellyfin password you just set. Your SlipStream PIN does not work there.</span></div></li>
      </ol>
      <button class="btn pri block" data-act="open-ms">${I('external-link')}<span>Open Jellyfin</span></button></div>`;
}

export function openMs(root, server) {
  toast(root, server.kind === 'Plex' ? 'Here SlipStream opens app.plex.tv.' : `Here SlipStream opens ${server.publicUrl.replace(/^https?:\/\//, '')}.`, 'external-link');
}

// ───────────── the portal card: set up, waiting, or how to watch

export function mediaCard(server, m) {
  const plex = server.kind === 'Plex';
  const wait = m.ms === '1' ? `Waiting for ${D.admin.name} to add you on Plex` : `Waiting for ${D.admin.name} to share Plex with you`;
  if (m.stage === 'unset') {
    return `<section class="ms-card" aria-labelledby="ms-h">
      <div class="ms-head">${msIcon(server.kind)}<div><p class="ms-k">Watch on ${server.kind}</p><h2 id="ms-h">${plex ? 'Set up Plex to watch what you request' : 'Get your Jellyfin account'}</h2></div></div>
      <p class="ms-p">${plex ? `${D.admin.name}'s library plays in Plex. Sign in to Plex, or make a free account, and ${D.admin.name} can share it with you.` : `${D.admin.name}'s library plays in Jellyfin. Pick a username and password, and SlipStream makes the account.`}</p>
      <button class="btn pri block" data-act="ms-setup">${plex ? `${I('play')}<span>Continue with Plex</span>` : `<span>Get my Jellyfin account</span>`}</button></section>`;
  }
  if (m.stage === 'waiting') {
    return `<section class="ms-card wait" aria-labelledby="ms-h">
      <div class="ms-head">${msIcon(server.kind)}<div><p class="ms-k"><span class="ms-pulse"></span>Plex</p><h2 id="ms-h">${wait}</h2></div></div>
      <p class="ms-p">We sent ${D.admin.name} your Plex account, <b>${esc(m.account.username)}</b>${m.account.verified ? '' : ', as you typed it'}. We will email you when you're in. Requests work in the meantime.</p>
      <div class="ms-foot">${m.ms === '1' ? `<button class="btn sm ghost" data-act="ms-change">Wrong account? Change it</button>` : ''}<button class="proto-sim" data-act="sim-grant">${I('flask-conical')}Prototype: ${D.admin.name} grants access</button></div></section>`;
  }
  if (m.dismissed) {
    return `<section class="ms-card slim" aria-label="${server.kind}">${msIcon(server.kind)}<p><b>Watch on ${server.kind}</b><span>${plex ? `Signed in as ${esc(m.account.username)}` : `${esc(server.publicUrl.replace(/^https?:\/\//, ''))} as ${esc(m.jfUser)}`}</span></p><button class="btn sm" data-act="open-ms">Open</button></section>`;
  }
  return `<section class="ms-card granted" aria-labelledby="ms-h">
    <div class="ms-head">${msIcon(server.kind)}<div><p class="ms-k">${I('circle-check')}You're in on ${server.kind}</p><h2 id="ms-h">How to watch</h2></div><button class="ib" data-act="ms-dismiss" aria-label="Done">${I('x')}</button></div>
    ${howToWatch(server, m.account || {}, { jfUser: m.jfUser })}</section>`;
}
