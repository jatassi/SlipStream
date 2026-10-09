// PROTOTYPE, throwaway. The Admin's side, for context: issue a Signup link, then the pending row with the honest email outcome.

import * as D from './data.js';
import { AMBIENT, I, busy, esc, icons, setAmbient, statusbar, toast } from './shared.js';

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

  const settings = (p) => [p.autoApprove ? 'Auto-approve' : 'Admin approves', p.quota ? `${p.quota} requests a week` : 'No request limit'].join(' · ');

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
        <button class="btn pri" type="submit" style="align-self:flex-start">${I('send')}<span>Send Signup link</span></button>
      </form>
      <section class="panel" aria-labelledby="pend-h"><h2 id="pend-h">Waiting to join <span class="t3" style="font-weight:400">${pending.filter((p) => p.state === 'live').length}</span></h2>
        ${pending.map(row).join('<hr style="border:0;height:1px;background:var(--hair);margin:4px 0">')}
      </section></div>`;
    icons();
    if (flashIndex !== undefined) { const el = view.querySelector(`.pend[data-i="${flashIndex}"]`); if (el) { el.classList.add('flash'); el.style.borderRadius = '12px'; } }
  }

  view.addEventListener('click', async (e) => {
    const b = e.target.closest('[data-act]');
    if (!b) return;
    const i = Number(b.dataset.i);
    const act = b.dataset.act;
    if (act === 'auto') { autoApprove = !autoApprove; b.setAttribute('aria-checked', autoApprove); }
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
    pending.unshift({ email, quota, autoApprove, token: newToken(), sentAt: tick(), rotatedFrom: null, state: 'live' });
    draw(0);
  });

  draw();
}
