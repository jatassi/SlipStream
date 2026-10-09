// PROTOTYPE, throwaway. The emails, HTML and plain-text parts side by side.
//   original / reissued: the Signup link email.  plex-admin: to the Admin when a Member reports a Plex account (wording follows ?ms).
//   plex-member: "You're in on Plex", to the Member when the Admin grants access.
// The HTML follows email-client rules: table layout, light scheme only, sRGB hex and px, no images, a bulletproof button.

import * as D from './data.js';
import { esc } from './shared.js';

const longDate = (d) => d.toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
const longWhen = (d) => `${longDate(d)} at ${d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })} ${D.zone.abbr}`;
const plex = D.mediaServers.plex;
const first = D.invitee.first;
const membersUrl = `${D.server.https}/settings/members#media-servers`;
const adminEmail = 'jackson@atassi.example';

// Each email is a list of blocks; html() and text() render the same list, so the two parts never drift.
function model(cfg) {
  if (cfg.email === 'plex-admin') {
    const checked = cfg.ms !== '1';
    return {
      title: 'Plex account to add', desc: `What ${D.admin.name} receives when ${first} reports a Plex account (ms=${cfg.ms}).`,
      to: adminEmail, sentAt: new Date('2026-10-08T21:52:00'),
      subject: `${first} wants to watch on Plex`,
      preheader: checked ? `${first} signed in with Plex. Share in one click.` : `${first} told us a Plex account. Add it in Plex.`,
      blocks: [
        { h: checked ? `Share Plex with ${first}` : `Add ${first} on Plex` },
        { p: checked ? `${D.invitee.name} signed in with Plex, so this is the account Plex confirmed.` : `${D.invitee.name} joined ${D.server.name} and told us a Plex account.` },
        { kv: [['Plex account', D.plexAccount.username], ...(checked ? [['Plex email', D.plexAccount.email]] : []), ['Checked', checked ? 'Yes, by Plex sign-in' : `No, ${first} typed it`], ['Plex server', plex.name]] },
        checked
          ? { p: `Share it from SlipStream in one click. SlipStream accepts the share for ${first}, then emails how to watch.` }
          : { p: `In Plex, open Settings, then Manage Library Access, and choose Grant Library Access. Paste the account and pick libraries. Then mark it as added, and we email ${first} how to watch.` },
        { button: [checked ? 'Share on Plex' : 'Mark as added', membersUrl] },
        { small: `You get this email because you are an Admin on ${D.server.name}.` },
      ],
      notes: [
        checked ? 'The share goes to the Member as a Plex friend, and SlipStream accepts it with the token from their Plex sign-in, so the Member never gets a Plex invite email.' : 'The account is shown exactly as typed. A typo surfaces only when Plex refuses it. Plex cannot share with an email that has no Plex account, so the Member makes one first.',
        'Jellyfin and Emby send no such email: SlipStream makes the account itself, so there is nothing to do.',
      ],
    };
  }
  if (cfg.email === 'plex-member') {
    return {
      title: "You're in on Plex", desc: `What ${D.invitee.email} receives when ${D.admin.name} grants access.`,
      to: D.invitee.email, sentAt: new Date('2026-10-09T08:30:00'),
      subject: "You're in on Plex",
      preheader: `${D.admin.name} added your Plex account. Here is how to watch.`,
      blocks: [
        { h: "You're in on Plex" },
        { p: `${D.admin.name} added your Plex account, ${D.plexAccount.username}, to ${plex.plexName}. Here is how to watch.` },
        { steps: [
          ['Get Plex', 'The Plex app on your TV, phone or tablet, or app.plex.tv in a browser.'],
          [`Sign in as ${D.plexAccount.username}`, 'Your SlipStream PIN does not work in Plex.'],
          [`Pick ${plex.plexName}`, `${D.admin.name}'s server sits in Plex's list of servers, beside Plex's own free movies.`],
        ] },
        { button: ['Open Plex', 'https://app.plex.tv'] },
        { small: "Watching away from home needs Plex Pass on the server, or your own Plex Pass or Remote Watch Pass." },
        { small: `Ask for more movies and series at ${D.server.https.replace('https://', '')}.` },
      ],
      notes: ['Sent once, when the Admin shares on Plex or marks the account as added.', 'The away-from-home line (Plex Pass or Remote Watch Pass, since April 2025) is unverified: sourced from search snippets of Plex support pages.', 'Plex server name, accounts and addresses are invented.'],
    };
  }
  const reissued = cfg.email === 'reissued';
  const url = `${D.server.https}/join/${reissued ? D.link.rotated : D.link.token}`;
  return {
    title: 'Signup link email', desc: `What ${D.invitee.email} receives when ${D.admin.name} issues a link.`,
    to: D.invitee.email, sentAt: reissued ? D.reissue.sentAt : D.link.sentAt,
    subject: reissued ? `A new link to join ${D.server.name}` : `${D.admin.name} invited you to ${D.server.name}`,
    preheader: 'Create your Account to ask for movies and series. The link works once, for seven days.',
    blocks: [
      { h: `${D.admin.name} invited you to ${D.server.name}` },
      { p: `Create your Account to ask for movies and series, and hear when they are ready to watch. ${D.grantsLine}` },
      ...(reissued ? [{ callout: `This replaces the link sent on ${longDate(D.link.sentAt)}, which no longer works.` }] : []),
      { button: ['Create your Account', url] },
      { small: `The link works once, until ${longWhen(reissued ? D.reissue.expiresAt : D.link.expiresAt)}.`, tight: true },
      { small: 'Already have an Account? Follow the link and sign in instead.' },
      { small: 'If you did not expect this email, you can ignore it. Nothing changes until the link is used.' },
    ],
    notes: [
      "The expiry is a date and time in the server's zone (" + D.zone.id + '), because the link dies at an instant, not at the end of a day.',
      'The server name, address and people are invented.',
    ],
  };
}

