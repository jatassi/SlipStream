// PROTOTYPE live ticker. Advances running jobs once a second and patches annotated elements in place, so variants never re-render.
// Annotate: [data-live-progress="<jobId>"] gets --p (0..1) and aria-valuenow; [data-live-pct], [data-live-eta], [data-live-rate] get text.
import { jobs } from './data.js';

const fmtEta = (m) => (m == null ? '' : m <= 0 ? 'now' : m < 60 ? `${Math.ceil(m)} min` : `${Math.floor(m / 60)} h ${Math.round(m % 60)} min`);
const fmtRate = (j) => (j.rate == null ? '' : `${j.rate.toFixed(j.rate < 20 ? 1 : 0)} ${j.unit}`);
export const format = { eta: fmtEta, rate: fmtRate, pct: (p) => `${Math.floor(p * 100)}%` };

const listeners = new Set();
export const onTick = (fn) => listeners.add(fn);

function patch(root) {
  for (const j of jobs) {
    root.querySelectorAll(`[data-live-progress="${j.id}"]`).forEach((el) => { el.style.setProperty('--p', j.progress); el.setAttribute('aria-valuenow', Math.floor(j.progress * 100)); });
    root.querySelectorAll(`[data-live-pct="${j.id}"]`).forEach((el) => { el.textContent = fmtPct(j.progress); });
    root.querySelectorAll(`[data-live-eta="${j.id}"]`).forEach((el) => { el.textContent = fmtEta(j.etaMin); });
    root.querySelectorAll(`[data-live-rate="${j.id}"]`).forEach((el) => { el.textContent = fmtRate(j); });
  }
}
const fmtPct = format.pct;

export function startLive(root) {
  patch(root);
  setInterval(() => {
    for (const j of jobs) {
      if (j.status !== 'running' || j.etaMin == null) continue;
      const step = j.etaMin > 0 ? (1 - j.progress) / (j.etaMin * 60) : 0;
      j.progress = Math.min(0.995, j.progress + step * (0.8 + Math.random() * 0.4));
      if (j.rate != null) j.rate = Math.max(0.5, j.rate * (0.94 + Math.random() * 0.12));
      j.etaMin = Math.max(0, j.etaMin - 1 / 60);
    }
    patch(root);
    listeners.forEach((fn) => fn());
  }, 1000);
}
