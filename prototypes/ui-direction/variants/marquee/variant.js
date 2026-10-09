// PROTOTYPE, throwaway. Direction B · Marquee: artwork-first, the Works lead, operations live on and around the media.

export function render(ctx) {
  const { root, data, format, onTick, surface, device, screen, go, icons } = ctx;
  const W = data.works;
  const phone = device === 'phone';
  const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
  root.classList.add('mq', `mq-${device}`, `mq-${surface}`);

  const member = (id) => data.members.find((m) => m.id === id);
  const jobFor = (id) => data.jobs.find((j) => j.id === id);
  const target = (id) => data.targets.find((t) => t.id === id);
  const minsBetween = (a, b) => Math.round((new Date(b) - new Date(a)) / 60000);
  const ago = (at) => {
    const m = minsBetween(at, data.now);
    if (m < 60) return `${m} min ago`;
    if (m < 180) return `${Math.floor(m / 60)} h ago`;
    if (new Date(at).toDateString() === data.now.toDateString()) return `today ${clock(at)}`;
    return new Date(at).toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short' });
  };
  const clock = (at) => new Date(at).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
  const day = (at) => new Date(at).toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short' });

  const art = (src, title, cls = '') => `<img class="${cls}" src="${src}" alt="" decoding="async" data-fb="${title}">`;
  const logo = (w, tag = 'h1', cls = '') => `<${tag} class="mq-logo ${cls}"><img src="${w.art.logo}" alt="${w.title}" data-fb-logo="${w.title}"></${tag}>`;
  const line = (jobId, cls = '') => `<span class="mq-line ${cls}" role="progressbar" aria-label="Progress" aria-valuemin="0" aria-valuemax="100" data-live-progress="${jobId}"><span></span></span>`;
  const meta = (w) => (w.type === 'series' ? `Series · ${w.network}` : `Movie · ${w.year}`);

  const running = data.jobs.filter((j) => j.status === 'running');
  const failedJob = data.jobs.find((j) => j.status === 'failed');
  const errorH = data.health.find((h) => h.level === 'error');
  const warnH = data.health.find((h) => h.level === 'warn');
  const dune = W['dune-2'];
  const duneJob = jobFor('j-dune-conv');
  const pending = data.requests.filter((r) => r.state === 'pending');
  const degraded = data.nodes.filter((n) => n.status !== 'online');
  const nodeLine = `${data.nodes.length} Nodes · ${degraded.map((n) => `${n.name} ${n.status}`).join(', ') || 'all online'}`;

  // ---------- shared fragments ----------

  const stageTrack = (job) => {
    const at = data.stages.indexOf(job.stage);
    return `<ol class="mq-stages" aria-label="Stages">${data.stages.map((s, i) => `<li class="${i < at ? 'done' : i === at ? 'now' : ''}"${i === at ? ' aria-current="step"' : ''}>${i < at ? '<i data-lucide="check"></i>' : ''}${s}</li>`).join('')}</ol>`;
  };

  const nodesDetail = () => `<ul class="mq-nodes">${data.nodes.map((n) => `
    <li class="mq-node ${n.status}">
      <div class="mq-node-head"><span class="mq-dot ${n.status}" aria-hidden="true"></span><strong>${n.name}</strong><span class="mq-dim">${n.kind} · ${n.where}</span>
        <span class="mq-node-state ${n.status}">${n.status === 'online' ? 'Online' : 'Degraded'}</span></div>
      <p class="mq-node-detail">${n.detail}${n.gpu ? ` · ${n.gpu}` : ''} · CPU ${n.cpu}%</p>
      <ul class="mq-roles" aria-label="Roles">${n.roles.map((r) => `<li>${r}</li>`).join('')}</ul>
    </li>`).join('')}</ul>`;

  const failureCard = () => {
    const w = W[failedJob.work];
    return `
    <article class="mq-need mq-fail" aria-labelledby="need-fail-t">
      <button class="mq-need-art" data-go aria-label="Open ${w.title}">${art(w.art.poster, w.title)}</button>
      <div class="mq-need-body">
        <p class="mq-need-tag"><span class="mq-chip-fail"><i data-lucide="circle-x"></i>Failed at ${errorH.stage}</span><span class="mq-dim">${failedJob.target} · ${failedJob.item} · ${ago(errorH.at)}</span></p>
        <h3 id="need-fail-t"><button class="mq-linklike" data-go>${w.title}</button> · the 2160p HDR release was misdescribed</h3>
        <dl class="mq-cause">
          <div><dt>Cause</dt><dd>${failedJob.cause}</dd></div>
          <div><dt>Recovery</dt><dd>${failedJob.recovery}</dd></div>
        </dl>
        <p class="mq-release" title="${failedJob.release}">${failedJob.release}</p>
        <div class="mq-actions">
          ${phone ? `<button class="mq-btn mq-btn-fail" data-toast="Manual search for The Batman isn't in this prototype">Manual search</button>
          <button class="mq-btn mq-btn-ghost" data-sheet="sheet-fail" aria-haspopup="dialog">More actions</button>` : `
          <button class="mq-btn mq-btn-fail" data-toast="Manual search for The Batman isn't in this prototype">Manual search</button>
          <button class="mq-btn mq-btn-ghost" data-toast="The Blocklist isn't in this prototype">View Blocklist entry</button>
          <button class="mq-btn mq-btn-ghost" data-toast="Dismissed in a real build; kept here for review">Dismiss</button>`}
        </div>
      </div>
    </article>`;
  };

  const warnCard = () => `
    <article class="mq-need mq-warn" aria-labelledby="need-warn-t">
      <div class="mq-need-icon" aria-hidden="true"><i data-lucide="radio-tower"></i></div>
      <div class="mq-need-body">
        <p class="mq-need-tag"><span class="mq-chip-warn"><i data-lucide="triangle-alert"></i>${warnH.stage} degraded</span><span class="mq-dim">relay-fra · since ${clock(warnH.at)}</span></p>
        <h3 id="need-warn-t">Plex remote access unconfirmed on relay-fra</h3>
        <p class="mq-need-text">${warnH.body}</p>
        <div class="mq-actions">
          <button class="mq-btn mq-btn-warn" data-toast="Remote access test isn't in this prototype">Test remote access</button>
          <button class="mq-btn mq-btn-ghost" data-pop-open="nodes">Show Nodes</button>
        </div>
      </div>
    </article>`;

  const inflightCard = (j) => {
    const w = W[j.work];
    const req = j.requestedBy ? member(j.requestedBy) : null;
    const indeterminate = j.etaMin == null;
    return `
    <li class="mq-card mq-flight">
      <button class="mq-pt" data-go data-work="${w.id}" aria-label="${w.title}, ${j.item}, ${j.stage} on ${j.node}">
        <span class="mq-pt-art">${art(w.art.poster, w.title)}
          ${req ? `<span class="mq-for" title="Requested by ${req.name}"><span class="mq-av">${req.initials}</span>for ${req.name.split(' ')[0]}</span>` : ''}
          <span class="mq-pt-over">
            <span class="mq-pt-stage"><span>${j.stage}</span>${indeterminate ? '' : `<span class="mq-num" data-live-pct="${j.id}"></span>`}</span>
            ${indeterminate ? '<span class="mq-line mq-line-scan" aria-label="Searching"><span></span></span>' : line(j.id)}
          </span>
        </span>
      </button>
      <div class="mq-cap">
        <p class="mq-cap-t">${w.title}</p>
        <p class="mq-cap-s">${j.item} · ${j.target}</p>
        <p class="mq-cap-m"><i data-lucide="${j.stage === 'Conversion' ? 'cpu' : j.stage === 'Transfer' ? 'arrow-right-left' : j.stage === 'Import' ? 'hard-drive' : j.stage === 'Searching' ? 'scan-search' : 'download'}"></i>${j.node}${j.rate != null ? ` · <span class="mq-num" data-live-rate="${j.id}"></span>` : ''}${j.etaMin ? ` · <span class="mq-num" data-live-eta="${j.id}"></span>` : ''}</p>
        ${indeterminate || !j.rate ? `<p class="mq-cap-m mq-dim">${j.release}</p>` : ''}
      </div>
    </li>`;
  };

  const upNextCard = (u) => {
    const w = W[u.work];
    return `
    <li class="mq-card mq-wide-card">
      <button class="mq-pt mq-pt-land" data-go data-work="${w.id}" aria-label="${w.title}, ${u.item}, ${day(u.at)} ${clock(u.at)}">
        <span class="mq-pt-art">${art(w.art.backdrop, w.title)}
          <span class="mq-land-logo"><img src="${w.art.logo}" alt="" data-fb-logo="${w.title}"></span>
          <span class="mq-land-when"><strong>${u.label}</strong> ${clock(u.at)}</span>
        </span>
      </button>
      <div class="mq-cap"><p class="mq-cap-t">${u.item}</p><p class="mq-cap-s">${w.title} · ${w.network} · 1080p</p></div>
    </li>`;
  };

  const requestCard = (r) => {
    const w = W[r.work];
    const m = member(r.by);
    return `
    <li class="mq-card mq-req">
      <button class="mq-pt" data-go data-work="${w.id}" aria-label="${w.title}, requested by ${m.name}">
        <span class="mq-pt-art">${art(w.art.poster, w.title)}
          <span class="mq-for"><span class="mq-av">${m.initials}</span>${m.name.split(' ')[0]}</span>
        </span>
      </button>
      <div class="mq-cap">
        <p class="mq-cap-t">${w.title}</p>
        <p class="mq-cap-s">${m.name} · ${ago(r.at)}${r.note ? ` · ${r.note}` : ''}</p>
        ${r.state === 'pending' ? `<div class="mq-req-act">
          <button class="mq-btn mq-btn-sm mq-btn-solid" data-approve="${r.id}">Approve</button>
          <button class="mq-btn mq-btn-sm mq-btn-ghost" data-decline="${r.id}">Decline</button></div>`
          : `<p class="mq-cap-m"><i data-lucide="circle-check"></i>Approved</p>`}
      </div>
    </li>`;
  };

  const posterCard = (id, sub) => {
    const w = W[id];
    return `
    <li class="mq-card">
      <button class="mq-pt" data-go data-work="${w.id}" aria-label="${w.title}">
        <span class="mq-pt-art">${art(w.art.poster, w.title)}</span>
      </button>
      <div class="mq-cap"><p class="mq-cap-t">${w.title}</p><p class="mq-cap-s">${sub || meta(w)}</p></div>
    </li>`;
  };

  const shelf = (id, title, count, items, opts = {}) => `
    <section class="mq-shelf ${opts.cls || ''}" aria-labelledby="sh-${id}">
      <header class="mq-shelf-h"><h2 id="sh-${id}">${title}</h2>${count != null ? `<span class="mq-count">${count}</span>` : ''}${opts.note ? `<p class="mq-shelf-note">${opts.note}</p>` : ''}
        ${opts.more ? `<button class="mq-more" data-toast="${opts.more} isn't in this prototype">${opts.moreLabel || 'See all'}<i data-lucide="chevron-right"></i></button>` : ''}</header>
      <ul class="mq-row" role="list">${items}</ul>
    </section>`;

  const statusSill = () => `
    <div class="mq-sill" role="region" aria-label="Nodes and today">
      <button class="mq-sill-nodes" data-pop-open="nodes" aria-haspopup="${phone ? 'dialog' : 'true'}"><span class="mq-dot ${degraded.length ? 'degraded' : 'online'}" aria-hidden="true"></span>${nodeLine}<i data-lucide="chevron-down"></i></button>
      <span class="mq-sill-item mq-num"><i data-lucide="arrow-down"></i>${data.today.downloadRate} MB/s <i data-lucide="arrow-up"></i>${data.today.uploadRate} MB/s</span>
      <span class="mq-sill-item">Today <b class="mq-num">${data.today.imported}</b> imported · <b class="mq-num">${data.today.grabbed}</b> grabbed · <b class="mq-num">${data.today.conversions}</b> Conversions · <b class="mq-num mq-red-ink">${data.today.failed}</b> failed</span>
      <span class="mq-sill-item mq-store"><span class="mq-store-bar" aria-hidden="true"><span style="--v:${data.storage.usedTB / data.storage.totalTB}"></span></span><span class="mq-num">${data.storage.usedTB} of ${data.storage.totalTB} TB</span> · Vault</span>
    </div>`;

  const heroSentence = () => `
    <p class="mq-sentence"><strong>forge</strong> is making the 1080p SDR copy from the 2160p HDR file. <span class="mq-sentence-live"><span class="mq-num" data-live-pct="${duneJob.id}"></span> through, <span data-eta-phrase>about <span class="mq-num" data-live-eta="${duneJob.id}"></span> to go</span>.</span></p>`;

  // ---------- admin home ----------

  const adminHome = () => `
    <section class="mq-hero" aria-labelledby="hero-t">
      <div class="mq-hero-art" aria-hidden="true">${art(dune.art.backdrop, dune.title)}</div>
      <div class="mq-hero-copy">
        <p class="mq-hero-kind"><span class="mq-live-dot" aria-hidden="true"></span>In flight now · Conversion</p>
        ${logo(dune, 'h1', 'mq-hero-logo').replace('<h1 ', '<h1 id="hero-t" ')}
        ${heroSentence()}
        ${stageTrack(duneJob)}
        <div class="mq-actions">
          <button class="mq-btn mq-btn-solid" data-go><i data-lucide="clapperboard"></i>Open ${dune.title}</button>
          <button class="mq-btn mq-btn-ghost" data-toast="Activity isn't in this prototype">All activity</button>
        </div>
      </div>
      <div class="mq-hero-meter">
        <span class="mq-hero-meter-l">1080p SDR · Standard Edition · NVENC HEVC · tone map HDR10 → SDR</span>
        <span class="mq-hero-meter-r mq-num"><span data-live-rate="${duneJob.id}"></span> · ${duneJob.size}</span>
        ${line(duneJob.id, 'mq-line-hero')}
      </div>
    </section>
    ${statusSill()}
    <section class="mq-needs" aria-labelledby="needs-t">
      <header class="mq-shelf-h"><h2 id="needs-t">Needs you</h2><span class="mq-count">${data.health.length}</span></header>
      <div class="mq-needs-grid">${failureCard()}${warnCard()}</div>
    </section>
    ${shelf('flight', 'In flight', running.length, running.map(inflightCard).join(''), { more: 'Activity', moreLabel: 'Activity' })}
    ${shelf('next', 'Up next', data.upNext.length, data.upNext.map(upNextCard).join(''), { more: 'Calendar', moreLabel: 'Calendar', cls: 'mq-shelf-land' })}
    ${shelf('req', 'Requests from Members', `${pending.length} pending`, data.requests.map(requestCard).join(''), { more: 'Requests', moreLabel: 'Requests' })}
    ${shelf('recent', 'Recently added', null, data.recentlyAdded.map((id) => posterCard(id)).join(''), { more: 'History', moreLabel: 'History', note: `${data.library.movies} movies · ${data.library.series} series · ${data.library.episodes.toLocaleString('en-US')} episodes` })}
  `;

  // ---------- admin work ----------

  const t4k = dune.targets[0];
  const tSdr = dune.targets[1];
  const prof = data.profiles.find((p) => p.id === dune.profile);

  const tileMet = () => `
    <article class="mq-tile mq-tile-met" aria-label="Standard at 2160p HDR: met">
      <p class="mq-tile-target">2160p HDR<span class="mq-dim">Downloaded</span></p>
      <header class="mq-tile-h"><span class="mq-state met"><i data-lucide="circle-check"></i>Met</span><span class="mq-dim">Cutoff reached · score <b class="mq-num">${t4k.score}</b></span></header>
      <p class="mq-file" title="${t4k.file}">${t4k.file}</p>
      <dl class="mq-specs">
        <div><dt>Size</dt><dd class="mq-num">${t4k.size}</dd></div>
        <div><dt>Video</dt><dd>${t4k.video}</dd></div>
        <div><dt>Audio</dt><dd>${t4k.audio}</dd></div>
      </dl>
      <p class="mq-tile-foot"><i data-lucide="arrow-up"></i>${t4k.seeding}</p>
    </article>`;

  const tileConv = () => `
    <article class="mq-tile mq-tile-conv" aria-label="Standard at 1080p SDR: converting">
      <p class="mq-tile-target">1080p SDR<span class="mq-dim">Derived from 2160p HDR</span></p>
      <header class="mq-tile-h"><span class="mq-state conv"><span class="mq-live-dot" aria-hidden="true"></span>Converting</span><span class="mq-dim">on forge</span></header>
      <p class="mq-tile-big"><span class="mq-num" data-live-pct="${duneJob.id}"></span><span class="mq-tile-big-s"><span class="mq-num" data-live-rate="${duneJob.id}"></span> · <span class="mq-num" data-live-eta="${duneJob.id}"></span> left</span></p>
      ${line(duneJob.id, 'mq-line-tile')}
      <dl class="mq-specs">
        <div><dt>From</dt><dd>${tSdr.from}</dd></div>
        <div><dt>Via</dt><dd>${tSdr.via}</dd></div>
        <div><dt>Size</dt><dd class="mq-num">≈ ${tSdr.estSize}</dd></div>
      </dl>
      <div class="mq-tile-foot mq-actions">
        <button class="mq-btn mq-btn-sm mq-btn-ghost" data-toast="Pausing a Conversion isn't in this prototype">Pause Conversion</button>
      </div>
    </article>`;

  const matrix = () => `
    <section class="mq-matrix" aria-labelledby="mx-t">
      <header class="mq-matrix-h">
        <h2 id="mx-t">Editions × Targets</h2>
        <p class="mq-dim">${prof.name} Profile · 1 Edition × ${prof.targets.length} Targets · <b>1 of 2 met</b>, 1 converting</p>
      </header>
      <div class="mq-grid" role="table" aria-label="Edition by Target state">
        <div class="mq-grid-row mq-grid-head" role="row">
          <span role="columnheader" class="mq-grid-corner">Edition</span>
          ${prof.targets.map((id) => { const t = target(id); return `<span role="columnheader" class="mq-col-h"><strong>${t.name}</strong><span class="mq-dim">${t.sourcing === 'derived' ? `<i data-lucide="git-branch"></i>Derived from ${target(t.from).name}` : '<i data-lucide="download"></i>Downloaded'}</span></span>`; }).join('')}
        </div>
        <div class="mq-grid-row" role="row">
          <span role="rowheader" class="mq-row-h"><strong>Standard</strong><span class="mq-dim">Default</span></span>
          <div role="cell" class="mq-cell">${tileMet()}</div>
          <div role="cell" class="mq-cell mq-cell-derived"><span class="mq-derive" aria-hidden="true"><i data-lucide="arrow-right"></i></span>${tileConv()}</div>
        </div>
        <div class="mq-grid-row mq-grid-ghost" role="row">
          <span role="rowheader" class="mq-row-h"><span class="mq-dim">Not wanted</span></span>
          <div role="cell" class="mq-cell mq-cell-span"><button class="mq-add-ed" data-toast="Editing Editions isn't in this prototype"><i data-lucide="plus"></i>Want another Edition, such as Extended or IMAX</button></div>
        </div>
      </div>
    </section>`;

  const filesPanel = () => `
    <section class="mq-files" aria-labelledby="files-t">
      <h2 id="files-t">Files and specs</h2>
      <table>
        <thead><tr><th scope="col">File</th><th scope="col">Meets</th><th scope="col">Size</th><th scope="col">Video</th><th scope="col">Audio</th></tr></thead>
        <tbody>
          <tr><td class="mq-file">${t4k.file}</td><td data-label="Meets">2160p HDR</td><td class="mq-num" data-label="Size">${t4k.size}</td><td data-label="Video">${t4k.video}</td><td data-label="Audio">${t4k.audio}</td></tr>
          <tr><td class="mq-file mq-dim">Dune.Part.Two.2024.1080p.SDR.HEVC.mkv · being made</td><td data-label="Meets">1080p SDR</td><td class="mq-num" data-label="Size">≈ ${tSdr.estSize}</td><td data-label="Video">HEVC · SDR</td><td data-label="Audio">from source</td></tr>
        </tbody>
      </table>
      <div class="mq-subs"><h3>Subtitles</h3><ul>${dune.subtitles.map((s) => `<li><i data-lucide="captions"></i>${s.lang}<span class="mq-dim mq-num">${s.note}</span></li>`).join('')}</ul></div>
    </section>`;

  const histIcon = { grabbed: 'magnet', downloaded: 'download', transfer: 'arrow-right-left', imported: 'hard-drive', conversion: 'cpu' };
  const filmstrip = () => `
    <section class="mq-film" aria-labelledby="film-t">
      <header class="mq-shelf-h"><h2 id="film-t">History</h2><span class="mq-count">${dune.history.length}</span></header>
      <ol class="mq-film-strip">${dune.history.map((h, i) => `
        <li class="mq-frame ${i === dune.history.length - 1 ? 'now' : ''}">
          <span class="mq-frame-when"><b>${day(h.at)}</b> ${clock(h.at)}</span>
          <span class="mq-frame-dot" aria-hidden="true"><i data-lucide="${histIcon[h.kind]}"></i></span>
          <p class="mq-frame-text">${h.text}</p>
          <span class="mq-frame-node"><i data-lucide="server"></i>${h.node}</span>
        </li>`).join('')}</ol>
    </section>`;

  const workActions = () => phone ? `
      <div class="mq-actions">
        <button class="mq-btn mq-btn-solid" data-toast="Search started for missing Targets of Dune: Part Two"><i data-lucide="search"></i>Search</button>
        <button class="mq-btn mq-btn-ghost" data-sheet="sheet-work" aria-haspopup="dialog"><i data-lucide="ellipsis"></i>More</button>
      </div>` : `
      <div class="mq-actions">
        <button class="mq-btn mq-btn-solid" data-toast="Search started for missing Targets of Dune: Part Two"><i data-lucide="search"></i>Search</button>
        <button class="mq-btn mq-btn-ghost" data-toast="Manual search isn't in this prototype"><i data-lucide="list"></i>Manual search</button>
        <button class="mq-btn mq-btn-ghost" data-toast="Profiles aren't in this prototype"><i data-lucide="sliders-horizontal"></i>Edit Profile</button>
        <button class="mq-btn mq-btn-ghost mq-toggle" aria-pressed="true" data-monitor><i data-lucide="eye"></i><span>Monitored</span></button>
        <div class="mq-menu-wrap">
          <button class="mq-btn mq-btn-icon mq-btn-ghost" aria-label="More actions" aria-expanded="false" aria-controls="pop-work" data-pop="work"><i data-lucide="ellipsis"></i></button>
          <div class="mq-pop mq-pop-menu" id="pop-work" role="menu" hidden>
            <button role="menuitem" data-toast="Metadata refresh queued"><i data-lucide="refresh-cw"></i>Refresh metadata</button>
            <button role="menuitem" data-toast="Rename preview isn't in this prototype"><i data-lucide="file-pen"></i>Preview renames</button>
            <button role="menuitem" data-toast="Manual import isn't in this prototype"><i data-lucide="folder-input"></i>Manual import</button>
            <hr>
            <button role="menuitem" class="mq-danger" data-toast="Deleting isn't in this prototype"><i data-lucide="trash-2"></i>Delete Work and files</button>
          </div>
        </div>
      </div>`;

  const adminWork = () => `
    <section class="mq-whero" aria-labelledby="work-t">
      <div class="mq-whero-blur" aria-hidden="true" style="--bd:url('${new URL(dune.art.backdrop, location.href).href}')"></div>
      <div class="mq-hero-art mq-whero-art" aria-hidden="true">${art(dune.art.backdrop, dune.title)}</div>
      ${phone ? `<div class="mq-phbar" data-phbar><button class="mq-back" data-home aria-label="Back to Home"><i data-lucide="chevron-left"></i></button><span class="mq-phbar-t" aria-hidden="true">${dune.title}</span></div>` : ''}
      <div class="mq-whero-copy">
        ${logo(dune, 'h1', 'mq-work-logo').replace('<h1 ', '<h1 id="work-t" ')}
        <p class="mq-meta"><span>${dune.year}</span><span>${dune.runtime}</span><span class="mq-rate-badge">${dune.rating}</span><span>${dune.genres.join(', ')}</span><span>${prof.name} Profile</span></p>
        <p class="mq-overview">${dune.overview}</p>
        ${workActions()}
      </div>
    </section>
    <div class="mq-work-body">
      <div class="mq-work-main">
        <div class="mq-poster-col">
          <div class="mq-poster-big">${art(dune.art.poster, dune.title)}</div>
          <p class="mq-dim mq-poster-note"><i data-lucide="eye"></i>Monitored · added ${day(dune.history[0].at)}</p>
        </div>
        ${matrix()}
      </div>
      ${filmstrip()}
      ${filesPanel()}
    </div>`;

  // ---------- portal ----------

  const ribbon = { ready: ['Ready', 'circle-check'], coming: ['Coming', 'clock'], requested: ['Requested', 'hourglass'] };
  const outcome = (state, cls = '') => `<span class="mq-ribbon ${state} ${cls}"><i data-lucide="${ribbon[state][1]}"></i>${ribbon[state][0]}</span>`;
  const libraryState = { 'the-batman': 'ready', oppenheimer: 'coming', 'black-panther': 'requested', interstellar: 'ready', 'the-prestige': 'ready', 'dune-2': 'ready', 'top-gun-maverick': 'ready', bcs: 'ready', narcos: 'ready', severance: 'coming', tlou: 'ready' };

  const mineCard = (m) => {
    const w = W[m.work];
    return `
    <li class="mq-card mq-mine">
      <button class="mq-pt" data-go data-work="${w.id}" aria-label="${w.title}: ${ribbon[m.state][0]}. ${m.line}">
        <span class="mq-pt-art">${art(w.art.poster, w.title)}${outcome(m.state, 'mq-ribbon-band')}
          ${m.job ? line(m.job, 'mq-line-ribbon') : ''}</span>
      </button>
      <div class="mq-cap"><p class="mq-cap-t">${m.line}</p><p class="mq-cap-s">${w.title} · ${m.detail}</p></div>
    </li>`;
  };

  const requestControl = (id) => {
    const st = libraryState[id];
    if (st === 'ready') return `<span class="mq-reqc ready"><i data-lucide="circle-check"></i>Ready to watch</span>`;
    if (st === 'coming') return `<span class="mq-reqc coming"><i data-lucide="clock"></i>Coming soon</span>`;
    if (st === 'requested') return `<span class="mq-reqc requested"><i data-lucide="hourglass"></i>Requested</span>`;
    return `<button class="mq-reqc ask" data-request="${id}"><i data-lucide="plus"></i>Request</button>`;
  };

  const wallCard = (id) => {
    const w = W[id];
    return `
    <li class="mq-card mq-wall">
      <button class="mq-pt" data-go data-work="${w.id}" aria-label="${w.title}">
        <span class="mq-pt-art">${art(w.art.poster, w.title)}${libraryState[id] ? outcome(libraryState[id], 'mq-ribbon-corner') : ''}</span>
      </button>
      <div class="mq-cap"><p class="mq-cap-t">${w.title}</p><p class="mq-cap-s">${meta(w)}</p>${requestControl(id)}</div>
    </li>`;
  };

  const wallBg = ['dune-2', 'interstellar', 'severance', 'the-batman', 'oppenheimer', 'tlou', 'the-matrix', 'hotd', 'top-gun-maverick', 'bcs', 'the-dark-knight', 'got', 'eeaao', 'narcos', 'the-prestige', 'black-panther', 'the-boys', 'tlou'];
  const portalHome = () => `
    <section class="mq-search-hero" aria-labelledby="ps-t">
      <div class="mq-wall-bg" aria-hidden="true">${wallBg.map((id) => `<img src="${W[id].art.poster}" alt="" data-fb-hide>`).join('')}</div>
      <div class="mq-search-copy">
        <h1 id="ps-t">What are we watching, ${data.portal.member.name}?</h1>
        <form class="mq-bigsearch" role="search" data-search-form>
          <i data-lucide="search"></i>
          <label class="mq-sr" for="ps-in">Search movies and series</label>
          <input id="ps-in" type="search" placeholder="Search movies and series" autocomplete="off">
          <button class="mq-btn mq-btn-solid" type="submit">Search</button>
        </form>
        <p class="mq-search-hint">Find anything, and ask for it if it isn't here yet. We'll tell you when it's ready.</p>
      </div>
    </section>
    ${shelf('mine', 'Your requests', data.portal.mine.length, data.portal.mine.map(mineCard).join(''), { cls: 'mq-shelf-mine' })}
    ${shelf('new', 'New in the library', null, data.portal.newInLibrary.map((id) => posterCard(id, `${meta(W[id])} · Ready`)).join(''))}
    ${shelf('pop', 'Popular this week', null, data.portal.popular.map(wallCard).join(''), { cls: 'mq-shelf-wall' })}
  `;

  const more = ['interstellar', 'oppenheimer', 'the-matrix', 'black-panther', 'the-dark-knight', 'the-prestige'];
  const portalWork = () => `
    <section class="mq-whero mq-whero-portal" aria-labelledby="pw-t">
      <div class="mq-whero-blur" aria-hidden="true" style="--bd:url('${new URL(dune.art.backdrop, location.href).href}')"></div>
      <div class="mq-hero-art mq-whero-art" aria-hidden="true">${art(dune.art.backdrop, dune.title)}</div>
      ${phone ? `<div class="mq-phbar" data-phbar><button class="mq-back" data-home aria-label="Back to Home"><i data-lucide="chevron-left"></i></button><span class="mq-phbar-t" aria-hidden="true">${dune.title}</span></div>` : ''}
      <div class="mq-whero-copy">
        ${logo(dune, 'h1', 'mq-work-logo').replace('<h1 ', '<h1 id="pw-t" ')}
        <p class="mq-meta"><span>${dune.year}</span><span>${dune.runtime}</span><span class="mq-rate-badge">${dune.rating}</span><span>${dune.genres.join(', ')}</span></p>
        <p class="mq-portal-state">${outcome('ready')}<span>${data.portalWork.line}</span></p>
        <p class="mq-overview">${dune.overview}</p>
        <div class="mq-actions">
          <button class="mq-btn mq-btn-solid mq-btn-lg" data-toast="Opens Dune: Part Two in Plex"><i data-lucide="play"></i>Watch on Plex</button>
        </div>
      </div>
    </section>
    <div class="mq-portal-body">
      <section class="mq-plain" aria-labelledby="plain-t">
        <h2 id="plain-t">What's here</h2>
        <ul>
          <li><i data-lucide="tv"></i><span><strong>In 4K HDR</strong>Looks its best on a 4K TV. Plex picks the right picture for every other screen.</span></li>
          <li><i data-lucide="captions"></i><span><strong>English and Spanish subtitles</strong>Timed to the picture, ready in Plex.</span></li>
          <li><i data-lucide="clock"></i><span><strong>${dune.runtime}</strong>Added ${day(dune.history[3].at)}.</span></li>
        </ul>
      </section>
      ${shelf('more', 'More like this', null, more.map(wallCard).join(''), { cls: 'mq-shelf-wall' })}
    </div>`;

  // ---------- shells ----------

  const adminNav = () => `
    <header class="mq-nav" data-nav>
      <a class="mq-wordmark" href="?variant=B&surface=admin&device=${device}&screen=home" data-home aria-label="SlipStream, Home"${screen === 'home' ? ' aria-current="page"' : ''}>Slip<span>Stream</span></a>
      <nav aria-label="Main"><ul class="mq-navlist">
        <li class="mq-menu-wrap"><button class="mq-navlink${screen === 'work' ? ' is-on' : ''}" aria-expanded="false" aria-controls="pop-lib" data-pop="lib">Library<i data-lucide="chevron-down"></i></button>
          <div class="mq-pop mq-pop-menu" id="pop-lib" role="menu" hidden>
            <button role="menuitem" data-toast="Movies isn't in this prototype"><i data-lucide="film"></i>Movies<span class="mq-num mq-dim">${data.library.movies}</span></button>
            <button role="menuitem" data-toast="Series isn't in this prototype"><i data-lucide="tv"></i>Series<span class="mq-num mq-dim">${data.library.series}</span></button>
          </div></li>
        <li><button class="mq-navlink" data-toast="Activity isn't in this prototype">Activity<span class="mq-badge mq-num">${running.length}</span></button></li>
        <li><button class="mq-navlink" data-toast="Calendar isn't in this prototype">Calendar</button></li>
        <li><button class="mq-navlink" data-toast="Wanted isn't in this prototype">Wanted</button></li>
        <li><button class="mq-navlink" data-toast="Requests isn't in this prototype">Requests<span class="mq-badge mq-num">${pending.length}</span></button></li>
        <li class="mq-menu-wrap"><button class="mq-navlink" aria-expanded="false" aria-controls="pop-nodes" data-pop="nodes">Nodes<span class="mq-dot degraded" aria-label="one degraded"></span></button>
          <div class="mq-pop mq-pop-nodes" id="pop-nodes" hidden><p class="mq-pop-h">${nodeLine}</p>${nodesDetail()}</div></li>
        <li class="mq-menu-wrap"><button class="mq-navlink" aria-expanded="false" aria-controls="pop-more" data-pop="more">More<i data-lucide="chevron-down"></i></button>
          <div class="mq-pop mq-pop-menu" id="pop-more" role="menu" hidden>
            <button role="menuitem" data-toast="History isn't in this prototype"><i data-lucide="history"></i>History</button>
            <button role="menuitem" data-toast="Manual import isn't in this prototype"><i data-lucide="folder-input"></i>Manual import</button>
            <button role="menuitem" data-toast="Settings isn't in this prototype"><i data-lucide="settings"></i>Settings</button>
            <button role="menuitem" data-toast="System isn't in this prototype"><i data-lucide="activity"></i>System</button>
          </div></li>
      </ul></nav>
      <button class="mq-searchbox" data-search aria-haspopup="dialog"><i data-lucide="search"></i><span>Search Works and releases</span><kbd>/</kbd></button>
      <div class="mq-menu-wrap">
        <button class="mq-avatar" aria-label="Account, ${data.admin.name}" aria-expanded="false" aria-controls="pop-acct" data-pop="acct">${data.admin.initials}</button>
        <div class="mq-pop mq-pop-menu mq-pop-right" id="pop-acct" role="menu" hidden>
          <p class="mq-pop-h">${data.admin.name}<span class="mq-dim">${data.admin.email} · Admin</span></p>
          <button role="menuitem" data-toast="Account isn't in this prototype"><i data-lucide="user"></i>Account</button>
          <button role="menuitem" data-toast="Signing out isn't in this prototype"><i data-lucide="log-out"></i>Sign out</button>
        </div>
      </div>
    </header>`;

  const portalNav = () => `
    <header class="mq-nav" data-nav>
      <a class="mq-wordmark" href="?variant=B&surface=portal&device=${device}&screen=home" data-home aria-label="SlipStream, Home"${screen === 'home' ? ' aria-current="page"' : ''}>Slip<span>Stream</span></a>
      <nav aria-label="Main"><ul class="mq-navlist">
        <li><button class="mq-navlink${screen === 'home' ? ' is-on' : ''}" data-home>Home</button></li>
        <li><button class="mq-navlink" data-toast="Movies isn't in this prototype">Movies</button></li>
        <li><button class="mq-navlink" data-toast="Series isn't in this prototype">Series</button></li>
        <li><button class="mq-navlink" data-toast="Your requests page isn't in this prototype">Your requests</button></li>
      </ul></nav>
      ${screen === 'work' ? '<button class="mq-searchbox" data-search aria-haspopup="dialog"><i data-lucide="search"></i><span>Search movies and series</span><kbd>/</kbd></button>' : '<span class="mq-nav-spacer"></span>'}
      ${bell()}
      <div class="mq-menu-wrap">
        <button class="mq-avatar" aria-label="Account, ${data.portal.member.name}" aria-expanded="false" aria-controls="pop-acct" data-pop="acct">${data.portal.member.initials}</button>
        <div class="mq-pop mq-pop-menu mq-pop-right" id="pop-acct" role="menu" hidden>
          <p class="mq-pop-h">Maya Chen<span class="mq-dim">${data.portal.member.email}</span></p>
          <button role="menuitem" data-toast="Account isn't in this prototype"><i data-lucide="user"></i>Account</button>
          <button role="menuitem" data-toast="Signing out isn't in this prototype"><i data-lucide="log-out"></i>Sign out</button>
        </div>
      </div>
    </header>`;

  function bell() {
    return `<div class="mq-menu-wrap">
        <button class="mq-bell" aria-label="Inbox, ${data.portal.inbox.length} new" aria-expanded="false" aria-controls="pop-inbox" data-pop="inbox"><i data-lucide="bell"></i><span class="mq-bell-n mq-num">${data.portal.inbox.length}</span></button>
        <div class="mq-pop mq-pop-inbox mq-pop-right" id="pop-inbox" hidden>
          <p class="mq-pop-h">Inbox</p>
          <ul>${data.portal.inbox.map((n) => `<li><i data-lucide="circle-check"></i><span>${n.text}<span class="mq-dim">${ago(n.at)}</span></span></li>`).join('')}</ul>
        </div>
      </div>`;
  }

  const tabs = surface === 'admin'
    ? [['house', 'Home', 'home'], ['library', 'Library', 'toast:Library'], ['activity', 'Activity', 'toast:Activity'], ['search', 'Search', 'search'], ['menu', 'More', 'sheet:sheet-more']]
    : [['house', 'Home', 'home'], ['search', 'Search', 'search'], ['bookmark', 'Requests', 'toast:Your requests page'], ['circle-user', 'Account', 'toast:Account']];
  const tabBar = () => `
    <nav class="mq-tabs" aria-label="Tabs">${tabs.map(([ic, label, act], i) => {
      const on = i === 0;
      const attr = act === 'home' ? 'data-home' : act === 'search' ? 'data-search aria-haspopup="dialog"' : act.startsWith('sheet:') ? `data-sheet="${act.slice(6)}" aria-haspopup="dialog"` : `data-toast="${act.slice(6)} isn't in this prototype"`;
      return `<button class="mq-tab${on ? ' is-on' : ''}" ${attr}${on ? ' aria-current="page"' : ''}><i data-lucide="${ic}"></i><span>${label}</span>${label === 'Activity' ? `<span class="mq-tab-badge mq-num">${running.length}</span>` : ''}</button>`;
    }).join('')}</nav>`;

  const phoneHeader = () => {
    if (screen === 'work') return '';
    if (surface === 'admin') return `
      <header class="mq-ph-head">
        <div class="mq-ph-title"><h1 class="mq-ph-h1">Tonight</h1><button class="mq-avatar" aria-label="Account, ${data.admin.name}" data-toast="Account isn't in this prototype">${data.admin.initials}</button></div>
      </header>`;
    return `
      <header class="mq-ph-head">
        <div class="mq-ph-title"><span class="mq-wordmark">Slip<span>Stream</span></span>${bell()}</div>
      </header>`;
  };

  const phoneAdminHome = () => `
    ${phoneHeader()}
    <section class="mq-hero mq-hero-card" aria-labelledby="hero-t">
      <div class="mq-hero-art" aria-hidden="true">${art(dune.art.backdrop, dune.title)}</div>
      <div class="mq-hero-copy">
        <p class="mq-hero-kind"><span class="mq-live-dot" aria-hidden="true"></span>In flight now · Conversion</p>
        ${logo(dune, 'h2', 'mq-hero-logo').replace('<h2 ', '<h2 id="hero-t" ')}
        ${heroSentence()}
        <button class="mq-btn mq-btn-solid mq-btn-block" data-go>Open ${dune.title}</button>
      </div>
      ${line(duneJob.id, 'mq-line-hero')}
    </section>
    <section class="mq-needs" aria-labelledby="needs-t">
      <header class="mq-shelf-h"><h2 id="needs-t">Needs you</h2><span class="mq-count">${data.health.length}</span></header>
      <div class="mq-needs-grid">${failureCard()}${warnCard()}</div>
    </section>
    ${statusSill()}
    ${shelf('flight', 'In flight', running.length, running.map(inflightCard).join(''))}
    ${shelf('next', 'Up next', data.upNext.length, data.upNext.map(upNextCard).join(''), { cls: 'mq-shelf-land' })}
    ${shelf('req', 'Requests from Members', `${pending.length} pending`, data.requests.map(requestCard).join(''))}
    ${shelf('recent', 'Recently added', null, data.recentlyAdded.map((id) => posterCard(id)).join(''))}`;

  const phonePortalHome = () => `
    ${phoneHeader()}
    <section class="mq-search-hero" aria-labelledby="ps-t">
      <div class="mq-wall-bg" aria-hidden="true">${wallBg.slice(0, 12).map((id) => `<img src="${W[id].art.poster}" alt="" data-fb-hide>`).join('')}</div>
      <div class="mq-search-copy">
        <h1 id="ps-t">What are we watching, ${data.portal.member.name}?</h1>
        <button class="mq-bigsearch mq-bigsearch-btn" data-search aria-haspopup="dialog"><i data-lucide="search"></i><span>Search movies and series</span></button>
      </div>
    </section>
    ${shelf('mine', 'Your requests', data.portal.mine.length, data.portal.mine.map(mineCard).join(''), { cls: 'mq-shelf-mine' })}
    ${shelf('new', 'New in the library', null, data.portal.newInLibrary.map((id) => posterCard(id, `${meta(W[id])} · Ready`)).join(''))}
    ${shelf('pop', 'Popular this week', null, data.portal.popular.map(wallCard).join(''), { cls: 'mq-shelf-wall' })}`;

  // ---------- sheets (phone) and palette ----------

  const sheet = (id, title, body) => `
    <div class="mq-sheet-wrap" id="${id}" hidden>
      <div class="mq-sheet-scrim" data-close></div>
      <div class="mq-sheet" role="dialog" aria-modal="true" aria-labelledby="${id}-t">
        <span class="mq-grabber" aria-hidden="true"></span>
        <h2 class="mq-sheet-t" id="${id}-t">${title}</h2>
        ${body}
      </div>
    </div>`;
  const sheetBtn = (icon, label, toast, cls = '') => `<button class="mq-sheet-item ${cls}" data-toast="${toast}" data-close><i data-lucide="${icon}"></i>${label}</button>`;

  const sheets = !phone ? '' : [
    surface === 'admin' ? sheet('sheet-more', 'More', `<div class="mq-sheet-group">
        ${sheetBtn('calendar', 'Calendar', "Calendar isn't in this prototype")}
        ${sheetBtn('star', 'Wanted', "Wanted isn't in this prototype")}
        ${sheetBtn('inbox', `Requests · ${pending.length} pending`, "Requests isn't in this prototype")}
        ${sheetBtn('history', 'History', "History isn't in this prototype")}
        ${sheetBtn('folder-input', 'Manual import', "Manual import isn't in this prototype")}
        <button class="mq-sheet-item" data-pop-open="nodes"><i data-lucide="server"></i>Nodes · relay-fra degraded</button>
        ${sheetBtn('settings', 'Settings', "Settings isn't in this prototype")}
        ${sheetBtn('activity', 'System', "System isn't in this prototype")}
      </div><button class="mq-sheet-cancel" data-close>Cancel</button>`) : '',
    surface === 'admin' ? sheet('sheet-nodes', nodeLine, `${nodesDetail()}<button class="mq-sheet-cancel" data-close>Done</button>`) : '',
    surface === 'admin' ? sheet('sheet-fail', 'The Batman · failed at Import', `<div class="mq-sheet-group">
        ${sheetBtn('list', 'Manual search', "Manual search isn't in this prototype")}
        ${sheetBtn('ban', 'View Blocklist entry', "The Blocklist isn't in this prototype")}
        ${sheetBtn('clapperboard', 'Open The Batman', '', '').replace('data-toast=""', 'data-go')}
        ${sheetBtn('x', 'Dismiss', 'Dismissed in a real build; kept here for review')}
      </div><button class="mq-sheet-cancel" data-close>Cancel</button>`) : '',
    surface === 'admin' ? sheet('sheet-work', 'Dune: Part Two', `<div class="mq-sheet-group">
        ${sheetBtn('list', 'Manual search', "Manual search isn't in this prototype")}
        ${sheetBtn('sliders-horizontal', 'Edit Profile', "Profiles aren't in this prototype")}
        ${sheetBtn('eye-off', 'Stop monitoring', 'Monitoring stays on in this prototype')}
        ${sheetBtn('refresh-cw', 'Refresh metadata', 'Metadata refresh queued')}
        ${sheetBtn('trash-2', 'Delete Work and files', "Deleting isn't in this prototype", 'mq-danger')}
      </div><button class="mq-sheet-cancel" data-close>Cancel</button>`) : '',
    surface === 'portal' ? sheet('sheet-request', 'Request this?', `<p class="mq-sheet-p" data-request-copy></p><div class="mq-sheet-group">
        <button class="mq-sheet-item mq-sheet-primary" data-request-confirm><i data-lucide="plus"></i>Request it</button></div><button class="mq-sheet-cancel" data-close>Cancel</button>`) : '',
  ].join('');

  const palette = `
    <div class="mq-palette-wrap" id="palette" hidden>
      <div class="mq-sheet-scrim" data-close></div>
      <div class="mq-palette" role="dialog" aria-modal="true" aria-label="Search">
        <div class="mq-palette-in"><i data-lucide="search"></i><input type="search" placeholder="${surface === 'admin' ? 'Search Works, Nodes and releases' : 'Search movies and series'}" aria-label="Search" autocomplete="off" data-palette-input><button class="mq-btn mq-btn-sm mq-btn-ghost" data-close>${phone ? 'Cancel' : 'Esc'}</button></div>
        <ul class="mq-palette-list" role="listbox" aria-label="Results" data-palette-list></ul>
      </div>
    </div>`;

  // ---------- assemble ----------

  let body;
  if (surface === 'admin') body = screen === 'work' ? adminWork() : phone ? phoneAdminHome() : adminHome();
  else body = screen === 'work' ? portalWork() : phone ? phonePortalHome() : portalHome();

  root.innerHTML = `
    <div class="mq-ambient" aria-hidden="true"><div class="mq-amb" data-amb="0"></div><div class="mq-amb" data-amb="1"></div><div class="mq-amb-scrim"></div></div>
    ${phone ? '' : surface === 'admin' ? adminNav() : portalNav()}
    <main class="mq-main mq-screen-${screen}">${body}</main>
    ${phone ? tabBar() : ''}
    ${sheets}
    ${palette}
    <div class="mq-toast" role="status" aria-live="polite"></div>`;

  // ---------- behavior ----------

  root.addEventListener('error', (e) => {
    const img = e.target;
    if (!(img instanceof HTMLImageElement)) return;
    if (img.dataset.fbLogo) { img.replaceWith(Object.assign(document.createElement('span'), { className: 'mq-logo-type', textContent: img.dataset.fbLogo })); return; }
    if (img.hasAttribute('data-fb-hide')) { img.style.visibility = 'hidden'; return; }
    if (img.dataset.fb) { img.replaceWith(Object.assign(document.createElement('span'), { className: 'mq-fallback', textContent: img.dataset.fb })); }
  }, true);

  const toastEl = root.querySelector('.mq-toast');
  let toastT;
  const toast = (msg) => {
    toastEl.textContent = msg;
    toastEl.classList.add('is-on');
    clearTimeout(toastT);
    toastT = setTimeout(() => toastEl.classList.remove('is-on'), 2600);
  };

  const closePops = (except) => root.querySelectorAll('[data-pop]').forEach((b) => {
    if (b === except) return;
    b.setAttribute('aria-expanded', 'false');
    const p = root.querySelector(`#${b.getAttribute('aria-controls')}`);
    if (p) p.hidden = true;
  });

  let lastFocus;
  const openLayer = (el) => {
    lastFocus = document.activeElement;
    el.hidden = false;
    requestAnimationFrame(() => el.classList.add('is-open'));
    const f = el.querySelector('input, button:not([data-close].mq-sheet-scrim), .mq-sheet-item, .mq-sheet-cancel');
    f && f.focus({ preventScroll: true });
  };
  const closeLayers = () => root.querySelectorAll('.mq-sheet-wrap:not([hidden]), .mq-palette-wrap:not([hidden])').forEach((el) => {
    el.classList.remove('is-open');
    setTimeout(() => { el.hidden = true; }, reduced ? 0 : 220);
    lastFocus && lastFocus.focus && lastFocus.focus({ preventScroll: true });
  });

  const works = Object.values(W);
  const paletteList = root.querySelector('[data-palette-list]');
  const renderResults = (q) => {
    const hits = works.filter((w) => w.title.toLowerCase().includes(q.toLowerCase())).slice(0, 7);
    paletteList.innerHTML = hits.length ? hits.map((w, i) => `
      <li role="option" aria-selected="${i === 0}"><button class="mq-hit" data-go data-work="${w.id}">${art(w.art.poster, w.title)}<span><strong>${w.title}</strong><span class="mq-dim">${meta(w)}${surface === 'portal' && libraryState[w.id] ? ` · ${ribbon[libraryState[w.id]][0]}` : ''}</span></span><i data-lucide="corner-down-left"></i></button></li>`).join('')
      : `<li class="mq-hit-empty">Nothing in the library matches "${q}".${surface === 'portal' ? ' Try another spelling, or request it from a Work page.' : ' Try Wanted to search indexers.'}</li>`;
    icons();
  };
  const openSearch = (q = '') => {
    closePops();
    const p = root.querySelector('#palette');
    const input = p.querySelector('input');
    input.value = q;
    renderResults(q);
    openLayer(p);
    input.focus();
  };
  root.querySelector('[data-palette-input]').addEventListener('input', (e) => renderResults(e.target.value));

  let pendingRequest;
  root.addEventListener('click', (e) => {
    const t = e.target.closest('button, a, [data-close]');
    if (!t || !root.contains(t)) { if (!e.target.closest('.mq-pop')) closePops(); return; }
    if (t.matches('a[data-home], [data-home]')) { e.preventDefault(); if (screen !== 'home') go('home'); else toast("You're on Home"); return; }
    if (t.hasAttribute('data-go')) { e.preventDefault(); go('work'); return; }
    if (t.hasAttribute('data-pop')) {
      const p = root.querySelector(`#${t.getAttribute('aria-controls')}`);
      const open = t.getAttribute('aria-expanded') !== 'true';
      closePops(t);
      t.setAttribute('aria-expanded', String(open));
      p.hidden = !open;
      if (open) { const first = p.querySelector('button'); first && first.focus({ preventScroll: true }); }
      return;
    }
    if (t.hasAttribute('data-pop-open')) {
      closeLayers();
      if (phone) { setTimeout(() => openLayer(root.querySelector('#sheet-nodes')), reduced ? 0 : 230); return; }
      const b = root.querySelector('[data-pop="nodes"]');
      closePops(b);
      b.setAttribute('aria-expanded', 'true');
      root.querySelector('#pop-nodes').hidden = false;
      b.scrollIntoView({ block: 'nearest' });
      return;
    }
    if (t.hasAttribute('data-search')) { openSearch(); return; }
    if (t.hasAttribute('data-sheet')) { openLayer(root.querySelector(`#${t.dataset.sheet}`)); return; }
    if (t.hasAttribute('data-request')) {
      const w = W[t.dataset.request];
      pendingRequest = t;
      if (phone) {
        root.querySelector('[data-request-copy]').textContent = `${w.title} isn't in the library yet. Ask for it and we'll tell you when it's ready to watch.`;
        openLayer(root.querySelector('#sheet-request'));
      } else markRequested(t, w);
      return;
    }
    if (t.hasAttribute('data-request-confirm')) { closeLayers(); markRequested(pendingRequest, W[pendingRequest.dataset.request]); return; }
    if (t.hasAttribute('data-approve') || t.hasAttribute('data-decline')) {
      const ok = t.hasAttribute('data-approve');
      const r = data.requests.find((x) => x.id === (t.dataset.approve || t.dataset.decline));
      const wrap = t.closest('.mq-req-act');
      wrap.outerHTML = `<p class="mq-cap-m"><i data-lucide="${ok ? 'circle-check' : 'circle-x'}"></i>${ok ? 'Approved · searching' : 'Declined'}</p>`;
      icons();
      toast(`${ok ? 'Approved' : 'Declined'} ${member(r.by).name.split(' ')[0]}'s request for ${W[r.work].title}`);
      return;
    }
    if (t.hasAttribute('data-monitor')) {
      const on = t.getAttribute('aria-pressed') !== 'true';
      t.setAttribute('aria-pressed', String(on));
      t.querySelector('span').textContent = on ? 'Monitored' : 'Not monitored';
      toast(on ? 'Monitoring Dune: Part Two' : 'Stopped monitoring Dune: Part Two');
      return;
    }
    if (t.dataset.toast !== undefined && t.dataset.toast !== '') { closePops(); toast(t.dataset.toast); }
    if (t.hasAttribute('data-close')) { closeLayers(); }
  });

  function markRequested(btn, w) {
    const s = document.createElement('span');
    s.className = 'mq-reqc requested';
    s.innerHTML = '<i data-lucide="hourglass"></i>Requested';
    btn.replaceWith(s);
    icons();
    toast(`Requested ${w.title}. We'll tell you when it's ready.`);
  }

  const form = root.querySelector('[data-search-form]');
  form && form.addEventListener('submit', (e) => { e.preventDefault(); openSearch(form.querySelector('input').value); });

  root.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') { closePops(); closeLayers(); }
    const pal = root.querySelector('#palette');
    if (!pal.hidden && e.key === 'Enter' && e.target.matches('[data-palette-input]')) {
      const first = pal.querySelector('.mq-hit');
      if (first) go('work');
    }
  });
  addEventListener('keydown', (e) => {
    const typing = e.target.closest && e.target.closest('input, textarea, [contenteditable]');
    if (!typing && (e.key === '/' || ((e.metaKey || e.ctrlKey) && e.key === 'k'))) { e.preventDefault(); openSearch(); }
  });

  const nav = root.querySelector('[data-nav]');
  if (nav) {
    const onScroll = () => nav.classList.toggle('is-solid', scrollY > 24);
    addEventListener('scroll', onScroll, { passive: true });
    onScroll();
  }

  const phbar = root.querySelector('[data-phbar]');
  if (phbar) {
    const onPhScroll = () => phbar.classList.toggle('is-solid', root.scrollTop > 220);
    root.addEventListener('scroll', onPhScroll, { passive: true });
    onPhScroll();
  }

  onTick(() => {
    const ph = root.querySelectorAll('[data-eta-phrase]');
    if (duneJob.etaMin <= 0) ph.forEach((el) => { el.textContent = 'finishing up'; });
  });

  // ---------- ambient: the house takes its color from the Work in focus ----------

  const ambLayers = [...root.querySelectorAll('.mq-amb')];
  let ambOn = 0;
  let ambWork = null;
  const tintCache = new Map();
  const sampleTint = (src) => {
    if (tintCache.has(src)) return tintCache.get(src);
    const p = new Promise((res) => {
      const im = new Image();
      im.onload = () => {
        try {
          const c = document.createElement('canvas');
          c.width = 24; c.height = 36;
          const g = c.getContext('2d', { willReadFrequently: true });
          g.drawImage(im, 0, 0, 24, 36);
          const d = g.getImageData(0, 0, 24, 36).data;
          let x = 0, y = 0, ws = 0, ss = 0;
          for (let i = 0; i < d.length; i += 4) {
            const r = d[i] / 255, gg = d[i + 1] / 255, b = d[i + 2] / 255;
            const mx = Math.max(r, gg, b), mn = Math.min(r, gg, b), l = (mx + mn) / 2, dd = mx - mn;
            if (dd < 0.04) continue;
            const s = dd / (1 - Math.abs(2 * l - 1));
            let h = mx === r ? ((gg - b) / dd) % 6 : mx === gg ? (b - r) / dd + 2 : (r - gg) / dd + 4;
            h = (h * 60 + 360) % 360;
            const w = s * s * (1 - Math.abs(l - 0.5) * 1.6);
            if (w <= 0) continue;
            x += Math.cos((h * Math.PI) / 180) * w; y += Math.sin((h * Math.PI) / 180) * w; ws += w; ss += s * w;
          }
          if (!ws) return res({ h: 220, s: 10 });
          res({ h: ((Math.atan2(y, x) * 180) / Math.PI + 360) % 360, s: Math.round(Math.min(62, Math.max(18, (ss / ws) * 70))) });
        } catch { res(null); }
      };
      im.onerror = () => res(null);
      im.src = src;
    });
    tintCache.set(src, p);
    return p;
  };

  const setAmbient = (id) => {
    if (ambWork === id) return;
    ambWork = id;
    const w = W[id];
    const next = ambLayers[1 - ambOn];
    const pre = new Image();
    pre.onload = () => {
      if (ambWork !== id) return;
      next.style.backgroundImage = `url('${w.art.backdrop}')`;
      next.classList.add('is-on');
      ambLayers[ambOn].classList.remove('is-on');
      ambOn = 1 - ambOn;
    };
    pre.src = w.art.backdrop;
    sampleTint(w.art.poster).then((t) => {
      if (!t || ambWork !== id) return;
      root.style.setProperty('--th', t.h.toFixed(1));
      root.style.setProperty('--ts', `${t.s}%`);
    });
  };

  const homeWork = surface === 'portal' && screen === 'home' ? 'interstellar' : 'dune-2';
  setAmbient(homeWork);

  if (!reduced && !phone) {
    let leaveT;
    const peek = (el) => {
      clearTimeout(leaveT);
      root.querySelectorAll('.mq-pt.is-peek').forEach((p) => p.classList.remove('is-peek'));
      el.classList.add('is-peek');
      root.classList.add('is-peeking');
      setAmbient(el.dataset.work);
    };
    const unpeek = () => {
      clearTimeout(leaveT);
      leaveT = setTimeout(() => {
        root.querySelectorAll('.mq-pt.is-peek').forEach((p) => p.classList.remove('is-peek'));
        root.classList.remove('is-peeking');
        setAmbient(homeWork);
      }, 160);
    };
    root.querySelectorAll('.mq-row .mq-pt[data-work]').forEach((el) => {
      el.addEventListener('mouseenter', () => peek(el));
      el.addEventListener('mouseleave', unpeek);
      el.addEventListener('focus', () => { if (el.matches(':focus-visible')) peek(el); });
      el.addEventListener('blur', unpeek);
    });
  }

  icons();
}
