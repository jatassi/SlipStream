// PROTOTYPE. Synthetic demonstration world for the UI redesign direction.
// Every title, person, Node and number here is invented.
(function () {
  const SS = (window.SS = window.SS || {});

  let seed = 20261006;
  const rnd = () => ((seed = (seed * 1664525 + 1013904223) >>> 0) / 4294967296);
  const pick = (a) => a[Math.floor(rnd() * a.length)];

  const NOW = new Date(2026, 9, 6, 21, 42, 0);
  const H = 3600e3, D = 24 * H, M = 60e3;
  const ago = (ms) => new Date(NOW.getTime() - ms);
  const at = (y, mo, d, h = 0, mi = 0) => new Date(y, mo - 1, d, h, mi);

  const firsts = ['Ines', 'Tomas', 'Ada', 'Rafael', 'Mireille', 'Oskar', 'Leilani', 'Bastien', 'Noor', 'Callum', 'Yuki', 'Saoirse', 'Matteo', 'Ingrid', 'Desmond', 'Amara', 'Linnea', 'Kofi', 'Elodie', 'Ruben', 'Hana', 'Silas', 'Pia', 'Joaquin', 'Wren', 'Anouk', 'Teo', 'Marisol', 'August', 'Freya'];
  const lasts = ['Vale', 'Okonkwo', 'Lindgren', 'Moreau', 'Achterberg', 'Sato', 'Ferreira', 'Kask', 'Delacroix', 'Abernathy', 'Novak', 'Halloran', 'Ishikawa', 'Brandt', 'Quill', 'Marlowe', 'Esposito', 'Varga', 'Nakashima', 'Orsini', 'Haverkamp', 'Lund', 'Castellanos', 'Reyes', 'Thorne', 'Kowalczyk', 'Mbeki', 'Aalto'];
  const person = () => `${pick(firsts)} ${pick(lasts)}`;

  const studios = ['Northlight', 'Tidewater Pictures', 'Halftone', 'Meridian Films', 'Lantern House', 'Greyfield', 'Paper Crane', 'Ninefold'];

  // Poster palettes: [ground top, ground bottom, accent, ink]
  const P = {
    dusk: ['#0d2a33', '#c8643a', '#f2b36b', '#f6ecdc'],
    arctic: ['#081624', '#4d7c9e', '#dbe8f1', '#ffffff'],
    desert: ['#2a1307', '#b9672f', '#f0c27a', '#fff3df'],
    noir: ['#070709', '#24242a', '#d9d4c7', '#efeae0'],
    crimson: ['#170406', '#6e0e16', '#e8473f', '#f6e7dc'],
    forest: ['#05110b', '#1d3a28', '#9ec27f', '#eaf2df'],
    violet: ['#0c0a1d', '#33265f', '#c6a6ff', '#f1ecff'],
    blush: ['#f0d6cb', '#d9a593', '#5b3a3a', '#2b1d1d'],
    gold: ['#110c05', '#53390f', '#e3b24f', '#fff4dc'],
    ocean: ['#03131d', '#0b4560', '#7fd1e0', '#e8fbff'],
    rose: ['#250c19', '#93395e', '#f2a7c3', '#fff0f6'],
    mint: ['#0a1d1a', '#2a7365', '#b7f0de', '#effff9'],
    bone: ['#ece3d2', '#cdb998', '#1d1a16', '#1d1a16'],
    ash: ['#111213', '#383b3f', '#b9bcc0', '#f2f2f2'],
    flamingo: ['#1b0e29', '#c94d72', '#ffb38a', '#fff1e8'],
    moss: ['#0a0e05', '#2b4411', '#cde86a', '#f4ffd8'],
    ember: ['#130704', '#982f16', '#ff8b4a', '#ffeede'],
    frost: ['#dbe6ee', '#a3bfd3', '#0e2233', '#0e2233'],
    cobalt: ['#050b1f', '#1a3a8f', '#8fb4ff', '#eef3ff'],
    sand: ['#e9dcc4', '#c9a877', '#3a2412', '#2a1a0e'],
  };

  const targets = [
    { id: 'uhd', name: '2160p HDR', short: '4K HDR', kind: 'downloaded', note: 'Dolby Vision or HDR10, remux or high-bitrate web' },
    { id: 'hd', name: '1080p', short: '1080p', kind: 'downloaded', note: 'Blu-ray or web, cutoff at Blu-ray' },
    { id: 'mobile', name: '720p Mobile', short: 'Mobile', kind: 'derived', from: 'hd', note: 'Derived by transcode from 1080p, 4 Mb/s' },
  ];
  const profiles = [
    { id: 'cinema', name: 'Cinema', targets: ['uhd', 'hd', 'mobile'] },
    { id: 'standard', name: 'Standard', targets: ['hd', 'mobile'] },
    { id: 'kids', name: 'Family', targets: ['hd', 'mobile'] },
  ];

  // [id, title, year, runtime, genres, cert, rating, status, palette, motif, font, tagline, overview, extra]
  const MOVIES = [
    ['halcyon-drift', 'Halcyon Drift', 2026, 134, ['Science Fiction', 'Drama'], 'PG-13', 8.1, 'downloading', 'cobalt', 'orbit', 'unbounded', 'Some distances are only crossed once.', 'A salvage pilot answers a distress call from a research station that went silent eleven years ago, and finds her own voice on the recording.'],
    ['paper-lanterns', 'Paper Lanterns', 2026, 118, ['Drama', 'Romance'], 'PG', 7.8, 'available', 'gold', 'lanterns', 'gloock', 'Every light was once a wish.', 'Two strangers keep missing each other at the same riverside festival, one summer after another.'],
    ['glass-orchard', 'Glass Orchard', 2026, 109, ['Drama'], 'R', 7.4, 'downloading', 'frost', 'forest', 'italiana', 'Nothing grows here by accident.', 'A botanist inherits a greenhouse that has not been opened since her grandmother vanished inside it.'],
    ['neon-psalm', 'Neon Psalm', 2025, 121, ['Thriller', 'Crime'], 'R', 7.2, 'available', 'flamingo', 'blinds', 'bebas', 'The city keeps its own confessions.', 'A night-shift radio host realises a caller is describing murders before they happen.'],
    ['low-tide-merrow', 'Low Tide at Merrow', 2024, 102, ['Mystery'], 'PG-13', 7.6, 'available', 'ocean', 'moon', 'gloock', 'The sea returns what it takes.', 'When the tide goes out farther than it has in a century, a fishing village finds a car on the sand with a passenger still buckled in.'],
    ['cartographers-daughter', "The Cartographer's Daughter", 2024, 141, ['Adventure', 'Drama'], 'PG-13', 7.9, 'available', 'desert', 'ridges', 'bodoni', 'Some places refuse to be mapped.', 'Following her father\'s last survey, a young surveyor crosses a desert that keeps redrawing itself.'],
    ['ember-season', 'Ember Season', 2026, 112, ['Romance', 'Drama'], 'PG-13', null, 'missing', 'ember', 'sun', 'italiana', 'Love, in the year everything burned.', 'A wildfire lookout and a smokejumper trade letters through one long, impossible summer.'],
    ['saltwater-kings', 'Saltwater Kings', 2025, 128, ['Crime', 'Drama'], 'R', 7.7, 'downloading', 'noir', 'waves', 'big-shoulders', 'Every harbour has a throne.', 'Three brothers inherit their father\'s fishing fleet and the debts that came with it.'],
    ['field-guide-leaving', 'A Field Guide to Leaving', 2024, 97, ['Comedy', 'Drama'], 'PG-13', 7.3, 'available', 'blush', 'field', 'gloock', 'Step one: tell no one.', 'A birdwatcher plans the perfect disappearance, then keeps getting invited to things.'],
    ['parallax', 'Parallax', 2026, 117, ['Science Fiction', 'Thriller'], 'PG-13', 7.5, 'upgradable', 'violet', 'rings', 'unbounded', 'Look again.', 'An astronomer discovers that one star shifts when nobody is watching it.'],
    ['ninth-wave', 'The Ninth Wave', 2025, 133, ['Drama', 'Disaster'], 'PG-13', 6.9, 'available', 'arctic', 'waves', 'bodoni', 'Count to nine.', 'The crew of a research vessel has six hours to cross a storm the forecasts said would not form.'],
    ['moth-lantern', 'Moth & Lantern', 2024, 94, ['Animation', 'Family'], 'G', 8.3, 'available', 'moss', 'lanterns', 'poiret', 'Follow the small lights.', 'A moth who is afraid of the dark leads a lost lantern home across a sleeping city.'],
    ['kilnworth', 'Kilnworth', 2026, 99, ['Horror'], 'R', 6.8, 'available', 'crimson', 'door', 'bebas', 'Do not stay past dark.', 'A potter rents a cottage beside an abandoned kiln that is still warm every morning.'],
    ['vesper-line', 'Vesper Line', 2025, 115, ['Thriller'], 'R', 7.4, 'upgradable', 'cobalt', 'streaks', 'big-shoulders', 'Last train. No stops.', 'A conductor on a night train discovers one of her passengers was declared dead this morning.'],
    ['heliotrope', 'Heliotrope', 2026, 106, ['Drama'], 'PG-13', 7.1, 'available', 'rose', 'sun', 'italiana', 'Turn toward the light.', 'A blind perfumer is asked to recreate a scent from a stranger\'s memory.'],
    ['the-understudy', 'The Understudy', 2024, 101, ['Comedy'], 'PG-13', 6.9, 'available', 'gold', 'type', 'limelight', 'The show must go on. Unfortunately.', 'An understudy who has never performed gets one night on stage, and the lead refuses to leave.'],
    ['coldwater-static', 'Coldwater Static', 2025, 96, ['Horror', 'Science Fiction'], 'R', 6.4, 'failed', 'ash', 'smoke', 'archivo-black', 'Something is broadcasting.', 'A radio engineer at an arctic relay station picks up a signal coming from under the ice.'],
    ['atlas-small-hours', 'Atlas of Small Hours', 2026, 124, ['Drama'], 'PG-13', 8.0, 'available', 'violet', 'stars', 'bodoni', 'Between three and five, the world is ours.', 'An insomniac cab driver and a night-shift nurse map the city after midnight.'],
    ['juniper-falls', 'Juniper Falls', 2024, 98, ['Family', 'Adventure'], 'PG', 7.2, 'available', 'forest', 'ridges', 'poiret', 'The summer the river spoke.', 'Three cousins search a mountain town for the treasure their grandfather swore was real.'],
    ['iron-lullaby', 'Iron Lullaby', 2025, 147, ['War', 'Drama'], 'R', 7.8, 'upgradable', 'ash', 'monolith', 'big-shoulders', 'Sleep, the guns are far away.', 'A field nurse smuggles children out of an occupied city in the back of a piano truck.'],
    ['velvet-hour', 'The Velvet Hour', 2026, 119, ['Music', 'Drama'], 'PG-13', 7.6, 'downloading', 'crimson', 'blinds', 'limelight', 'One last set.', 'A jazz singer reunites her old quintet for a single night in the club where it fell apart.'],
    ['driftless', 'Driftless', 2024, 126, ['Western'], 'R', 7.4, 'available', 'sand', 'ridges', 'bodoni', 'No glacier ever touched this land. Neither did law.', 'A surveyor and a deserter cross the unglaciated hills of 1857 Wisconsin with stolen maps.'],
    ['signal-noise', 'Signal & Noise', 2025, 88, ['Documentary'], 'PG', 7.9, 'available', 'mint', 'waves', 'archivo-black', 'Listen closer.', 'The engineers who keep the world\'s undersea cables alive, filmed over one year.'],
    ['bright-hollow', 'Bright Hollow', 2025, 103, ['Horror'], 'R', 6.6, 'available', 'moss', 'forest', 'bebas', 'The trees remember.', 'A family moves into a valley where the forest grows a little closer every night.'],
    ['lighthouse-protocol', 'Lighthouse Protocol', 2026, 113, ['Science Fiction', 'Thriller'], 'PG-13', 7.0, 'available', 'ocean', 'tower', 'unbounded', 'Keep the light on.', 'The last keeper of an automated lighthouse learns why it was never switched off.'],
    ['paloma-blue', 'Paloma Blue', 2024, 104, ['Romance', 'Comedy'], 'PG-13', 7.1, 'available', 'frost', 'split', 'italiana', 'Blue is a warm color.', 'A painter who only uses blue falls for the man who sells her the pigment.'],
    ['glass-republic', 'The Glass Republic', 2025, 131, ['Thriller', 'Drama'], 'R', 7.5, 'available', 'arctic', 'monolith', 'bodoni', 'Everyone can see. No one can look.', 'A transparency law makes every government meeting public, and a clerk finds the one room without windows.'],
    ['ferris-moon', 'Ferris & the Moon', 2026, 92, ['Animation', 'Family'], 'G', 8.0, 'available', 'cobalt', 'moon', 'poiret', 'He only wanted to borrow it.', 'A boy builds a ladder to the moon to return a pebble he once stole from it.'],
    ['sundown-mechanics', 'Sundown Mechanics', 2024, 110, ['Action'], 'R', 6.7, 'available', 'ember', 'road', 'archivo-black', 'Fix it or run.', 'A crew of getaway drivers takes one last job repairing the car that ruined them.'],
    ['overcast', 'Overcast', 2025, 95, ['Drama'], 'R', 7.3, 'available', 'ash', 'field', 'gloock', 'Weather permitting.', 'A meteorologist who cannot predict her own marriage forecasts a storm for her hometown.'],
    ['wolf-hour', 'Wolf Hour', 2026, 116, ['Thriller'], 'R', null, 'unreleased', 'noir', 'moon', 'bebas', 'The hour between dog and wolf.', 'A night guard at a shuttered zoo hears howling from an empty enclosure.', { release: at(2026, 10, 9) }],
    ['meridian', 'Meridian', 2027, null, ['Science Fiction'], 'PG-13', null, 'unreleased', 'gold', 'rings', 'unbounded', 'Noon, everywhere at once.', 'An expedition sails to the place where every time zone meets.', { release: at(2027, 3, 12) }],
    ['the-orchardist', 'The Orchardist', 2024, 121, ['Drama'], 'PG-13', 7.7, 'available', 'moss', 'field', 'gloock', 'Seasons do not wait.', 'An ageing apple grower teaches a runaway to graft the trees his son abandoned.'],
    ['northbound', 'Northbound, After Rain', 2025, 108, ['Drama', 'Road'], 'PG-13', 7.4, 'available', 'dusk', 'streaks', 'italiana', 'Keep driving until it clears.', 'Two estranged sisters drive their mother\'s ashes to the coast in a borrowed hearse.'],
    ['quiet-meridian', 'The Quiet Meridian', 2026, 127, ['Science Fiction', 'Drama'], 'PG-13', 7.9, 'upgradable', 'arctic', 'orbit', 'unbounded', 'Silence has a shape.', 'A deaf linguist is sent to the edge of the solar system to listen to something that does not make sound.'],
    ['sable-and-rye', 'Sable & Rye', 2025, 100, ['Comedy'], 'R', 6.8, 'available', 'sand', 'type', 'archivo-black', 'Bakers. Rivals. Neighbours.', 'Two bakeries share a wall, an oven flue, and a very old grudge.'],
    ['harbour-of-ghosts', 'Harbour of Ghosts', 2024, 112, ['Horror', 'Mystery'], 'R', 6.9, 'available', 'ocean', 'smoke', 'bebas', 'The fog knows your name.', 'A dock inspector logs a ship that arrives every night and never unloads.'],
    ['amber-protocol', 'Amber Protocol', 2026, 118, ['Action', 'Thriller'], 'PG-13', 6.5, 'missing', 'ember', 'road', 'big-shoulders', 'Lockdown in 60 minutes.', 'A courier is trapped inside the airport she was hired to rob.'],
    ['the-cartwright-sisters', 'The Cartwright Sisters', 2025, 129, ['Drama', 'Period'], 'PG', 7.8, 'available', 'bone', 'split', 'bodoni', 'Four sisters. One inheritance.', 'In 1912 Yorkshire, four sisters contest a will that leaves the family mill to a stranger.'],
    ['tin-crown', 'Tin Crown', 2024, 93, ['Animation', 'Adventure'], 'PG', 7.4, 'available', 'gold', 'monolith', 'poiret', 'A king made of leftovers.', 'A toy robot assembled from junkyard scraps is crowned king of the forgotten things.'],
    ['white-noise-hotel', 'White Noise Hotel', 2025, 105, ['Mystery', 'Comedy'], 'R', 7.0, 'available', 'mint', 'door', 'limelight', 'Check in. Keep quiet.', 'A night clerk at a hotel for insomniacs investigates a guest who finally fell asleep.'],
    ['last-orchard-light', 'The Last Orchard Light', 2026, 98, ['Drama'], 'PG', null, 'missing', 'dusk', 'sun', 'gloock', 'Harvest before the frost.', 'A family races the first frost to bring in the harvest after their father\'s stroke.'],
    ['pale-fire-bay', 'Pale Fire Bay', 2025, 111, ['Thriller'], 'R', 7.1, 'available', 'flamingo', 'waves', 'bebas', 'The bay is burning.', 'An oil-spill investigator becomes the only witness to a fire nobody reports.'],
    ['the-long-quiet', 'The Long Quiet', 2025, 136, ['Drama'], 'R', 8.2, 'available', 'noir', 'figure', 'italiana', 'Some silences are inherited.', 'A retired sound engineer returns to the village where he recorded his most famous silence.'],
    ['copper-sky', 'Copper Sky', 2024, 119, ['Western', 'Drama'], 'PG-13', 7.0, 'available', 'desert', 'sun', 'big-shoulders', 'Mine the sky.', 'A mining town strikes copper in the one place the elders said to leave alone.'],
    ['second-spring', 'Second Spring', 2026, 101, ['Romance'], 'PG', 7.2, 'missing', 'blush', 'lanterns', 'gloock', 'Late, and right on time.', 'Two widowed gardeners compete for the same allotment, then share it.'],
  ];

  // [id, title, startYear, endYear|null, network, genres, cert, rating, palette, motif, font, tagline, overview, seasons:[[episodes, airYear]]]
  const SERIES = [
    ['brightwater', 'Brightwater', 2024, null, 'Helix', ['Drama', 'Mystery'], 'TV-MA', 8.6, 'ocean', 'waves', 'gloock', 'The town keeps its secrets underwater.', 'When a reservoir is drained, a small town confronts what it buried to build it.', [[8, 2024], [8, 2025], [8, 2026]]],
    ['signal-keepers', 'The Signal Keepers', 2025, null, 'Northstar+', ['Science Fiction'], 'TV-14', 8.2, 'violet', 'orbit', 'unbounded', 'Someone has to listen.', 'A small team at a deep-space relay intercepts messages that should not exist yet.', [[10, 2025], [10, 2026]]],
    ['low-orbit', 'Low Orbit', 2023, 2026, 'Helix', ['Comedy'], 'TV-14', 7.9, 'cobalt', 'rings', 'archivo-black', 'Workplace comedy, 400 km up.', 'Life aboard the least glamorous space station in commercial service.', [[8, 2023], [8, 2024], [8, 2025], [6, 2026]]],
    ['harbor-lights', 'Harbor Lights', 2025, null, 'Tidewater', ['Crime', 'Drama'], 'TV-MA', 8.0, 'noir', 'lanterns', 'bebas', 'Every light is a witness.', 'A harbour patrol detective works the night shift in a port city that never quite sleeps.', [[10, 2025], [10, 2026]]],
    ['paper-moon-motel', 'Paper Moon Motel', 2024, null, 'Lumen', ['Comedy', 'Mystery'], 'TV-14', 7.7, 'flamingo', 'moon', 'limelight', 'Vacancy. Mystery included.', 'A desert motel\'s new owner solves a crime a week, mostly by accident.', [[8, 2024], [8, 2025]]],
    ['arcadia-station', 'Arcadia Station', 2026, null, 'Northstar+', ['Science Fiction', 'Drama'], 'TV-14', 8.1, 'mint', 'monolith', 'unbounded', 'The last stop before the dark.', 'A refuelling station at the edge of settled space becomes the front line of a quiet war.', [[9, 2026]]],
    ['saltmarsh', 'Saltmarsh', 2022, 2025, 'Helix', ['Drama'], 'TV-MA', 8.4, 'moss', 'field', 'gloock', 'The marsh keeps its own time.', 'Three generations of a family farm the edge of a rising sea.', [[6, 2022], [6, 2023], [6, 2024], [6, 2025]]],
    ['the-understory', 'The Understory', 2026, null, 'Verdant', ['Documentary', 'Nature'], 'TV-G', 8.8, 'forest', 'forest', 'italiana', 'Life below the canopy.', 'A year on the forest floor, filmed at the scale of the things that live there.', [[6, 2026]]],
    ['copperline', 'Copperline', 2025, null, 'Tidewater', ['Western', 'Drama'], 'TV-MA', 7.8, 'desert', 'ridges', 'big-shoulders', 'The rail stops here.', 'A railhead town in 1881 fights over the line that will make or bury it.', [[8, 2025], [8, 2026]]],
    ['night-shift-halden', 'Night Shift at Halden', 2024, null, 'Lumen', ['Drama', 'Medical'], 'TV-14', 7.6, 'arctic', 'door', 'bodoni', 'The lights never go out.', 'The overnight staff of a small hospital in a northern mining town.', [[10, 2024], [10, 2025], [4, 2026]]],
    ['fieldwork', 'Fieldwork', 2025, null, 'Verdant', ['Comedy'], 'TV-14', 7.5, 'sand', 'field', 'archivo-black', 'Science, eventually.', 'Graduate students stuck on a remote research station with one working kettle.', [[8, 2025]]],
    ['quiet-hours', 'Quiet Hours', 2026, null, 'Helix', ['Thriller'], 'TV-MA', null, 'ash', 'smoke', 'bebas', 'Lights out at ten.', 'A night watchman at a boarding school notices the students are rehearsing something.', [[8, 2026]]],
  ];

  const epWords = ['Low Water', 'The Drowned Bell', 'Survey', 'Ballast', 'Undertow', 'The Keeper', 'Slack Tide', 'Weir', 'First Light', 'Signal Fire', 'Cold Open', 'Interference', 'Carrier Wave', 'Static', 'Dead Air', 'Station Break', 'Perigee', 'Apogee', 'Burn', 'Drift', 'Small Hours', 'Black Box', 'Long Haul', 'Crosswind', 'Night Watch', 'Lantern', 'Floodgate', 'Echo', 'Waypoint', 'Meridian', 'Shoal', 'Deadhead', 'Night Ferry', 'Silt', 'Spillway', 'Breakwater', 'Low Ceiling', 'Clearance', 'Holding Pattern', 'Last Call'];

  const editionsFor = (id) => (id === 'iron-lullaby' ? ['Theatrical', "Director's Cut"] : id === 'the-long-quiet' ? ['Standard', 'Extended'] : ['Standard']);
  const groups = ['FRAMESTOR', 'HONE', 'NTb', 'KiNGDOM', 'BYNDR', 'PiRaTeS', 'HDO', 'CiNEPHiLES', 'SPiRiT', 'LUMiNA'];

  function filesFor(w) {
    const out = [];
    const want = profiles.find((p) => p.id === w.profile).targets;
    const has = w.coverage;
    const base = w.title.replace(/[^A-Za-z0-9 ]/g, '').replace(/ +/g, '.');
    if (has.uhd === 'met') out.push({ target: 'uhd', size: 52e9 + rnd() * 24e9, name: `${base}.${w.year}.2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.Atmos.7.1-${pick(groups)}`, video: 'HEVC · Dolby Vision P7 + HDR10', audio: 'TrueHD Atmos 7.1', resolution: '3840×2160', node: 'vault' });
    if (has.hd === 'met' || has.hd === 'upgradable') out.push({ target: 'hd', size: 9e9 + rnd() * 8e9, name: `${base}.${w.year}.1080p.BluRay.x264.DTS-HD.MA.5.1-${pick(groups)}`, video: 'AVC · SDR', audio: 'DTS-HD MA 5.1', resolution: '1920×1080', node: 'vault' });
    if (has.mobile === 'met') out.push({ target: 'mobile', size: 2.1e9 + rnd() * 1e9, name: `${base}.${w.year}.720p.Mobile.HEVC.AAC.2.0`, video: 'HEVC · 4 Mb/s', audio: 'AAC 2.0', resolution: '1280×720', node: 'vault', derived: true });
    if (!want.includes('uhd')) return out.filter((f) => f.target !== 'uhd');
    return out;
  }

  function coverageFor(status, profile) {
    const want = profiles.find((p) => p.id === profile).targets;
    const c = {};
    for (const t of ['uhd', 'hd', 'mobile']) {
      if (!want.includes(t)) { c[t] = 'unwanted'; continue; }
      c[t] = {
        available: 'met',
        upgradable: t === 'uhd' ? 'missing' : t === 'hd' ? 'upgradable' : 'met',
        downloading: t === 'uhd' ? 'downloading' : 'waiting',
        missing: 'missing',
        failed: 'missing',
        unreleased: 'unreleased',
      }[status];
    }
    return c;
  }

  const works = [];
  let addedRank = 0;
  for (const m of MOVIES) {
    const [id, title, year, runtime, genres, cert, rating, status, pal, motif, font, tagline, overview, extra = {}] = m;
    let profile = genres.includes('Family') || genres.includes('Animation') ? 'kids' : rnd() < 0.72 ? 'cinema' : 'standard';
    if (status === 'downloading') profile = 'cinema';
    const w = {
      id, type: 'movie', title, year, runtime, genres, cert, rating, status, tagline, overview,
      director: person(), cast: [person(), person(), person(), person()], studio: pick(studios), profile,
      editions: editionsFor(id),
      added: ago((addedRank++ < 6 ? addedRank * 0.4 : addedRank * 2.7 + rnd() * 5) * D),
      addedBy: null,
      art: { palette: P[pal], paletteName: pal, motif, font, seed: Math.floor(rnd() * 1e9) },
      ...extra,
    };
    w.coverage = coverageFor(status, profile);
    w.files = filesFor(w);
    works.push(w);
  }

  const today = NOW;
  // Weekly air slot of each running season's first episode.
  const anchors = {
    brightwater: at(2026, 9, 8, 21, 0),
    'harbor-lights': at(2026, 8, 11, 22, 0),
    'signal-keepers': at(2026, 8, 24, 20, 0),
    'quiet-hours': at(2026, 9, 3, 22, 0),
    'arcadia-station': at(2026, 8, 2, 21, 0),
    'night-shift-halden': at(2026, 9, 17, 21, 0),
  };
  for (const s of SERIES) {
    const [id, title, startYear, endYear, network, genres, cert, rating, pal, motif, font, tagline, overview, seasons] = s;
    const continuing = endYear === null;
    const profile = rnd() < 0.5 ? 'cinema' : 'standard';
    const w = {
      id, type: 'series', title, year: startYear, endYear, network, genres, cert, rating, tagline, overview,
      continuing, creator: person(), cast: [person(), person(), person(), person()], profile, editions: ['Standard'],
      added: ago((40 + rnd() * 300) * D), addedBy: null,
      art: { palette: P[pal], paletteName: pal, motif, font, seed: Math.floor(rnd() * 1e9) },
      seasons: [],
    };
    seasons.forEach(([count, airYear], si) => {
      const isLast = si === seasons.length - 1;
      const eps = [];
      for (let e = 1; e <= count; e++) {
        let air;
        if (isLast && anchors[id]) air = new Date(anchors[id].getTime() + (e - 1) * 7 * D);
        else air = at(airYear, 2 + Math.floor((e - 1) / 4), 1 + ((e * 7) % 27), 21, 0);
        let st = 'available';
        if (air > today) st = 'unreleased';
        eps.push({ n: e, title: pick(epWords), air, status: st, runtime: 42 + Math.floor(rnd() * 18) });
      }
      w.seasons.push({ n: si + 1, year: airYear, episodes: eps });
    });
    works.push(w);
  }

  const W = Object.fromEntries(works.map((w) => [w.id, w]));
  const ep = (sid, s, e) => W[sid].seasons[s - 1].episodes[e - 1];

  // Tonight's state for featured series
  ep('brightwater', 3, 5).status = 'downloading';
  ep('brightwater', 3, 5).title = 'The Drowned Bell';
  ep('brightwater', 3, 6).title = 'Spillway';
  ep('brightwater', 3, 4).title = 'Silt';
  ep('harbor-lights', 2, 6).status = 'missing';
  ep('night-shift-halden', 3, 3).status = 'failed';
  ep('arcadia-station', 1, 6).status = 'missing';
  for (const w of works.filter((x) => x.type === 'series')) {
    const all = w.seasons.flatMap((s) => s.episodes);
    const aired = all.filter((e) => e.status !== 'unreleased');
    w.counts = {
      total: all.length,
      aired: aired.length,
      available: all.filter((e) => e.status === 'available').length,
      missing: all.filter((e) => e.status === 'missing' || e.status === 'failed').length,
      downloading: all.filter((e) => e.status === 'downloading').length,
      upcoming: all.filter((e) => e.status === 'unreleased').length,
    };
    w.status = w.counts.downloading ? 'downloading' : w.counts.missing ? 'missing' : w.counts.available === w.counts.total ? 'available' : 'available';
    w.nextAir = all.find((e) => e.status === 'unreleased')?.air || null;
    w.coverage = coverageFor(w.counts.missing ? 'missing' : 'available', w.profile);
    w.size = all.filter((e) => e.status === 'available').length * (2.4e9 + rnd() * 2e9);
  }
  W['quiet-hours'].coverage = coverageFor('available', W['quiet-hours'].profile);

  const roleGroups = [
    { id: 'acquire', name: 'Acquire', roles: ['Searching', 'Downloading'] },
    { id: 'library', name: 'Library', roles: ['Storage', 'Metadata'] },
    { id: 'process', name: 'Process', roles: ['Transcoding', 'Subtitles'] },
    { id: 'deliver', name: 'Deliver', roles: ['Media Server link', 'Serving', 'Relaying'] },
    { id: 'notify', name: 'Notify', roles: ['Notifying'] },
  ];
  const roles = roleGroups.flatMap((g) => g.roles.map((r) => ({ name: r, group: g.id })));

  const nodes = [
    { id: 'atlas', name: 'atlas', controller: true, hardware: 'Mac mini · M4 Pro · 48 GB', os: 'macOS 27', place: 'Study', address: '10.0.1.2', roles: ['Searching', 'Metadata', 'Notifying', 'Serving'], state: 'online', cpu: 0.11, mem: 0.38, down: 0, up: 0, temp: 41, uptime: 19 * D, version: '2.3.0', disks: [{ path: 'Internal SSD', total: 2e12, used: 0.71e12 }] },
    { id: 'vault', name: 'vault', hardware: 'TrueNAS · Xeon E-2386G · 8 × 16 TB', os: 'TrueNAS SCALE', place: 'Closet rack', address: '10.0.1.10', roles: ['Storage', 'Media Server link', 'Subtitles'], state: 'online', cpu: 0.24, mem: 0.52, down: 0, up: 0, temp: 46, uptime: 63 * D, version: '2.3.0', disks: [{ path: '/mnt/media', total: 96e12, used: 71.4e12 }, { path: '/mnt/staging', total: 4e12, used: 0.9e12 }] },
    { id: 'forge', name: 'forge', hardware: 'Workstation · Ryzen 9 9950X · RTX 5080', os: 'Ubuntu 26.04', place: 'Office', address: '10.0.1.21', roles: ['Transcoding', 'Subtitles'], state: 'degraded', stateNote: 'Transcode queue 2 h behind; GPU at 87 °C', cpu: 0.81, mem: 0.64, down: 0, up: 0, temp: 87, gpu: 0.94, uptime: 4 * D, version: '2.3.0', disks: [{ path: '/scratch', total: 2e12, used: 1.31e12 }] },
    { id: 'ams-1', name: 'ams-1', hardware: 'Seedbox · 10 Gb/s · 8 TB', os: 'Debian 13', place: 'Amsterdam', address: 'ams-1.seed.example', roles: ['Downloading'], state: 'online', remote: true, cpu: 0.33, mem: 0.41, down: 0, up: 0, temp: null, uptime: 41 * D, version: '2.3.0', disks: [{ path: '/data', total: 8e12, used: 5.2e12 }], seeding: 412, ratio: 3.84 },
    { id: 'relay', name: 'relay', hardware: 'VPS · 2 vCPU · 4 GB', os: 'Debian 13', place: 'Frankfurt', address: 'watch.rowan.example', roles: ['Relaying', 'Serving'], state: 'online', remote: true, cpu: 0.06, mem: 0.29, down: 0, up: 0, temp: null, uptime: 88 * D, version: '2.3.0', disks: [{ path: '/', total: 80e9, used: 21e9 }] },
    { id: 'den', name: 'den', hardware: 'ThinkPad X1 · i7-1365U', os: 'Fedora 44', place: 'Den', address: '10.0.1.33', roles: ['Transcoding'], state: 'offline', stateNote: 'Last seen 18:04, 3 h 38 m ago', lastSeen: new Date(2026, 9, 6, 18, 4), cpu: 0, mem: 0, down: 0, up: 0, temp: null, uptime: 0, version: '2.3.0', disks: [{ path: '/', total: 1e12, used: 0.42e12 }] },
  ];

  const accounts = [
    { id: 'alex', name: 'Alex Rowan', email: 'alex@rowan.example', admin: true, hue: 210, since: at(2024, 1, 4) },
    { id: 'maya', name: 'Maya Okafor', email: 'maya@okafor.example', admin: false, hue: 18, since: at(2025, 2, 11), quota: { movie: [3, 5], series: [1, 2] } },
    { id: 'theo', name: 'Theo Lindqvist', email: 'theo@lindqvist.example', admin: false, hue: 160, since: at(2025, 6, 2), quota: { movie: [1, 5], series: [2, 2] } },
    { id: 'june', name: 'June Park', email: 'june@park.example', admin: false, hue: 280, since: at(2025, 9, 20), quota: { movie: [2, 5], series: [0, 2] } },
    { id: 'sam', name: 'Sam Ortiz', email: 'sam@ortiz.example', admin: false, hue: 95, since: at(2026, 1, 8), quota: { movie: [0, 3], series: [0, 1] } },
    { id: 'rosa', name: 'Rosa Rowan', email: 'rosa@rowan.example', admin: false, hue: 340, since: at(2024, 1, 4), quota: { movie: [1, 5], series: [0, 2] } },
  ];
  W['halcyon-drift'].addedBy = 'june';
  W['juniper-falls'].addedBy = 'sam';
  W['paper-lanterns'].addedBy = 'rosa';
  W['moth-lantern'].addedBy = 'rosa';

  // Live jobs. progress is 0..1, rate in bytes/s.
  const jobs = [
    { id: 'j1', kind: 'download', work: 'halcyon-drift', label: 'Halcyon Drift', target: 'uhd', release: 'Halcyon.Drift.2026.2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.Atmos.7.1-FRAMESTOR', node: 'ams-1', size: 61.8e9, progress: 0.64, rate: 48.2e6, seeds: 212, peers: 37, state: 'running', indexer: 'Halide', requestedBy: 'june' },
    { id: 'j2', kind: 'download', work: 'brightwater', episode: [3, 5], label: 'Brightwater · S3 E5', target: 'hd', release: 'Brightwater.S03E05.The.Drowned.Bell.1080p.HLX.WEB-DL.DDP5.1.H.264-NTb', node: 'ams-1', size: 3.4e9, progress: 0.31, rate: 22.4e6, seeds: 1840, peers: 412, state: 'running', indexer: 'Kestrel' },
    { id: 'j3', kind: 'download', work: 'velvet-hour', label: 'The Velvet Hour', target: 'uhd', release: 'The.Velvet.Hour.2026.2160p.WEB-DL.DV.HDR.DDP5.1.Atmos.H.265-BYNDR', node: 'ams-1', size: 18.2e9, progress: 0.12, rate: 0, seeds: 0, peers: 3, state: 'stalled', message: 'No seeders for 41 minutes', indexer: 'Aperture' },
    { id: 'j4', kind: 'transfer', work: 'glass-orchard', label: 'Glass Orchard', target: 'uhd', from: 'ams-1', to: 'vault', node: 'vault', size: 22.6e9, progress: 0.78, rate: 94.0e6, streams: 16, state: 'running' },
    { id: 'j5', kind: 'import', work: 'saltwater-kings', label: 'Saltwater Kings', target: 'uhd', node: 'vault', size: 58.1e9, progress: 0.45, rate: 640e6, state: 'running', step: 'Verifying chunks' },
    { id: 'j6', kind: 'transcode', work: 'neon-psalm', label: 'Neon Psalm', target: 'mobile', from: 'hd', node: 'forge', size: 2.6e9, progress: 0.37, rate: 0, speed: 4.1, encoder: 'NVENC HEVC', state: 'running' },
    { id: 'j7', kind: 'subtitles', work: 'low-tide-merrow', label: 'Low Tide at Merrow', target: 'hd', node: 'forge', progress: 0, state: 'queued', step: 'English · timing sync' },
    { id: 'j8', kind: 'transcode', work: 'atlas-small-hours', label: 'Atlas of Small Hours', target: 'mobile', from: 'hd', node: 'forge', size: 2.2e9, progress: 0, state: 'queued' },
    { id: 'j9', kind: 'import', work: 'coldwater-static', label: 'Coldwater Static', target: 'hd', node: 'vault', size: 48e6, progress: 1, state: 'failed', message: 'Release is a sample: 48 MB, expected at least 2 GB', release: 'Coldwater.Static.2025.1080p.WEBRip.x264-SPiRiT' },
  ];

  const requests = [
    { id: 'r1', work: 'ember-season', by: 'maya', at: ago(2 * H + 14 * M), state: 'pending', note: '' },
    { id: 'r2', work: 'arcadia-station', by: 'theo', at: ago(19 * H), state: 'pending', seasons: [1], note: 'The first season is enough to see if we like it' },
    { id: 'r3', work: 'halcyon-drift', by: 'june', at: ago(1 * D + 3 * H), state: 'downloading', job: 'j1' },
    { id: 'r4', work: 'juniper-falls', by: 'sam', at: ago(3 * D), state: 'available', doneAt: ago(1 * D + 6 * H) },
    { id: 'r5', work: 'amber-protocol', by: 'maya', at: ago(5 * D), state: 'approved', note: '' },
    { id: 'r6', work: 'second-spring', by: 'rosa', at: ago(4 * H), state: 'pending' },
    { id: 'r7', work: 'paper-lanterns', by: 'rosa', at: ago(9 * D), state: 'available', doneAt: ago(12 * M) },
    { id: 'r8', work: 'sundown-mechanics', by: 'theo', at: ago(12 * D), state: 'denied', reason: 'Already in the library. It is under Movies.' },
  ];

  const alerts = [
    { id: 'a1', level: 'error', title: 'den is offline', detail: 'Last seen at 18:04. Its Transcoding Role is covered by forge.', node: 'den', at: new Date(2026, 9, 6, 18, 4), action: 'Open den' },
    { id: 'a2', level: 'error', title: 'Coldwater Static failed to import', detail: 'The release is a 48 MB sample. Blocklist it and search again?', work: 'coldwater-static', at: ago(26 * M), action: 'Blocklist and search' },
    { id: 'a3', level: 'warning', title: 'The Velvet Hour has stalled', detail: 'No seeders for 41 minutes on ams-1.', work: 'velvet-hour', at: ago(41 * M), action: 'Find another release' },
    { id: 'a4', level: 'warning', title: 'forge is behind', detail: 'Transcode queue is 2 h behind. GPU at 87 °C.', node: 'forge', at: ago(55 * M), action: 'Open forge' },
    { id: 'a5', level: 'info', title: 'Halide is rate limited', detail: 'Searches resume at 22:10.', at: ago(18 * M) },
    { id: 'a6', level: 'info', title: 'SlipStream 2.3.1 is ready', detail: 'atlas updates first, then each Worker in turn.', at: ago(6 * H), action: 'Review update' },
  ];

  const history = [
    { at: ago(12 * M), kind: 'imported', work: 'paper-lanterns', target: 'uhd', text: 'Imported in 2160p HDR', node: 'vault' },
    { at: ago(26 * M), kind: 'failed', work: 'coldwater-static', target: 'hd', text: 'Import failed: sample release', node: 'vault' },
    { at: ago(38 * M), kind: 'grabbed', work: 'brightwater', episode: [3, 5], target: 'hd', text: 'Grabbed S3 E5 from Kestrel', node: 'ams-1' },
    { at: ago(1 * H + 5 * M), kind: 'derived', work: 'moth-lantern', target: 'mobile', text: 'Derived 720p Mobile on forge', node: 'forge' },
    { at: ago(2 * H + 30 * M), kind: 'upgraded', work: 'driftless', target: 'uhd', text: 'Upgraded to 2160p HDR remux', node: 'vault' },
    { at: ago(3 * H), kind: 'transferred', work: 'paper-lanterns', target: 'uhd', text: 'Transfer from ams-1 completed, 58.2 GB in 9 m 41 s', node: 'vault' },
    { at: ago(5 * H), kind: 'imported', work: 'brightwater', episode: [3, 4], target: 'hd', text: 'Imported S3 E4 in 1080p', node: 'vault' },
    { at: ago(1 * D + 6 * H), kind: 'imported', work: 'juniper-falls', target: 'hd', text: 'Imported in 1080p', node: 'vault' },
  ];

  const calendar = [
    { at: new Date(2026, 9, 6, 22, 0), work: 'harbor-lights', episode: [2, 9], title: 'Night Ferry', kind: 'airs' },
    { at: new Date(2026, 9, 8, 21, 0), work: 'night-shift-halden', episode: [3, 4], title: 'Clearance', kind: 'airs' },
    { at: new Date(2026, 9, 8, 22, 0), work: 'quiet-hours', episode: [1, 6], title: 'Last Call', kind: 'airs' },
    { at: new Date(2026, 9, 9, 0, 0), work: 'wolf-hour', kind: 'release', title: 'Digital release' },
    { at: new Date(2026, 9, 12, 20, 0), work: 'signal-keepers', episode: [2, 8], title: 'Perigee', kind: 'airs' },
    { at: new Date(2026, 9, 13, 21, 0), work: 'brightwater', episode: [3, 6], title: 'Spillway', kind: 'airs' },
  ];
  for (const c of calendar) if (c.episode) ep(c.work, ...c.episode).title = c.title;

  const playing = [
    { who: 'theo', work: 'brightwater', episode: [3, 4], device: 'Living Room TV', mode: 'Direct play', progress: 0.42, target: 'hd' },
    { who: 'rosa', work: 'paper-lanterns', device: 'iPad', mode: 'Direct play', progress: 0.08, target: 'mobile', remote: true },
  ];

  const indexers = [
    { id: 'halide', name: 'Halide', kind: 'Private', state: 'limited', note: 'Rate limited until 22:10' },
    { id: 'kestrel', name: 'Kestrel', kind: 'Private', state: 'ok' },
    { id: 'aperture', name: 'Aperture', kind: 'Private', state: 'ok' },
    { id: 'lumen', name: 'Lumen Torznab', kind: 'Torznab', state: 'ok' },
  ];

  const mediaServers = [{ id: 'plex', name: 'Plex', node: 'vault', state: 'online', streams: 2 }];

  SS.data = {
    now: NOW, targets, profiles, works, W, nodes, N: Object.fromEntries(nodes.map((n) => [n.id, n])),
    roleGroups, roles, accounts, A: Object.fromEntries(accounts.map((a) => [a.id, a])),
    me: 'alex', member: 'maya', jobs, requests, alerts, history, calendar, playing, indexers, mediaServers,
    contentTypes: [{ id: 'movie', name: 'Movies' }, { id: 'series', name: 'Series' }],
  };
})();
