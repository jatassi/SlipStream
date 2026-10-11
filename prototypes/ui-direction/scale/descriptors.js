// PROTOTYPE. What each Content Type hands the core so the Work page renders without per-type code (The Content Type contract, Q17 and Q21).
// The renderers read these and the Work tree only. They never test a Content Type's id: `node scale/check.mjs` fails if one does.
//
// Layout hints:
//   items      'one'  the Work holds a single Item, so the matrix is the page.
//              'many' the Work holds many Items under its levels, so the matrix multiplies.
//   levels[].layout  'section' each node is a collapsible section.  open: which sections start open.
//   item.rowLabel    how one Item reads in a row; item.code is its short key for dense views.

const pad = (n) => String(n).padStart(2, '0');

export const contentTypes = {
  movie: {
    id: 'movie',
    noun: 'Movie',
    levels: [],
    item: { noun: 'Movie', label: () => 'Movie', code: () => '', rowLabel: () => '' },
    display: [
      { key: 'year', mono: true },
      { key: 'kind', value: () => 'Movie' },
      { key: 'runtime' },
      { key: 'rating', badge: true },
      { key: 'genres', join: ', ' },
    ],
    artwork: ['poster', 'backdrop', 'logo'],
    searchable: ['item'],
    layout: { items: 'one' },
    nav: 'movies',
    crumb: 'Movies',
  },
  series: {
    id: 'series',
    noun: 'Series',
    levels: [
      { id: 'season', noun: 'Season', label: (n) => (n === 0 ? 'Specials' : `Season ${n}`), short: (n) => `S${pad(n)}`, layout: 'section', open: 'wanted', searchable: true },
    ],
    item: {
      noun: 'Episode',
      label: (i) => i.title,
      code: (i, parent) => `S${pad(parent.n)}E${pad(i.n)}`,
      rowLabel: (i) => `${i.n}. ${i.title}`,
      dated: 'airs',
    },
    display: [
      { key: 'year', mono: true },
      { key: 'kind', value: () => 'Series' },
      { key: 'network' },
      { key: 'rating', badge: true },
      { key: 'genres', join: ', ' },
    ],
    artwork: ['poster', 'backdrop', 'logo'],
    searchable: ['season', 'item'],
    layout: { items: 'many', group: 'season' },
    nav: 'series',
    crumb: 'Series',
  },
};
