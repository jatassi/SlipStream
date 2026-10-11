// PROTOTYPE. Node detail sections. Every variant opens the same detail its own way: pushed page (A), inspector (B), sheet (C).

import { ROLES, ROLE_NAMES, CONTROLLER_ONLY, CONTROLLER_VERSION, IMAGE, groupOf, probe, trialsFor, torrents, works } from './data.js';
import { esc, I, dot, hm, minsFromNow, plural, statePill, kindChip, dialText, heartbeatText, workRow, itemLabel } from './shell.js';
import { jobsOn, roleOfJob, controllerOf } from './model.js';

export function handoverBlock(n, nodes) {
  if (n.kind === 'Controller') return 'It is the Controller.';
  if (n.state === 'offline') return `Offline since ${hm(n.lastSeen)}.`;
  if (n.version !== controllerOf(nodes).version) return `On ${n.version}. A Handover needs the Controller's version, ${CONTROLLER_VERSION}.`;
  return null;
}

export function header(app, n, actions) {
  const ctl = controllerOf(app.S.nodes);
  return `<header class="nd-head">
    <div class="nd-title"><h1 class="nd-name mono">${esc(n.name)}</h1>${statePill(n)}${kindChip(n)}</div>
    <p class="nd-sub t2">${esc(n.hw)}<span class="s-sep">·</span>${esc(n.where)}<span class="s-sep">·</span>${esc(n.os)} ${esc(n.arch)}${n.install === 'docker' ? '<span class="s-sep">·</span>Docker' : ''}<span class="s-sep">·</span><span class="num">${esc(n.version)}</span></p>
    ${n.kind === 'Worker' ? `<p class="nd-sub t3">${esc(dialText(n, ctl))}${heartbeatText(n) ? `<span class="s-sep">·</span>${esc(heartbeatText(n))}` : ''}</p>` : ''}
    ${actions ? `<div class="nd-act">${actions}</div>` : ''}
  </header>`;
}

export function defaultActions(app, n, editBtn) {
  const nodes = app.S.nodes;
  if (n.kind === 'Controller') {
    return `${editBtn}<button class="btn" data-act="pair">${I('plus')}Pair a Node</button>${nodes.length > 1 ? `<button class="btn" data-act="handover">${I('crown')}Hand over Controller</button>` : ''}`;
  }
  const block = handoverBlock(n, nodes);
  return `${editBtn}<button class="btn" data-act="test-conn" data-node="${n.id}">${I('radio-tower')}Test connection</button>
    <button class="btn" data-act="handover" data-node="${n.id}" ${block ? `aria-disabled="true" title="${esc(block)}"` : ''}>${I('crown')}Hand over Controller here</button>
    <button class="btn ghost danger-t" data-act="remove" data-node="${n.id}">${I('trash-2')}Remove</button>`;
}

