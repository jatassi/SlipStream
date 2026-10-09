// PROTOTYPE, throwaway. The Admin's side, for context: issue a Signup link, then the pending row with the honest email outcome.
// Round 2: the link names the Media Servers it grants, and a Media Server access section lists Members waiting to watch (?ms, ?server).

import * as D from './data.js';
import { AMBIENT, I, busy, esc, icons, setAmbient, statusbar, toast } from './shared.js';
import { msIcon } from './media.js';

const when = (d) => `${d.toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short' })}, ${d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })}`;
const DAY = 86400000;
const newToken = () => btoa(String.fromCharCode(...crypto.getRandomValues(new Uint8Array(32)))).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
const QUOTAS = [['0', 'No limit'], ['3', '3 a week'], ['5', '5 a week'], ['10', '10 a week'], ['20', '20 a week']];

export function mount({ root, cfg }) {
  root.className = `lc lc-${cfg.device}`;
  root.innerHTML = AMBIENT + statusbar() + '<div data-view></div>';
  const view = root.querySelector('[data-view]');
  setAmbient(root, 'dune-2');
  let clock = D.now.getTime();
  const tick = () => new Date((clock += 60000));

  const pending = [{ email: D.invitee.email, quota: D.grants.quota, autoApprove: D.grants.autoApprove, token: D.link.token, sentAt: D.link.sentAt, rotatedFrom: null, state: 'live' }];
  let autoApprove = true;

  // Media Servers the link grants: only those switched on for Members can be ticked, and those start ticked.
  const MSS = Object.values(D.mediaServers).map((m) => ({ ...m, members: cfg.server === m.id }));
  let grantsMs = MSS.filter((m) => m.members).map((m) => m.id);
  pending[0].ms = [...grantsMs];
  const access = accessRows();

  const settings = (p) => [p.autoApprove ? 'Auto-approve' : 'Admin approves', p.quota ? `${p.quota} requests a week` : 'No request limit', ...(p.ms || []).map((id) => D.mediaServers[id].kind)].join(' · ');

  const msChecklist = () => `<fieldset class="msl"><legend class="fld-l">Media Servers</legend>
      ${MSS.map((m) => `<label class="msl-i ${m.members ? '' : 'off'}"><input type="checkbox" name="ms" value="${m.id}" ${grantsMs.includes(m.id) ? 'checked' : ''} ${m.members ? '' : 'disabled'}>
        <span class="msl-box">${I('check')}</span>${msIcon(m.kind)}<span class="msl-t"><b>${esc(m.name)}</b><span>${m.kind}${m.members ? (m.kind === 'Plex' ? (cfg.ms === '1' ? '. You add them on Plex once they tell us their account.' : '. You share it once they sign in with Plex.') : '. SlipStream makes their account.') : '. Not shown to Members.'}</span></span>
        ${m.members ? '' : `<a href="#" class="msl-on" onclick="return false">Show to Members</a>`}</label>`).join('')}
      <span class="fld-c">${MSS.some((m) => m.members) ? 'The new Member can watch on the ticked servers.' : 'No Media Server is shown to Members, so the new Member gets no Media Server step.'}</span></fieldset>`;

  // ── Media Server access: who is waiting to watch, per ?ms
  function accessRows() {
    if (cfg.server === 'none') return [];
    const rosa = { name: D.invitee.name, initials: D.invitee.initials, email: D.invitee.email, joined: 'Today, 21:52', isNew: true };
    const [dev, mina] = D.accessMembers;
    if (cfg.server === 'jellyfin') {
      return [{ ...rosa, jf: 'rosa', state: 'granted', at: 'Today, 21:53' }, { ...dev, jf: dev.jfUser, state: 'granted', at: 'Mon 5 Oct, 18:02' }, { ...mina, jf: mina.jfUser, state: 'granted', at: mina.grantedAt }];
    }
    if (cfg.ms === '1') return [{ ...rosa, plex: { username: D.plexAccount.username, verified: false }, state: 'waiting' }, { ...dev, plex: { username: dev.plex.email, verified: false }, state: 'waiting' }, { ...mina, plex: { username: mina.plex.username, verified: false }, state: 'granted', at: mina.grantedAt }];
    if (cfg.ms === '2') return [{ ...rosa, plex: { ...D.plexAccount, verified: true }, state: 'waiting' }, { ...dev, plex: { ...dev.plex, verified: true }, state: 'waiting' }, { ...mina, plex: { ...mina.plex, verified: true }, state: 'granted', at: mina.grantedAt }];
    return [{ ...rosa, state: 'unset' }, { ...dev, plex: { ...dev.plex, verified: true }, state: 'waiting' }, { ...mina, plex: { ...mina.plex, verified: true }, state: 'granted', at: mina.grantedAt }];
  }

  const plexKv = (r) => `<dl class="kv acc-kv"><dt>Plex</dt><dd><b class="mono">${esc(r.plex.username)}</b></dd>
      ${r.plex.email && r.plex.email !== r.plex.username ? `<dt>Email</dt><dd class="mono">${esc(r.plex.email)}</dd>` : ''}
      <dt>Source</dt><dd>${r.plex.verified ? `<span class="src ok">${I('badge-check')}Checked by Plex sign-in</span>` : `<span class="src typed">${I('pencil')}Typed by ${esc(r.name.split(' ')[0])}, not checked</span>`}</dd></dl>`;

  const how = `In Plex, open Settings, then Manage Library Access, and choose Grant Library Access. Paste the account, pick libraries, then mark it here.`;

  const accRow = (r, i) => {
    const first = r.name.split(' ')[0];
    const pill = r.state === 'granted' ? `<span class="pill ok">${I('circle-check')}Granted</span>` : r.state === 'waiting' ? `<span class="pill wait">${I('hourglass')}Waiting</span>` : `<span class="pill idle">${I('circle-dashed')}Not set up</span>`;
    const head = `<div class="pend-h"><span class="avatar">${r.initials}</span><div><b>${esc(r.name)}</b><span>Member since ${r.joined}</span></div>${pill}</div>`;
    if (r.jf) return `<div class="acc" data-r="${i}">${head}<p class="acc-done">${I('circle-check')}<span>SlipStream made <b class="mono">${esc(r.jf)}</b> on ${esc(D.mediaServers.jellyfin.name)}, ${r.at}. ${first} set the password.</span></p></div>`;
    if (r.state === 'unset') return `<div class="acc" data-r="${i}">${head}<p class="acc-note">${I('info')}<span>${first} hasn't connected Plex yet. The portal asks each time ${first} opens it. Nothing to do here until then.</span></p></div>`;
    const body = plexKv(r);
    if (r.state === 'granted') return `<div class="acc" data-r="${i}">${head}${body}<p class="acc-done">${I('circle-check')}<span>${(r.how || (cfg.ms === '1' ? 'mark' : 'share')) === 'share' ? `Shared on Plex as a friend ${r.at}. SlipStream accepted it for ${first}, so no Plex invite email.` : `Added on Plex ${r.at}.`} We emailed ${first} how to watch.</span></p></div>`;
    const copy = `<button class="btn sm" data-act="acc-copy" data-r="${i}">${I('copy')}<span>Copy</span></button>`;
    const mark = `<button class="btn sm ${cfg.ms === '1' ? 'pri' : ''}" data-act="acc-mark" data-r="${i}">${I('check')}<span>Mark as added</span></button>`;
    if (cfg.ms === '1') return `<div class="acc" data-r="${i}">${head}${body}<p class="acc-note">${I('info')}<span>${how}</span></p><div class="ad-act">${copy}${mark}</div></div>`;
    return `<div class="acc" data-r="${i}">${head}${body}
      <div class="ad-act"><button class="btn sm pri" data-act="acc-share" data-r="${i}">${I('share-2')}<span>Share on Plex</span></button></div>
      <details class="acc-man"><summary>Share by hand instead</summary><p class="acc-note">${I('info')}<span>${how}</span></p><div class="ad-act">${copy}${mark}</div></details></div>`;
  };

  const accessPanel = () => {
    if (cfg.server === 'none') return `<section class="panel" aria-labelledby="acc-h"><h2 id="acc-h">Media Server access</h2>
      <p class="acc-note">${I('info')}<span>No Media Server is shown to Members, so new Members skip the Media Server step. <a href="#" onclick="return false"><b>Media Servers</b></a></span></p></section>`;
    const srv = D.mediaServers[cfg.server];
    const todo = access.filter((r) => r.state === 'waiting').length;
    const lead = srv.kind === 'Jellyfin'
      ? 'Nothing to do. SlipStream makes each Member\'s Jellyfin account with your API key when they sign up.'
      : cfg.ms === '1' ? 'Members tell us their Plex account. Add each one in Plex, then mark it here.'
        : cfg.ms === '2' ? 'Members sign in with Plex during signup, so each account here is checked. Share in one click: SlipStream shares your libraries and accepts the share for them.'
          : 'Members connect Plex from the portal when they like. Share in one click once they do, and SlipStream accepts it for them.';
    return `<section class="panel" aria-labelledby="acc-h">
      <div class="acc-top"><h2 id="acc-h">Media Server access <span class="t3" style="font-weight:400">${todo ? `${todo} waiting` : ''}</span></h2><span class="acc-srv">${msIcon(srv.kind)}${esc(srv.name)}</span></div>
      <p class="ad-sub" style="margin:0">${lead}</p>
      ${access.map(accRow).join('<hr class="acc-hr">')}
      ${srv.kind === 'Plex' && !cfg.plexpass ? `<p class="mail off" role="status">${I('triangle-alert')}<span><b>Your Plex account has no Plex Pass.</b> Members can watch at your home, but away from it each one needs their own Plex Pass or a Remote Watch Pass. <span class="fc">Unverified</span></span></p>` : ''}
      ${srv.kind === 'Plex' ? `<p class="ad-note">${I('info')}<span>Members join your Plex as friends, with their own Plex account, never as Plex Home users.</span></p>` : ''}
      <p class="ad-note">${I('flask-conical')}<span>People and accounts here are invented.</span></p></section>`;
  };

  const mailLine = (p) => {
    if (cfg.mail === 'failed') return `<p class="mail failed" role="status">${I('mail-x')}<span><b>The email did not send.</b> The mail server answered <code>550 5.1.1 ${esc(p.email)}: Recipient address rejected</code>. Check the address, or copy the link and send it yourself.</span></p>`;
    if (cfg.mail === 'off') return `<p class="mail off" role="status">${I('mail-warning')}<span>Email is not set up, so copy the link and send it yourself. <a href="#" onclick="return false"><b>Set up email</b></a></span></p>`;
    return `<p class="mail sent" role="status">${I('mail-check')}<span>Emailed to ${esc(p.email)} at ${p.sentAt.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })}.</span></p>`;
  };

  const row = (p, i) => {
    if (p.state === 'cancelled') return `<div class="pend"><p class="gone">${I('link-2-off')}<span>Cancelled. The link sent to ${esc(p.email)} no longer works.</span></p></div>`;
    const url = `${cfg.https ? D.server.https : D.server.http}/join/${p.token}`;
    return `<div class="pend" data-i="${i}">
      <div class="pend-h"><span class="avatar">${esc(p.email[0].toUpperCase())}</span><div><b>${esc(p.email)}</b><span>${settings(p)}</span></div></div>
      <dl class="kv"><dt>Sent</dt><dd>${when(p.sentAt)}</dd><dt>Expires</dt><dd>${when(new Date(p.sentAt.getTime() + 7 * DAY))} ${D.zone.abbr}</dd></dl>
      ${p.rotatedFrom ? `<p class="rot">${I('rotate-cw')}Sent again. The link sent at ${p.rotatedFrom.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })} no longer works.</p>` : ''}
      ${mailLine(p)}
      <div class="linkbox"><label class="sr" for="lk-${i}">Signup link</label><input class="in" id="lk-${i}" readonly value="${url}"><button class="btn sm" data-act="copy" data-i="${i}">${I('copy')}<span>Copy link</span></button></div>
      <div class="ad-act"><button class="btn sm" data-act="again" data-i="${i}">${I('send')}<span>Send again</span></button><button class="btn sm ghost" data-act="cancel" data-i="${i}">Cancel</button></div>
      <p class="ad-note">${I('info')}<span>Whoever opens this link can create an Account, and it works once, for seven days. Sending again makes a new link, and this one stops working.</span></p>
    </div>`;
  };

  function draw(flashIndex) {
    view.innerHTML = `<div class="ad">
      <div><p class="ad-crumb">Settings ${I('chevron-right')} Members</p><h1 style="margin-top:6px">Members</h1><p class="ad-sub">People who ask for Works through the portal.</p></div>
      <form class="panel" data-form="invite" novalidate>
        <h2>Invite a Member</h2>
        <label class="fld" data-f="email"><span class="fld-l">Email</span><input class="in" name="email" type="email" placeholder="name@example.com" autocomplete="off"><span class="fld-c">We email the Signup link here. You can also copy it and send it yourself.</span></label>
        <div class="ad-row">
          <label class="fld"><span class="fld-l">Request limit</span><select class="in sel" name="quota">${QUOTAS.map(([v, l]) => `<option value="${v}" ${v === String(D.grants.quota) ? 'selected' : ''}>${l}</option>`).join('')}</select><span class="fld-c">Optional. Defaults to your Member settings.</span></label>
          <button type="button" class="tg" role="switch" aria-checked="${autoApprove}" data-act="auto"><span class="tg-t"><b>Auto-approve</b><span>Requests start without waiting for you</span></span><span class="sw"><span></span></span></button>
        </div>
        ${msChecklist()}
        <button class="btn pri" type="submit" style="align-self:flex-start">${I('send')}<span>Send Signup link</span></button>
      </form>
      <section class="panel" aria-labelledby="pend-h"><h2 id="pend-h">Waiting to join <span class="t3" style="font-weight:400">${pending.filter((p) => p.state === 'live').length}</span></h2>
        ${pending.map(row).join('<hr style="border:0;height:1px;background:var(--hair);margin:4px 0">')}
      </section>
      ${accessPanel()}</div>`;
    icons();
    if (flashIndex !== undefined) { const el = view.querySelector(`.pend[data-i="${flashIndex}"]`); if (el) { el.classList.add('flash'); el.style.borderRadius = '12px'; } }
  }

  view.addEventListener('click', async (e) => {
    const b = e.target.closest('[data-act]');
    if (!b) return;
    const i = Number(b.dataset.i ?? b.dataset.r);
    const act = b.dataset.act;
    if (act === 'auto') { autoApprove = !autoApprove; b.setAttribute('aria-checked', autoApprove); }
    else if (act === 'acc-copy') {
      const r = access[i];
      try { await navigator.clipboard.writeText(r.plex.username); } catch { /* clipboard refused in some browsers; the toast still names it */ }
      toast(root, `Copied ${r.plex.username}.`, 'copy');
    } else if (act === 'acc-share' || act === 'acc-mark') {
      await busy(b, act === 'acc-share' ? 1100 : 500);
      const r = access[i];
      Object.assign(r, { state: 'granted', at: `today, ${tick().toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })}`, how: act === 'acc-share' ? 'share' : 'mark' });
      draw();
      const el = view.querySelector(`.acc[data-r="${i}"]`);
      if (el) el.classList.add('flash');
      toast(root, act === 'acc-share' ? `Shared on Plex. We emailed ${r.name.split(' ')[0]}.` : `Marked as added. We emailed ${r.name.split(' ')[0]}.`, 'circle-check');
    }
    else if (act === 'copy') {
      const inp = view.querySelector(`#lk-${i}`);
      inp.select();
      try { await navigator.clipboard.writeText(inp.value); toast(root, 'Link copied.', 'copy'); } catch { toast(root, 'Copying was refused, so the link is selected for you to copy.', 'copy'); }
    } else if (act === 'again') {
      await busy(b, 500);
      const p = pending[i];
      p.rotatedFrom = p.sentAt;
      p.sentAt = tick();
      p.token = newToken();
      draw(i);
      toast(root, cfg.mail === 'sent' ? `New link sent to ${p.email}.` : 'New link made. The old one no longer works.', 'rotate-cw');
    } else if (act === 'cancel') {
      pending[i].state = 'cancelled';
      draw();
    }
  });

  view.addEventListener('change', (e) => {
    if (e.target.name === 'ms') grantsMs = [...view.querySelectorAll('input[name=ms]:checked')].map((x) => x.value);
  });

  view.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;
    const email = form.email.value.trim();
    const fld = form.querySelector('[data-f="email"]');
    fld.classList.remove('bad');
    fld.querySelector('.err')?.remove();
    if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) { fld.classList.add('bad'); fld.insertAdjacentHTML('beforeend', `<span class="err">${I('circle-alert')}Enter an email address.</span>`); icons(); return; }
    await busy(form.querySelector('[type=submit]'), 600);
    const quota = Number(form.quota.value);
    const same = pending.findIndex((p) => p.state === 'live' && p.email.toLowerCase() === email.toLowerCase());
    if (same >= 0) {
      const p = pending[same];
      Object.assign(p, { quota, autoApprove, rotatedFrom: p.sentAt, sentAt: tick(), token: newToken() });
      draw(same);
      return;
    }
    const ms = [...form.querySelectorAll('input[name=ms]:checked')].map((x) => x.value);
    pending.unshift({ email, quota, autoApprove, ms, token: newToken(), sentAt: tick(), rotatedFrom: null, state: 'live' });
    draw(0);
  });

  draw();
}
