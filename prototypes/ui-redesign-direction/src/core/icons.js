// PROTOTYPE. Icons are Lucide (ISC). build.mjs inlines only the names the variants use,
// found by scanning the sources for SS.icon calls, into SS.ICONS.
(function () {
  const SS = window.SS;
  SS.icon = (name, { size = 18, stroke = 1.5, cls = '', title = '' } = {}) => {
    const node = SS.ICONS?.[name];
    if (!node) { console.warn('[icons] not bundled:', name); return ''; }
    const inner = node.map(([tag, attrs]) => `<${tag} ${Object.entries(attrs).map(([k, v]) => `${k}="${v}"`).join(' ')}/>`).join('');
    return `<svg class="ss-icon ${cls}" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${stroke}" stroke-linecap="round" stroke-linejoin="round" aria-hidden="${title ? 'false' : 'true'}"${title ? ` role="img" aria-label="${title}"` : ''}>${inner}</svg>`;
  };
})();
