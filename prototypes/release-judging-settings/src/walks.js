/* PROTOTYPE walkthroughs. Each step mutates settings or view; the page re-renders after every step. */
(() => {
  const work = (D, title) => D.works.find(w => w.title.toLowerCase() === title.toLowerCase());
  const open = (ctx, title, item, section = 'try') => { const w = work(ctx.D, title); if (w) { ctx.view.work = w.key; ctx.view.item = item; } ctx.view.section = section; };
  const r2160 = S => S.renditions.find(r => r.id === 'r2160');
  const moveTo = (r, measure, pos) => { const i = r.priorities.findIndex(p => p.measure === measure); if (i < 0) return; const [p] = r.priorities.splice(i, 1); r.priorities.splice(pos, 0, p); };
  window.WALKS = [
    {
      title: 'Three releases of one episode',
      text: 'House of the Dragon S03E05 has three real releases in your Downloads: a 1080p AMZN WEB-DL from playWEB, a 2160p AMZN WEB-DL from Draken02 with Atmos but no HDR tag, and a 2160p HMAX REPACK from FLUX tagged DV HDR. Watch which one each Rendition takes, why, and how one move in the priority list changes it.',
      steps: [
        { label: 'Reset and open the episode', run: c => open(c, 'House of the Dragon', 'S03E05') },
        { label: 'Move Audio above Picture in 2160p HDR', run: c => { moveTo(r2160(c.S), 'audio', 1); c.view.rendition = 'r2160'; } },
        { label: 'Let SDR into 2160p HDR', run: c => { const f = r2160(c.S).filters; f.picture = [...new Set(f.picture.concat('sdr'))]; } },
        { label: 'Put Picture back on top', run: c => moveTo(r2160(c.S), 'picture', 1) },
      ],
      done: 'The same three names, judged three ways. Each pick names the one priority that decided it, and the 1080p Rendition judged them separately the whole time.',
    },
    {
      title: 'Names that lie',
      text: 'Every release in your Downloads was probed. This shows which claims your files backed up and which they did not, and how that becomes per-group trust. Then it tightens the trust threshold and shows the picks that change.',
      steps: [
        { label: 'Open Trust and verification', run: c => { c.view.section = 'trust'; } },
        { label: 'Demand more: a claim counts only at 0.8 confidence', run: c => { c.S.trust.weakBelow = 0.8; c.view.section = 'knowledge'; } },
        { label: 'Make priors worth only 2 of your checks', run: c => { c.S.trust.priorStrength = 2; } },
      ],
      done: 'Raising the bar turns more claims into worst-case values. The replay table under Knowledge packs lists every pick that moved because of it.',
    },
    {
      title: 'One file, every Rendition',
      text: 'Your library keeps byte-identical copies of the same file, one per v1 slot. Under ADR 0002 one file fills every Rendition it meets. This walk shows what fills what, and what happens when the 1080p Rendition stops demanding SDR.',
      steps: [
        { label: 'Open Library impact', run: c => { c.view.section = 'impact'; } },
        { label: 'Allow HDR10 in the 1080p Rendition', run: c => { const r = c.S.renditions.find(r => r.id === 'r1080'); r.filters.picture = [...new Set(r.filters.picture.concat(['hdr10', 'dv+hdr10']))]; c.view.rendition = 'r1080'; c.view.section = 'renditions'; } },
        { label: 'Allow 2160p qualities in the 1080p Rendition', run: c => { const r = c.S.renditions.find(r => r.id === 'r1080'); r.qualities.push(['webdl-2160'], ['remux-2160']); } },
      ],
      done: 'Once 1080p accepts what 2160p holds, the editor flags that 2160p is fully contained by 1080p. Overlap is allowed, it is only a hint.',
    },
    {
      title: 'Turning on TRaSH',
      text: 'The converted TRaSH pack brings terms, hazards and group reputations, but no scores. Turn it on and see the replay against your library before accepting it.',
      steps: [
        { label: 'Open Knowledge packs', run: c => { c.view.section = 'knowledge'; } },
        { label: 'Turn on the TRaSH pack', run: c => { c.S.knowledge.trash = true; } },
        { label: 'Keep its terms, drop its hazards', run: c => { c.S.knowledge.trashKinds.hazard = false; } },
        { label: 'Drop its reputations too', run: c => { c.S.knowledge.trashKinds.reputation = false; } },
      ],
      done: 'Each change shows its effect as a list of changed picks before anything is grabbed.',
    },
    {
      title: 'Cold start for Horizon',
      text: 'Balanced patience needs history. With no Release Log, the only stand-in is when your downloads finished. Watch the forecast table say when it would wait.',
      steps: [
        { label: 'Open Timing for 2160p HDR', run: c => { c.view.section = 'timing'; c.view.rendition = 'r2160'; } },
        { label: 'Accept forecasts from 2 examples', run: c => { r2160(c.S).timing.minSamples = 2; } },
        { label: 'Shrink the step window to 12 hours', run: c => { r2160(c.S).timing.stepWindowHours = 12; } },
      ],
      done: 'With real Release Log data this table fills by service and network. Until then, Balanced behaves like Eager and says so.',
    },
  ];
})();
