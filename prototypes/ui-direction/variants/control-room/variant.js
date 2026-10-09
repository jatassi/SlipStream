// PROTOTYPE stub, replaced by the variant build.
export function render({ root, data, surface, device, screen }) {
  root.innerHTML = `<p style="color:#ccc;font:14px system-ui;padding:24px">control-room · ${surface} · ${device} · ${screen} · ${data.jobs.length} jobs <span data-live-pct="j-dune-conv"></span></p>`;
}