export function stateAlert(app, n) {
  const ctl = controllerOf(app.S.nodes);
  if (n.state === 'offline') {
    const left = minsFromNow(n.requeueAt);
    const js = jobsOn(n).filter((j) => j.status === 'running');
    const portable = js.filter((j) => ROLES[roleOfJob(j)]?.portable);
    const tied = js.filter((j) => !ROLES[roleOfJob(j)]?.portable);
    const otherSearch = app.S.nodes.some((m) => m !== n && m.roles.Searching?.state === 'active');
    return `<section class="nd-alert al-fail" aria-label="Offline">
      <div class="al-h"><span class="chip-fail">${I('unplug')}Offline</span><p><b>${esc(n.name)} missed its heartbeats.</b> <span class="fail-dim">Last seen ${hm(n.lastSeen)}, ${-minsFromNow(n.lastSeen)} min ago.</span></p></div>
      <div class="al-grid">
        <div class="al-col">
          <h3>Requeues at ${hm(n.requeueAt)} <span class="fail-dim num">in ${left} min</span></h3>
          <p class="fail-dim">After a 15 min grace period, portable jobs move to another Node with the same Role.</p>
          <ul class="wl">${portable.map((j) => workRow(j, { compact: true, held: true, note: otherSearch ? 'Requeues to another Searching Node' : 'No other Node holds Searching. It waits until one does.', noteCls: 'fail-dim' })).join('')}</ul>
          ${otherSearch ? '' : `<button class="btn sm" data-act="add-role" data-node="${ctl.id}" data-role="Searching">${I('plus')}Add Searching to ${esc(ctl.name)}</button>`}
        </div>
        <div class="al-col">
          <h3>Waits for ${esc(n.name)}</h3>
          <p class="fail-dim">Downloads, seeding and Transfers are tied to this Node. They resume when it reconnects.</p>
          <ul class="wl">${tied.map((j) => workRow(j, { compact: true, held: true, note: j.stage === 'Transfer' ? 'Paused. Resumes from verified chunks.' : 'Waiting', noteCls: 'fail-dim' })).join('')}</ul>
          ${n.seeding ? `<p class="al-seed">${I('arrow-up')}<span><b class="num">${n.seeding.count}</b> torrents seeding to goals, waiting</span></p>` : ''}
        </div>
      </div>
      <div class="al-foot"><p class="fail-dim">Gone for good? Force marks its tied work lost.</p><button class="btn sm" data-act="force" data-node="${n.id}">${I('triangle-alert')}Force…</button><button class="btn sm" data-act="remove" data-node="${n.id}">Remove ${esc(n.name)}</button></div>
    </section>`;
  }
  if (n.state === 'updating') {
    return `<section class="nd-alert al-warn" aria-label="Updating">
      <div class="al-h"><span class="chip-warn">${I('refresh-cw')}Updating</span><p><b>${esc(n.name)} runs ${esc(n.version)}. ${esc(ctl.name)} runs ${CONTROLLER_VERSION}.</b> <span class="warn-dim">Until it updates it finishes in-flight work and starts nothing new.</span></p></div>
      <div class="al-up"><div class="bar al-bar" style="--p:${n.update.progress}" role="progressbar" aria-valuenow="${Math.round(n.update.progress * 100)}" aria-label="Update progress"><span class="fill"></span></div><span class="num">${Math.round(n.update.progress * 100)}%</span></div>
      <p class="warn-dim">${esc(n.update.phase)}. Then it checks the signature, swaps and restarts.</p>
    </section>`;
  }
  if (n.state === 'behind') {
    return `<section class="nd-alert al-warn" aria-label="Older version">
      <div class="al-h"><span class="chip-warn">${I('container')}Pull ${CONTROLLER_VERSION}</span><p><b>${esc(n.name)} runs ${esc(n.version)} in Docker.</b> <span class="warn-dim">SlipStream can't swap a container image. Pull the new tag and recreate the container. Until then it finishes in-flight work and starts nothing new.</span></p></div>
      <div class="codeline"><code class="mono">docker pull ${IMAGE}:${CONTROLLER_VERSION}</code><button class="btn sm" data-act="copy" data-copy="docker pull ${IMAGE}:${CONTROLLER_VERSION}">${I('copy')}Copy</button></div>
    </section>`;
  }
  if (n.state === 'degraded') {
    return `<section class="nd-alert al-warn al-line" aria-label="Degraded"><span class="chip-warn">${I('triangle-alert')}Degraded</span><p><b>${esc(n.warn)}.</b> <span class="warn-dim">${esc(n.warnBody)}</span></p><button class="btn warn sm" data-act="test-conn" data-node="${n.id}">Test connection</button></section>`;
  }
  return '';
}

