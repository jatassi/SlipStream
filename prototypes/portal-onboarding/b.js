// PROTOTYPE, throwaway. Variant B · Guided steps: one question per screen, phone-native, a big keypad for the PIN, ending on a first request.
// Round 2 adds the Media Server step, three ways (?ms=1|2|3), for Plex or Jellyfin (?server=plex|jellyfin|none).
//   1 Tell us your Plex: have Plex? yes or no, a guided trip to (fake) plex.tv, then the Member types the account; the Admin adds it by hand.
//   2 Sign in with Plex: one button opens Plex's own (fake) sign-in window; SlipStream gets the verified account; the Admin shares in one click.
//   3 Later, in the portal: onboarding stays as round 1; the portal carries a "Watch on Plex" card that runs 2's sign-in.

import * as D from './data.js';
import { AMBIENT, I, Portal, busy, createAccount, deadCard, esc, expiryShort, icons, openSheet, passkeySystem, request, resetAmbient, session, setAmbient, signIn, statusbar, toast, useLink, W } from './shared.js';
import { howToWatch, mediaCard, mediaState, msIcon, openMs, plexWindow, serverOf } from './media.js';

const PICKS = ['the-matrix', 'the-dark-knight', 'hotd', 'eeaao', 'the-boys', 'got'];