function html(m) {
  const font = 'font-family:Helvetica,Arial,sans-serif;mso-line-height-rule:exactly;';
  const p = (text, size = 16, color = '#3a3a3c', extra = '') => `<tr><td style="${font}font-size:${size}px;line-height:${Math.round(size * 1.5)}px;color:${color};padding:0 0 16px 0;${extra}">${esc(text)}</td></tr>`;
  const button = ([label, url]) => `<tr><td style="padding:4px 0 24px 0;">
          <!--[if mso]><v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" href="${url}" style="height:48px;v-text-anchor:middle;width:240px;" arcsize="50%" stroke="f" fillcolor="#1d1d1f"><center style="color:#ffffff;font-family:Arial,sans-serif;font-size:16px;font-weight:bold;">${esc(label)}</center></v:roundrect><![endif]-->
          <!--[if !mso]><!--><a href="${url}" style="${font}display:inline-block;background-color:#1d1d1f;color:#ffffff;font-size:16px;line-height:48px;font-weight:bold;text-decoration:none;border-radius:24px;padding:0 28px;">${esc(label)}</a><!--<![endif]-->
        </td></tr>`;
  const kv = (rows) => `<tr><td style="padding:0 0 20px 0;"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="border:1px solid #e3e0da;border-radius:10px;">
      ${rows.map(([k, v], i) => `<tr><td style="${font}font-size:14px;line-height:20px;color:#6b6b70;padding:10px 14px;width:120px;${i ? 'border-top:1px solid #ecebe8;' : ''}">${esc(k)}</td><td style="${font}font-size:15px;line-height:20px;color:#1d1d1f;font-weight:bold;padding:10px 14px;${i ? 'border-top:1px solid #ecebe8;' : ''}">${esc(v)}</td></tr>`).join('')}
    </table></td></tr>`;
  const steps = (list) => `<tr><td style="padding:0 0 12px 0;"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      ${list.map(([t, d], i) => `<tr><td valign="top" width="40" style="padding:0 0 14px 0;"><table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td align="center" width="28" height="28" style="${font}width:28px;height:28px;border-radius:14px;background-color:#1d1d1f;color:#ffffff;font-size:14px;line-height:28px;font-weight:bold;">${i + 1}</td></tr></table></td>
        <td valign="top" style="${font}padding:3px 0 14px 0;font-size:16px;line-height:22px;color:#1d1d1f;"><b>${esc(t)}</b><br><span style="font-size:15px;line-height:22px;color:#5a5a5f;">${esc(d)}</span></td></tr>`).join('')}
    </table></td></tr>`;
  const callout = (text) => `<tr><td style="padding:0 0 20px 0;"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr><td style="${font}font-size:15px;line-height:22px;color:#5c3d00;background-color:#fff4dc;border-left:3px solid #e0a020;padding:12px 14px;">${esc(text)}</td></tr></table></td></tr>`;
  const block = (b) => (b.h ? `<tr><td style="${font}font-size:24px;line-height:30px;font-weight:bold;color:#1d1d1f;padding:0 0 16px 0;">${esc(b.h)}</td></tr>`
    : b.p ? p(b.p)
      : b.small ? p(b.small, 14, '#5a5a5f', b.tight ? 'padding-bottom:6px;' : '')
        : b.kv ? kv(b.kv) : b.steps ? steps(b.steps) : b.callout ? callout(b.callout) : button(b.button));
  const link = m.blocks.find((b) => b.button).button[1];
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light only"><meta name="supported-color-schemes" content="light">
<title>${esc(m.subject)}</title>
<!--[if mso]><style>table,td,a{font-family:Arial,sans-serif !important;}</style><![endif]-->
</head>
<body style="margin:0;padding:0;background-color:#f4f2ee;color:#1d1d1f;">
<div style="display:none;max-height:0;overflow:hidden;mso-hide:all;font-size:1px;line-height:1px;color:#f4f2ee;">${esc(m.preheader)}</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f4f2ee;">
<tr><td align="center" style="padding:32px 16px;">
  <table role="presentation" width="560" cellpadding="0" cellspacing="0" border="0" style="width:100%;max-width:560px;">
    <tr><td style="${font}font-size:17px;line-height:24px;font-weight:bold;color:#1d1d1f;padding:0 0 16px 4px;">SlipStream</td></tr>
    <tr><td style="background-color:#ffffff;border:1px solid #e3e0da;border-radius:12px;padding:32px 32px 16px 32px;">
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
        ${m.blocks.map(block).join('\n        ')}
      </table>
    </td></tr>
    <tr><td style="${font}font-size:12px;line-height:18px;color:#8a8780;padding:18px 4px 0 4px;">If the button does not work, copy this link into your browser:<br><a href="${link}" style="color:#8a8780;text-decoration:underline;word-break:break-all;">${link}</a></td></tr>
    <tr><td style="${font}font-size:12px;line-height:18px;color:#8a8780;padding:12px 4px 0 4px;">Sent by ${esc(D.server.name)}.</td></tr>
  </table>
</td></tr></table>
</body></html>`;
}

function text(m) {
  const out = [];
  for (const b of m.blocks) {
    if (b.h || b.p || b.callout) out.push(b.h || b.p || b.callout, '');
    else if (b.small) out.push(b.small, ...(b.tight ? [] : ['']));
    else if (b.kv) out.push(...b.kv.map(([k, v]) => `${k}: ${v}`), '');
    else if (b.steps) out.push(...b.steps.flatMap(([t, d], i) => [`${i + 1}. ${t}`, `   ${d}`]), '');
    else if (b.button) out.push(`${b.button[0]}:`, b.button[1], '');
  }
  out.push(`Sent by ${D.server.name}.`);
  return out.join('\n');
}

const TABS = [['original', 'Signup link'], ['reissued', 'Re-issued'], ['plex-admin', 'Plex account to add'], ['plex-member', "You're in on Plex"]];

export function mount({ root, cfg, navigate }) {
  const m = model(cfg);
  root.className = 'ev';
  root.innerHTML = `
    <header class="ev-h"><h1>${esc(m.title)}</h1><p>${esc(m.desc)}</p>
      <div class="ev-seg" role="group" aria-label="Email">${TABS.map(([v, l]) => `<button aria-pressed="${cfg.email === v}" data-v="${v}">${esc(l)}</button>`).join('')}</div></header>
    <div class="ev-grid">
      <section class="ev-pane" aria-label="HTML part">
        <div class="ev-pane-h"><span>HTML part</span><span>text/html</span></div>
        <dl class="ev-env">
          <dt>From</dt><dd>${esc(D.server.name)} &lt;${D.server.from}&gt;<span class="ev-tag">invented</span></dd>
          <dt>To</dt><dd>${esc(m.to)}</dd>
          <dt>Date</dt><dd>${longWhen(m.sentAt)}</dd>
          <dt>Subject</dt><dd class="subj">${esc(m.subject)}</dd>
        </dl>
        <iframe class="ev-frame" title="HTML email" sandbox=""></iframe>
      </section>
      <section class="ev-pane" aria-label="Plain-text part">
        <div class="ev-pane-h"><span>Plain-text part</span><span>text/plain</span></div>
        <pre class="ev-txt"></pre>
      </section>
    </div>
    <ul class="ev-notes">
      <li>Light scheme only, sRGB hex and px, no images, table layout; the button is a padded link with a VML fallback for Outlook.</li>
      ${m.notes.map((n) => `<li>${esc(n)}</li>`).join('')}
    </ul>`;
  root.querySelector('iframe').srcdoc = html(m);
  root.querySelector('.ev-txt').textContent = text(m);
  root.querySelectorAll('[data-v]').forEach((b) => { b.onclick = () => navigate({ email: b.dataset.v }); });
}
