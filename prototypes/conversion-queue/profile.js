// PROTOTYPE: the Profile editor page that sits behind the backfill dialog. Cinema has just gained 720p mobile and the admin pressed Save.

export function profilePage(S) {
  const { D, esc, I, isPhone } = S;
  const p = D.profiles.find((x) => x.id === D.backfillAsk.profile);
  const t = D.contentTypes[p.contentType];
  const rows = [...p.targets, D.backfillAsk.addedTarget].map((id) => {
    const tg = D.targets[id], added = id === D.backfillAsk.addedTarget;
    const how = tg.sourcing === 'derived' ? `${I('replace')}Derived from ${esc(D.targets[tg.from].name)}` : `${I('download')}Downloaded`;
    return `<li class="pe-t${added ? ' pe-new' : ''}"><b>${esc(tg.name)}</b><span class="t3">${how}</span>${tg.settings ? `<span class="mono t3 pe-set">${esc(tg.settings.summary)}</span>` : ''}${added ? '<span class="chip chip-live">New</span>' : ''}</li>`;
  }).join('');
  return `<section class="pe${isPhone ? ' pe-ph' : ''}" aria-labelledby="pe-h">
    ${isPhone ? '' : `<div class="sec-h"><h1 id="pe-h" class="pe-h1">Cinema</h1><span class="chip">${I('star')}Default for ${esc(t.plural)}</span><span class="t3 num">${p.works} Works</span><span class="grow"></span><button class="btn ghost sm" data-toast="Discarding is not in this prototype">Discard</button><button class="btn pri sm" data-open-backfill>Save</button></div>`}
    <div class="pe-card">
      <h2 class="pe-sub">Targets</h2>
      <ul class="pe-list">${rows}</ul>
      <button class="btn ghost sm" data-toast="Adding a Target is not in this prototype">${I('plus')}Add a Target</button>
    </div>
    <div class="pe-card">
      <h2 class="pe-sub">Editions wanted</h2><p class="t2">Standard</p>
      <h2 class="pe-sub">Wanted Languages</h2><p class="t2">English · Spanish (forced only)</p>
    </div>
    ${isPhone ? '<div class="pe-save"><button class="btn pri" data-open-backfill>Save</button></div>' : ''}
  </section>`;
}
