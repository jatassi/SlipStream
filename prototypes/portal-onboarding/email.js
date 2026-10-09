// PROTOTYPE, throwaway. The Signup link email, HTML and plain-text parts side by side, original or re-issued.
// The HTML follows email-client rules: table layout, light scheme only, sRGB hex and px, no images, a bulletproof button.

import * as D from './data.js';
import { esc } from './shared.js';

const longDate = (d) => d.toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
const longWhen = (d) => `${longDate(d)} at ${d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })} ${D.zone.abbr}`;

function model(reissued) {
  const token = reissued ? D.link.rotated : D.link.token;
  return {
    reissued,
    url: `${D.server.https}/join/${token}`,
    sentAt: reissued ? D.reissue.sentAt : D.link.sentAt,
    expiresAt: reissued ? D.reissue.expiresAt : D.link.expiresAt,
    subject: reissued ? `A new link to join ${D.server.name}` : `${D.admin.name} invited you to ${D.server.name}`,
    heading: `${D.admin.name} invited you to ${D.server.name}`,
    body: `Create your Account to ask for movies and series, and hear when they are ready to watch. ${D.grantsLine}`,
    replaces: reissued ? `This replaces the link sent on ${longDate(D.link.sentAt)}, which no longer works.` : '',
    expiry: `The link works once, until ${longWhen(reissued ? D.reissue.expiresAt : D.link.expiresAt)}.`,
    existing: 'Already have an Account? Follow the link and sign in instead.',
    ignore: 'If you did not expect this email, you can ignore it. Nothing changes until the link is used.',
  };
}

function html(m) {
  const font = "font-family:Helvetica,Arial,sans-serif;mso-line-height-rule:exactly;";
  const p = (text, size = 16, color = '#3a3a3c', extra = '') => `<tr><td style="${font}font-size:${size}px;line-height:${Math.round(size * 1.5)}px;color:${color};padding:0 0 16px 0;${extra}">${esc(text)}</td></tr>`;
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light only"><meta name="supported-color-schemes" content="light">
<title>${esc(m.subject)}</title>
<!--[if mso]><style>table,td,a{font-family:Arial,sans-serif !important;}</style><![endif]-->
</head>
<body style="margin:0;padding:0;background-color:#f4f2ee;color:#1d1d1f;">
<div style="display:none;max-height:0;overflow:hidden;mso-hide:all;font-size:1px;line-height:1px;color:#f4f2ee;">Create your Account to ask for movies and series. The link works once, for seven days.</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f4f2ee;">
<tr><td align="center" style="padding:32px 16px;">
  <table role="presentation" width="560" cellpadding="0" cellspacing="0" border="0" style="width:100%;max-width:560px;">
    <tr><td style="${font}font-size:17px;line-height:24px;font-weight:bold;color:#1d1d1f;padding:0 0 16px 4px;">SlipStream</td></tr>
    <tr><td style="background-color:#ffffff;border:1px solid #e3e0da;border-radius:12px;padding:32px 32px 16px 32px;">
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
        <tr><td style="${font}font-size:24px;line-height:30px;font-weight:bold;color:#1d1d1f;padding:0 0 16px 0;">${esc(m.heading)}</td></tr>
        ${p(m.body)}
        ${m.replaces ? `<tr><td style="padding:0 0 20px 0;"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr><td style="${font}font-size:15px;line-height:22px;color:#5c3d00;background-color:#fff4dc;border-left:3px solid #e0a020;padding:12px 14px;">${esc(m.replaces)}</td></tr></table></td></tr>` : ''}
        <tr><td style="padding:4px 0 24px 0;">
          <!--[if mso]><v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" href="${m.url}" style="height:48px;v-text-anchor:middle;width:240px;" arcsize="50%" stroke="f" fillcolor="#1d1d1f"><center style="color:#ffffff;font-family:Arial,sans-serif;font-size:16px;font-weight:bold;">Create your Account</center></v:roundrect><![endif]-->
          <!--[if !mso]><!--><a href="${m.url}" style="${font}display:inline-block;background-color:#1d1d1f;color:#ffffff;font-size:16px;line-height:48px;font-weight:bold;text-decoration:none;border-radius:24px;padding:0 28px;">Create your Account</a><!--<![endif]-->
        </td></tr>
        ${p(m.expiry, 14, '#5a5a5f', 'padding-bottom:6px;')}
        ${p(m.existing, 14, '#5a5a5f')}
        ${p(m.ignore, 14, '#5a5a5f')}
      </table>
    </td></tr>
    <tr><td style="${font}font-size:12px;line-height:18px;color:#8a8780;padding:18px 4px 0 4px;">If the button does not work, copy this link into your browser:<br><a href="${m.url}" style="color:#8a8780;text-decoration:underline;word-break:break-all;">${m.url}</a></td></tr>
    <tr><td style="${font}font-size:12px;line-height:18px;color:#8a8780;padding:12px 4px 0 4px;">Sent by ${esc(D.server.name)}.</td></tr>
  </table>
</td></tr></table>
</body></html>`;
}

function text(m) {
  return [
    m.heading, '',
    m.body, '',
    ...(m.replaces ? [m.replaces, ''] : []),
    'Create your Account:', m.url, '',
    m.expiry,
    m.existing, '',
    m.ignore, '',
    `Sent by ${D.server.name}.`,
  ].join('\n');
}

export function mount({ root, cfg, navigate }) {
  const m = model(cfg.email === 'reissued');
  root.className = 'ev';
  root.innerHTML = `
    <header class="ev-h"><h1>Signup link email</h1><p>What ${esc(D.invitee.email)} receives when ${D.admin.name} issues a link.</p>
      <div class="ev-seg" role="group" aria-label="Version"><button aria-pressed="${!m.reissued}" data-v="original">Original</button><button aria-pressed="${m.reissued}" data-v="reissued">Re-issued</button></div></header>
    <div class="ev-grid">
      <section class="ev-pane" aria-label="HTML part">
        <div class="ev-pane-h"><span>HTML part</span><span>text/html</span></div>
        <dl class="ev-env">
          <dt>From</dt><dd>${esc(D.server.name)} &lt;${D.server.from}&gt;<span class="ev-tag">invented</span></dd>
          <dt>To</dt><dd>${esc(D.invitee.email)}</dd>
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
      <li>The expiry is a date and time in the server's zone (${D.zone.id}), because the link dies at an instant, not at the end of a day.</li>
      <li>The server name, address and people are invented.</li>
    </ul>`;
  root.querySelector('iframe').srcdoc = html(m);
  root.querySelector('.ev-txt').textContent = text(m);
  root.querySelectorAll('[data-v]').forEach((b) => { b.onclick = () => navigate({ email: b.dataset.v }); });
}