export function rolesSection(app, n, editBtn = '') {
  const held = ROLE_NAMES.filter((r) => n.roles[r]);
  const blocked = ROLE_NAMES.filter((r) => !n.roles[r] && probe(n.id, r).v === 'blocked');
  const rows = held.map((r) => {
    const st = n.roles[r], pr = probe(n.id, r);
    const state = st.state === 'draining'
      ? `<span class="rs rs-drain">${I('log-out')}Draining</span><span class="t2">${esc(st.left)}. No new work.</span>`
      : `<span class="rs rs-on">${dot(n.state === 'online' || n.state === 'degraded' ? 'draft' : 'idle')}Active</span>${st.handedBack ? '<span class="t3">Handed back. No other Node holds it, and it is always held.</span>' : n.single ? '<span class="t3">In process</span>' : ''}`;
    return `<li class="rrow"><span class="rrow-n"><b>${esc(r)}</b><span class="t3">${groupOf(r)}</span></span><span class="rrow-s">${state}</span>${pr.v === 'weak' ? `<span class="rrow-w warn-t">${dot('warn')}${esc(pr.note)}</span>` : ''}</li>`;
  }).join('');
  return `<section class="nd-sec" aria-labelledby="roles-${n.id}">
    <div class="sec-h"><h2 id="roles-${n.id}">Roles</h2><span class="t3 num">${held.length} of 10</span><span class="grow"></span>${editBtn}</div>
    <div class="nd-card">
      ${n.kind === 'Controller' ? `<p class="ctl-only">${I('lock')}<span><b>Always on the Controller:</b> ${CONTROLLER_ONLY}</span></p>` : ''}
      ${held.length ? `<ul class="rlist">${rows}</ul>` : '<p class="nd-empty t2">No Roles yet. This Node does nothing until you assign one.</p>'}
      ${blocked.length ? `<p class="rblocked t3">${I('lock')}<span><b class="t2">Can't hold</b> ${blocked.map((r) => `${esc(r)}: ${esc(probe(n.id, r).note.replace(/\.$/, ''))}`).join('. ')}.</span></p>` : ''}
    </div>
  </section>`;
}

export function jobsSection(app, n) {
  const js = jobsOn(n);
  const drainingRoles = ROLE_NAMES.filter((r) => n.roles[r]?.state === 'draining');
  const offline = n.state === 'offline';
  const note = (j) => (drainingRoles.includes(roleOfJob(j)) ? 'Finishing. No new work after it.' : j.node.includes('→') ? `Transfer ${j.node}` : '');
  const tor = n.seeding ? `<div class="tor-h"><span>${I('arrow-up')}<b class="num">${n.seeding.count}</b> torrents seeding · ratio <span class="num">${n.seeding.ratio}</span></span><span class="t3">Goals come from the Controller's intent. Torrents stay here unless you move them.</span></div>
    <ul class="tor">${torrents.map((t) => `<li><span class="ell">${esc(works[t.work].title)} <span class="t3">${esc(itemLabel(t.work, t.itemKey))}</span></span><span class="num t2">${t.ratio}</span><span class="t3">to ${esc(t.goal)}</span></li>`).join('')}</ul>` : '';
  if (offline) return '';
  return `<section class="nd-sec" aria-labelledby="jobs-${n.id}">
    <div class="sec-h"><h2 id="jobs-${n.id}">Jobs on this Node</h2><span class="t3 num">${plural(js.filter((j) => j.status === 'running').length, 'running', 'running')}</span></div>
    <div class="nd-card">${js.length ? `<ul class="wl">${js.map((j) => workRow(j, { note: note(j) })).join('')}</ul>` : '<p class="nd-empty t2">Nothing running here.</p>'}${tor}</div>
  </section>`;
}

const VERDICT = { ok: ['check', 'ok-t', 'Passed'], weak: ['triangle-alert', 'warn-t', 'Weak'], blocked: ['lock', 'fail-t', 'Blocked'] };
export function probesSection(n) {
  const t = trialsFor(n.id);
  return `<section class="nd-sec" aria-labelledby="pr-${n.id}">
    <div class="sec-h"><h2 id="pr-${n.id}">Health and probes</h2><span class="t3">Run at Pairing, rerun when hardware or drivers change</span><span class="grow"></span><button class="btn ghost sm" data-toast="Rerunning probes is not in this prototype">${I('refresh-cw')}Rerun</button></div>
    <div class="nd-card">
      <ul class="plist">${ROLE_NAMES.map((r) => { const p = probe(n.id, r), v = VERDICT[p.v]; return `<li><span class="${v[1]}" title="${v[2]}">${I(v[0])}</span><b>${esc(r)}</b><span class="${p.v === 'ok' ? 't3' : v[1]} ell">${esc(p.note)}</span></li>`; }).join('')}</ul>
      ${t ? `<div class="trials"><p class="trials-h"><b>Converting proved</b><span class="t3"> · ${esc(t.engine)} · trial runs cached until ffmpeg or the driver changes</span></p>
        <table class="ttab"><thead><tr><th>Codec</th><th>Depth</th><th>Dynamic range</th><th>Trial</th></tr></thead><tbody>${t.runs.map(([c, d, r, ok, s]) => `<tr><td class="mono">${c}</td><td class="mono">${d}</td><td>${esc(r)}</td><td class="${ok ? 'ok-t' : 'fail-t'}">${I(ok ? 'check' : 'x')}<span class="${ok ? 'num t2' : ''}">${esc(s)}</span></td></tr>`).join('')}</tbody></table></div>` : ''}
    </div>
  </section>`;
}

export function linkSection(app, n) {
  const ctl = controllerOf(app.S.nodes);
  const isCtl = n.kind === 'Controller';
  const workers = app.S.nodes.filter((m) => m.kind === 'Worker');
  return `<section class="nd-sec" aria-labelledby="ln-${n.id}">
    <div class="sec-h"><h2 id="ln-${n.id}">Node link</h2>${isCtl ? '' : `<span class="t3">Chosen at Pairing from what each side can reach</span>`}</div>
    <div class="nd-card"><dl class="kvl">
      ${isCtl ? `<dt>Links</dt><dd>${n.single ? 'None. Every Role runs in process.' : workers.map((w) => `<span class="lk">${esc(dialText(w, ctl))}</span>`).join('')}</dd>`
        : `<dt>Who dials</dt><dd><span class="dial"><span class="mono">${esc(ctl.name)}</span>${I(n.link.dial === 'worker' ? 'arrow-left' : 'arrow-right')}<span class="mono">${esc(n.name)}</span></span><span class="t3 dd-n">${esc(dialText(n, ctl))}. One channel carries calls both ways.</span></dd>
      <dt>Relay</dt><dd class="t2">${n.link.relay ? `Through ${esc(n.link.relay)}` : 'Not used'}</dd>
      <dt>Heartbeat</dt><dd>${n.state === 'offline' ? `<span class="fail-t">${esc(heartbeatText(n))}</span>` : `<span class="num">${n.link.heartbeat} s ago</span><span class="t3"> · <span class="num">${n.link.latency} ms</span></span>`}</dd>`}
      <dt>Address</dt><dd class="mono">${esc(n.addr)}</dd>
      <dt>Key</dt><dd><span class="fp mono">${esc(n.link.fingerprint)}</span><span class="t3 dd-n">${isCtl ? 'Workers pin this key.' : `Pinned at Pairing. Removing ${esc(n.name)} deletes it.`}</span></dd>
      <dt>Version</dt><dd><span class="num">${esc(n.version)}</span> <span class="${n.version === CONTROLLER_VERSION ? 't3' : 'warn-t'}">${isCtl ? '· every Worker runs this version' : n.version === CONTROLLER_VERSION ? '· same as the Controller' : `· the Controller runs ${CONTROLLER_VERSION}`}</span><span class="t3"> · ${n.install === 'docker' ? `Docker, ${IMAGE}` : 'native install, updates from the Controller'}</span></dd>
    </dl></div>
  </section>`;
}

export function drainNote(n) {
  if (!ROLE_NAMES.some((r) => n.roles[r]?.state === 'draining')) return '';
  return `<p class="nd-info">${I('info')}<span>A drain takes no new work and finishes what it holds. Seeding counts as unfinished until its goals are met. Torrents and library files stay put unless you start a move.</span><button class="btn ghost sm" data-toast="Moving torrents is not in this prototype">Move torrents…</button></p>`;
}

export function detailBody(app, n, editBtn) {
  return `${stateAlert(app, n)}${drainNote(n)}${rolesSection(app, n, editBtn)}${jobsSection(app, n)}${probesSection(n)}${linkSection(app, n)}`;
}