export function mount({ root, cfg }) {
  const s = session(cfg);
  root.className = `lc lc-${cfg.device}`;
  root.innerHTML = AMBIENT + statusbar() + '<div data-view></div>';
  const view = root.querySelector('[data-view]');
  setAmbient(root, 'dune-2');

  // The flow is a list of steps; the welcome screen decides which list runs.
  const server = serverOf(cfg);
  s.media = mediaState(cfg);
  const mediaSteps = !server || cfg.ms === '3' ? [] : server.kind === 'Plex' ? ['media'] : ['media', 'watch'];
  const FLOWS = {
    new: ['welcome', 'name', 'pin', ...(cfg.https ? ['passkey'] : []), ...mediaSteps, 'notify', 'first'],
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
  let mstage = 'ask'; // inside the Plex step: ask | new | enter | sent
  let plexTyped = '';
  let jfUser = '', jfPw = '';
  let portal = null;

  const step = () => flow[at];
  let plain = false; // signing in after a used link: nothing gained
  // Existing Accounts finish on the portal with a note of what the link gave them.
  const next = () => { at += 1; if (at >= flow.length) return done(flow === FLOWS.new || plain ? '' : `Your Account now has ${D.grantsShort}.`); draw(); };
  const PLEX_BACK = { new: 'ask', enter: 'ask', sent: cfg.ms === '1' ? 'enter' : 'ask' };
  const back = () => {
    if (step() === 'media' && mstage !== 'ask' && PLEX_BACK[mstage]) { mstage = PLEX_BACK[mstage]; return draw(); }
    at = Math.max(0, at - 1); pin = first = pinMsg = ''; draw();
  };

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
    media: () => `${progress()}${server.kind === 'Plex' ? PLEX[cfg.ms][mstage]() : JF()}`,
    watch: () => `${progress()}<div class="b-step">
        <span class="b-ms">${msIcon(server.kind)}${I('circle-check')}Account made</span>
        <h1>You're in on ${server.kind}</h1><p class="lede">Here is how to watch, any time you like.</p>
        <div class="b-hw">${howToWatch(server, {}, { jfUser })}</div>
        <div class="b-fill"></div><div class="b-foot"><button class="btn block" data-act="next"><span>Continue</span></button></div></div>`,
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

  // ── the Media Server step. Plex has two shapes (ms=1 typed, ms=2 signed in); Jellyfin has one, lighter.
  const tag = () => `<span class="b-ms">${msIcon('Plex')}Plex</span>`;
  const connected = (a) => `<div class="who"><span class="avatar lg px-avatar">${esc(a.username[0].toUpperCase())}</span><div><b>${esc(a.username)}</b><span>${esc(a.email)}</span></div><span class="vbadge">${I('badge-check')}Checked by Plex</span></div>`;
  const PLEX = {
    1: {
      ask: () => `<div class="b-step">${tag()}
        <h1>Do you have a Plex account?</h1>
        <p class="lede">${D.admin.name}'s movies and series play in Plex. ${D.admin.name} adds your Plex account, then they show up in your Plex app.</p>
        <div style="margin-top:26px">
          <button class="opt" data-act="plex-have"><span class="opt-ic">${I('user-check')}</span><span class="opt-t"><b>Yes, I have Plex</b><span>Tell us your username or email</span></span>${I('chevron-right', 't3')}</button>
          <button class="opt" data-act="plex-new"><span class="opt-ic">${I('user-plus')}</span><span class="opt-t"><b>No, I'm new to Plex</b><span>Make an account. It's free.</span></span>${I('chevron-right', 't3')}</button>
        </div></div>`,
      new: () => `<div class="b-step">${tag()}
        <h1>Make a Plex account</h1>
        <p class="lede">Plex opens in a new tab. Come back to this tab when you're done.</p>
        <ol class="b-guide">
          <li><span class="hw-n">1</span><div><b>Choose Create a free account</b><span>Use your email, or Google or Apple.</span></div></li>
          <li><span class="hw-n">2</span><div><b>Note your Plex username</b><span>Plex shows it when you're done. Your email works too.</span></div></li>
          <li><span class="hw-n">3</span><div><b>Come back here</b><span>Tell us the username or email, and we pass it to ${D.admin.name}.</span></div></li>
        </ol>
        <div class="b-fill"></div>
        <div class="b-foot"><button class="btn pri block" data-act="plex-open-tab">${I('external-link')}<span>Open plex.tv</span></button><button class="btn ghost block" data-act="plex-have">I already made one</button></div></div>`,
      enter: () => `<form class="b-step" data-form="plex" novalidate>${tag()}
        <h1>What's your Plex username or email?</h1>
        <p class="lede">${D.admin.name} adds this account to his Plex. In Plex, it's under Account.</p>
        <label class="sr" for="b-plex">Plex username or email</label>
        <input class="big-in" id="b-plex" placeholder="Username or email" autocomplete="off" autocapitalize="off" spellcheck="false" enterkeyhint="next" value="${esc(plexTyped)}">
        <p class="err" data-plex-err hidden>${I('circle-alert')}Enter your Plex username or email.</p>
        <div class="b-fill"></div><div class="b-foot"><button class="btn pri block" type="submit"><span>Send to ${D.admin.name}</span></button></div></form>`,
      sent: () => `<div class="b-step"><div class="b-fill"></div><div class="pk">
        <span class="pk-ic">${I('send')}</span><h2>We told ${D.admin.name}</h2>
        <p>${D.admin.name} adds <b>${esc(s.media.account.username)}</b> on Plex. We will email you when you're in. You can ask for things now.</p></div><div class="b-fill"></div>
        <div class="b-foot"><button class="btn pri block" data-act="next"><span>Continue</span></button></div></div>`,
    },
    2: {
      ask: () => `<div class="b-step">${tag()}
        <h1>Watch on Plex</h1>
        <p class="lede">${D.admin.name}'s movies and series play in Plex. Sign in to Plex, and ${D.admin.name} can share his library with you.</p>
        <div class="b-fill"></div>
        <div class="b-link" aria-hidden="true"><span class="b-link-a">${I('user-round')}</span><span class="b-link-l"></span>${msIcon('Plex')}<span class="b-link-l"></span><span class="b-link-a">${I('house')}</span></div>
        <div class="b-fill"></div>
        <div class="b-foot"><button class="btn pri block" data-act="plex-auth">${I('play')}<span>Continue with Plex</span></button>
          <p class="a-fine">New to Plex? You can make a free account on the Plex page. <span class="fc">Inferred, not yet tested</span></p></div></div>`,
      sent: () => `<div class="b-step">${tag()}
        <h1>Plex is connected</h1>
        <p class="lede">${D.admin.name} shares his Plex with this account. We will email you when you're in. You can ask for things now.</p>
        <div style="margin-top:24px">${connected(s.media.account)}</div>
        <button class="btn ghost sm" style="align-self:flex-start;margin-top:8px" data-act="plex-auth">Use another Plex account</button>
        <div class="b-fill"></div><div class="b-foot"><button class="btn pri block" data-act="next"><span>Continue</span></button></div></div>`,
    },
  };
  const JF = () => `<form class="b-step" data-form="jf" novalidate>
      <span class="b-ms">${msIcon('Jellyfin')}Jellyfin</span>
      <h1>Your Jellyfin sign-in</h1>
      <p class="lede">${D.admin.name}'s library plays in Jellyfin. Pick a username and password, and SlipStream makes the account on ${esc(server.name)}.</p>
      <div class="b-jf">
        <label class="fld" data-f="jf-user"><span class="fld-l">Jellyfin username</span><input class="in" name="user" autocomplete="off" autocapitalize="off" spellcheck="false" value="${esc(jfUser || (name.split(/\s+/)[0] || '').toLowerCase())}"></label>
        <label class="fld" data-f="jf-pw"><span class="fld-l">Jellyfin password</span><input class="in" name="pw" type="password" autocomplete="new-password" value="${esc(jfPw)}"><span class="fld-c">For the Jellyfin app only. Your PIN stays for SlipStream.</span></label>
      </div>
      <div class="b-fill"></div><div class="b-foot"><button class="btn pri block" type="submit"><span>Make my Jellyfin account</span></button></div></form>`;

  function draw() {
    if (link !== 'live') { view.innerHTML = deadCard(link); icons(); return; }
    view.innerHTML = STEPS[step()]();
    icons();
    root.scrollTop = 0;
    const f = view.querySelector('#b-name, #b-email, #b-plex');
    if (f) f.focus();
    pinBad = false;
  }

  const showCard = () => server && flow === FLOWS.new;
  const readyLine = () => {
    if (!server) return 'In the library.';
    const m = s.media;
    if (!showCard() || m.stage === 'granted') return `In the library. Watch it on ${server.kind}.`;
    if (m.stage === 'waiting') return `In the library. You can watch it once ${D.admin.name} adds you on Plex.`;
    return `In the library. Set up ${server.kind} on Home to watch it.`;
  };

  function done(note) {
    resetAmbient();
    portal = Portal(view, root, s, {
      homeWork: pick || 'dune-2',
      banner: () => (showCard() ? mediaCard(server, s.media) : ''),
      readyLine,
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

  // The portal's Media Server card. The portal itself handles open, back and request.
  async function portalAct(act, b) {
    const m = s.media;
    if (act === 'open-ms') return openMs(root, server);
    if (act === 'ms-dismiss') { m.dismissed = true; return portal.refresh(); }
    if (act === 'sim-grant') {
      await busy(b, 500);
      m.stage = 'granted';
      portal.refresh();
      return toast(root, `You're in on Plex. We emailed you too.`, 'party-popper');
    }
    if (act === 'ms-change') {
      const sh = openSheet(root, `<form class="sh-form" data-sh="plex" novalidate><h2 class="sh-t">Change your Plex account</h2>
        <label class="fld" data-f="plex"><span class="fld-l">Plex username or email</span><input class="in" name="plex" autocomplete="off" autocapitalize="off" spellcheck="false" value="${esc(m.account.username)}"></label>
        <p class="fld-c">We tell ${D.admin.name} the new one.</p>
        <button class="btn pri block" type="submit"><span>Send to ${D.admin.name}</span></button><button type="button" class="btn ghost block" data-sh-x>Cancel</button></form>`, { label: 'Change Plex account', onScrim: (api) => api.close() });
      sh.el.querySelector('[data-sh-x]').onclick = () => sh.close();
      sh.el.querySelector('form').onsubmit = async (e) => {
        e.preventDefault();
        const v = e.target.plex.value.trim();
        if (!v) return;
        m.account = { username: v, email: v.includes('@') ? v : '', verified: false };
        await sh.close();
        portal.refresh();
        toast(root, `Sent ${v} to ${D.admin.name}.`, 'send');
      };
      return;
    }
    if (act !== 'ms-setup') return;
    if (server.kind === 'Plex') {
      const a = await plexWindow(root, 'auth', { email: s.account.email });
      if (!a) return;
      m.account = a; m.stage = 'waiting';
      portal.refresh();
      return toast(root, `Plex connected. We told ${D.admin.name}.`, 'badge-check');
    }
    const first = (s.account.first || 'rosa').toLowerCase();
    const sh = openSheet(root, `<form class="sh-form" data-sh="jf" novalidate>${'<span class="b-ms">' + msIcon('Jellyfin') + 'Jellyfin</span>'}<h2 class="sh-t">Your Jellyfin sign-in</h2>
      <p class="fld-c">SlipStream makes the account on ${esc(server.name)}.</p>
      <label class="fld" data-f="jf-user"><span class="fld-l">Jellyfin username</span><input class="in" name="user" autocomplete="off" autocapitalize="off" value="${esc(first)}"></label>
      <label class="fld" data-f="jf-pw"><span class="fld-l">Jellyfin password</span><input class="in" name="pw" type="password" autocomplete="new-password"><span class="fld-c">For the Jellyfin app only. Your PIN stays for SlipStream.</span></label>
      <button class="btn pri block" type="submit"><span>Make my Jellyfin account</span></button></form>`, { label: 'Jellyfin account', onScrim: (api) => api.close() });
    sh.el.querySelector('form').onsubmit = async (e) => {
      e.preventDefault();
      const f = e.target;
      if (f.pw.value.length < 4 || !f.user.value.trim()) { f.pw.focus(); return; }
      await busy(f.querySelector('[type=submit]'), 900);
      Object.assign(m, { stage: 'granted', jfUser: f.user.value.trim(), dismissed: false });
      await sh.close();
      portal.refresh();
      toast(root, `You're in on Jellyfin as ${m.jfUser}.`, 'circle-check');
    };
  }

  view.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target.dataset.form;
    if (form === 'plex') {
      const v = view.querySelector('#b-plex').value.trim();
      if (!v) { view.querySelector('[data-plex-err]').hidden = false; view.querySelector('#b-plex').focus(); return; }
      plexTyped = v;
      await busy(e.target.querySelector('[type=submit]'), 500);
      s.media.account = { username: v, email: v.includes('@') ? v : '', verified: false };
      s.media.stage = 'waiting';
      mstage = 'sent';
      return draw();
    }
    if (form === 'jf') {
      const f = e.target;
      const user = f.user.value.trim(), pw = f.pw.value;
      f.querySelectorAll('.err').forEach((x) => x.remove());
      const bad = (k, msg) => { f.querySelector(`[data-f="${k}"]`).insertAdjacentHTML('beforeend', `<span class="err">${I('circle-alert')}${msg}</span>`); icons(); };
      if (!/^[a-z0-9._-]{2,}$/i.test(user)) return bad('jf-user', 'Use two or more letters or digits.');
      if (pw.length < 4) return bad('jf-pw', 'Use at least four characters.');
      jfUser = user; jfPw = pw;
      const b = f.querySelector('[type=submit]');
      await busy(b, 900);
      s.media = { ...s.media, stage: 'granted', jfUser: user, dismissed: true };
      return next();
    }
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
    if (portal) return portalAct(act, b);
    if (act === 'next') next();
    else if (act === 'plex-have') { mstage = 'enter'; draw(); }
    else if (act === 'plex-new') { mstage = 'new'; draw(); }
    else if (act === 'plex-open-tab') {
      const a = await plexWindow(root, 'tab', { email: s.account.email });
      if (a) { plexTyped = a.username; mstage = 'enter'; draw(); toast(root, 'Welcome back. Now tell us that Plex account.', 'corner-down-left'); }
    } else if (act === 'plex-auth') {
      const a = await plexWindow(root, 'auth', { email: s.account.email });
      if (a) { s.media.account = a; s.media.stage = 'waiting'; mstage = 'sent'; draw(); }
    }
    else if (act === 'open-ms') openMs(root, server);
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
