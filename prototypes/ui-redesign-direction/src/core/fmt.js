// PROTOTYPE. Formatting shared by every variant, so numbers read the same everywhere.
(function () {
  const SS = window.SS;
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const bytes = (n, dp) => {
    if (!n) return '0 B';
    const i = Math.min(units.length - 1, Math.floor(Math.log10(Math.abs(n)) / 3));
    const v = n / 1000 ** i;
    return `${v.toFixed(dp ?? (v >= 100 ? 0 : 1))} ${units[i]}`;
  };
  const rate = (n) => (n ? `${bytes(n)}/s` : '0 B/s');
  const dur = (s) => {
    if (s == null || !isFinite(s)) return '';
    s = Math.max(0, Math.round(s));
    if (s < 60) return `${s} s`;
    const m = Math.floor(s / 60);
    if (m < 60) return `${m} min`;
    const h = Math.floor(m / 60);
    if (h < 24) return `${h} h ${m % 60} min`;
    return `${Math.floor(h / 24)} d ${h % 24} h`;
  };
  const runtime = (min) => (min ? `${Math.floor(min / 60)} h ${min % 60} min` : '');
  const ago = (date, now = SS.sim?.clock || SS.data.now) => {
    const s = (now - date) / 1000;
    if (s < 45) return 'just now';
    if (s < 3600) return `${Math.round(s / 60)} min ago`;
    if (s < 86400) return `${Math.round(s / 3600)} h ago`;
    const days = Math.round(s / 86400);
    if (days === 1) return 'yesterday';
    if (days < 7) return `${days} days ago`;
    return date.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
  };
  const until = (date, now = SS.sim?.clock || SS.data.now) => {
    const s = (date - now) / 1000;
    if (s < 0) return ago(date, now);
    if (s < 3600) return `in ${Math.max(1, Math.round(s / 60))} min`;
    const sameDay = date.toDateString() === now.toDateString();
    if (sameDay) return `tonight at ${time(date)}`;
    const tomorrow = new Date(now.getTime() + 86400e3).toDateString() === date.toDateString();
    if (tomorrow) return `tomorrow at ${time(date)}`;
    return date.toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short' });
  };
  const time = (date) => date.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
  const day = (date) => date.toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long' });
  const pct = (p, dp = 0) => `${(p * 100).toFixed(dp)}%`;
  const ep = (e) => (e ? `S${e[0]} E${e[1]}` : '');
  const sxe = (e) => (e ? `S${String(e[0]).padStart(2, '0')}E${String(e[1]).padStart(2, '0')}` : '');
  const plural = (n, one, many = one + 's') => `${n} ${n === 1 ? one : many}`;
  const list = (xs) => (xs.length < 2 ? xs.join('') : `${xs.slice(0, -1).join(', ')} and ${xs[xs.length - 1]}`);
  const initials = (name) => name.split(' ').map((p) => p[0]).slice(0, 2).join('');
  const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

  const statusLabel = { available: 'Available', upgradable: 'Upgradable', downloading: 'Downloading', missing: 'Missing', failed: 'Failed', unreleased: 'Unreleased' };
  const coverageLabel = { met: 'Met', upgradable: 'Upgradable', missing: 'Missing', downloading: 'Downloading', waiting: 'Derives after download', deriving: 'Deriving', unreleased: 'Unreleased', unwanted: 'Not in Profile' };
  const jobVerb = { download: 'Downloading', transfer: 'Transferring', import: 'Importing', transcode: 'Deriving', subtitles: 'Subtitles', search: 'Searching' };

  SS.fmt = { bytes, rate, dur, runtime, ago, until, time, day, pct, ep, sxe, plural, list, initials, esc, statusLabel, coverageLabel, jobVerb };
})();
