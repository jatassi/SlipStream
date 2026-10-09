// PROTOTYPE, throwaway. Every person, server, address, token and number here is invented. Artwork is v1's local cache (see run.sh).

const movie = (id) => ({ poster: `art/movie/${id}_poster.jpg`, backdrop: `art/movie/${id}_backdrop.jpg`, logo: `art/movie/${id}_logo.png` });
const series = (id) => ({ poster: `art/series/${id}_poster.jpg`, backdrop: `art/series/${id}_backdrop.jpg`, logo: `art/series/${id}_logo.png` });

export const now = new Date('2026-10-08T21:40:00');
export const zone = { id: 'America/Chicago', abbr: 'CDT' };

export const server = {
  name: 'SlipStream at the Atassi house',
  short: 'the Atassi house',
  https: 'https://watch.atassi.example',
  http: 'http://192.168.1.20:5055',
  from: 'slipstream@atassi.example',
};

export const admin = { name: 'Jackson', fullName: 'Jackson Atassi', initials: 'JA' };

// The person the Admin issued the Signup link to.
export const invitee = { name: 'Rosa Alvarez', first: 'Rosa', email: 'rosa@example.com', initials: 'RA' };

// The Member settings the Admin chose when issuing the link. The link carries them.
export const grants = { autoApprove: true, quota: 10 };
export const grantsLine = 'Your requests are approved automatically, up to 10 a week.';
export const grantsShort = 'auto-approved requests, up to 10 a week';

// For ?state=existing and ?state=signedin: Rosa already has an Account with fewer settings, so the link still gives her something.
export const existingAccount = { name: 'Rosa Alvarez', first: 'Rosa', email: 'rosa@example.com', initials: 'RA', passkey: false, quota: 3, autoApprove: false, requests: ['severance'] };

export const link = {
  token: 'k7Qm2vX9pL4tR8wN1cY6hB3fJ0sD5gZaEuTiOyWq_xM',
  rotated: 'Hc3uR0aYp7NnXe2tLqW9mZ5vJ1kBgS8dFiO4yTwQ_rE',
  sentAt: new Date('2026-10-08T21:40:00'),
  expiresAt: new Date('2026-10-15T21:40:00'),
};

// The same link sent again the next morning: a new token, a new seven days, and the first link void.
export const reissue = { sentAt: new Date('2026-10-09T08:15:00'), expiresAt: new Date('2026-10-16T08:15:00') };

export const works = {
  'dune-2': { id: 'dune-2', type: 'movie', title: 'Dune: Part Two', year: 2024, art: movie(693134), overview: 'Paul Atreides unites with Chani and the Fremen while seeking revenge against the conspirators who destroyed his family.' },
  oppenheimer: { id: 'oppenheimer', type: 'movie', title: 'Oppenheimer', year: 2023, art: movie(872585), overview: 'The story of J. Robert Oppenheimer and the making of the atomic bomb.' },
  interstellar: { id: 'interstellar', type: 'movie', title: 'Interstellar', year: 2014, art: movie(157336), overview: 'A team of explorers travels through a wormhole in search of a new home for humanity.' },
  'top-gun-maverick': { id: 'top-gun-maverick', type: 'movie', title: 'Top Gun: Maverick', year: 2022, art: movie(361743), overview: 'Maverick returns to train a group of graduates for a mission no living pilot has seen.' },
  eeaao: { id: 'eeaao', type: 'movie', title: 'Everything Everywhere All at Once', year: 2022, art: movie(545611), overview: 'A laundromat owner is swept into a multiverse adventure where only she can save existence.' },
  'the-batman': { id: 'the-batman', type: 'movie', title: 'The Batman', year: 2022, art: movie(414906), overview: 'In his second year of fighting crime, Batman uncovers corruption in Gotham City.' },
  'black-panther': { id: 'black-panther', type: 'movie', title: 'Black Panther', year: 2018, art: movie(284054), overview: "T'Challa returns home to Wakanda to take his place as king." },
  'the-matrix': { id: 'the-matrix', type: 'movie', title: 'The Matrix', year: 1999, art: movie(603), overview: 'A hacker learns that the world he knows is a simulation, and joins the fight against its makers.' },
  'the-dark-knight': { id: 'the-dark-knight', type: 'movie', title: 'The Dark Knight', year: 2008, art: movie(155), overview: 'Batman faces the Joker, a criminal who wants to watch Gotham burn.' },
  'the-prestige': { id: 'the-prestige', type: 'movie', title: 'The Prestige', year: 2006, art: movie(1124), overview: 'Two rival magicians in 1890s London push each other to dangerous lengths.' },
  severance: { id: 'severance', type: 'series', title: 'Severance', year: 2022, network: 'Apple TV+', art: series(95396), overview: 'Office workers have their work and personal memories surgically divided.' },
  'the-boys': { id: 'the-boys', type: 'series', title: 'The Boys', year: 2019, network: 'Prime Video', art: series(76479), overview: 'A group of vigilantes sets out to take down corrupt superheroes.' },
  hotd: { id: 'hotd', type: 'series', title: 'House of the Dragon', year: 2022, network: 'HBO', art: series(94997), overview: 'The Targaryen dynasty tears itself apart two hundred years before Game of Thrones.' },
  tlou: { id: 'tlou', type: 'series', title: 'The Last of Us', year: 2023, network: 'HBO', art: series(100088), overview: 'Joel and Ellie cross a ruined America, twenty years after a pandemic.' },
  bcs: { id: 'bcs', type: 'series', title: 'Better Call Saul', year: 2015, network: 'AMC', art: series(60059), overview: 'Small-time lawyer Jimmy McGill becomes Saul Goodman.' },
  got: { id: 'got', type: 'series', title: 'Game of Thrones', year: 2011, network: 'HBO', art: series(1399), overview: 'Noble families fight for control of the Seven Kingdoms of Westeros.' },
  narcos: { id: 'narcos', type: 'series', title: 'Narcos', year: 2015, network: 'Netflix', art: series(63351), overview: 'The rise and fall of the Medellin cartel and the agents who hunted it.' },
};

// What the library already holds, as a Member sees it. Anything not listed can be requested.
export const availability = {
  interstellar: 'ready', 'dune-2': 'ready', 'top-gun-maverick': 'ready', bcs: 'ready', 'the-prestige': 'ready', narcos: 'ready', tlou: 'ready', 'the-batman': 'ready',
  oppenheimer: 'coming', severance: 'coming',
};

export const trending = ['the-matrix', 'the-dark-knight', 'hotd', 'eeaao', 'the-boys', 'got', 'black-panther', 'oppenheimer'];
export const newInLibrary = ['interstellar', 'dune-2', 'top-gun-maverick', 'bcs', 'the-prestige', 'narcos'];
export const wall = ['dune-2', 'interstellar', 'severance', 'the-batman', 'oppenheimer', 'tlou', 'the-matrix', 'hotd', 'top-gun-maverick', 'bcs', 'the-dark-knight', 'got', 'eeaao', 'narcos', 'the-prestige', 'black-panther', 'the-boys', 'tlou'];

// ───────────── round 2: Media Servers. All invented, like everything above.

// The Admin's Media Servers. ?server= picks which one is switched on for Members; 'none' switches both off.
export const mediaServers = {
  plex: { id: 'plex', kind: 'Plex', name: 'Atassi Plex', plexName: 'Atassi house' },
  jellyfin: { id: 'jellyfin', kind: 'Jellyfin', name: 'Den Jellyfin', publicUrl: 'https://jellyfin.atassi.example' },
};

// The Plex account Rosa already has, or makes on the fake plex.tv.
export const plexAccount = { username: 'rosa.alvarez', email: 'rosa@example.com' };

// Other Members on the Admin's Media Server access list.
export const accessMembers = [
  { name: 'Dev Patel', initials: 'DP', email: 'dev@example.com', plex: { username: 'devp_watches', email: 'dev.patel@example.net' }, jfUser: 'dev', joined: 'Mon 5 Oct' },
  { name: 'Mina Cho', initials: 'MC', email: 'mina@example.com', plex: { username: 'minacho', email: 'mina@example.com' }, jfUser: 'mina', joined: 'Thu 1 Oct', grantedAt: 'Thu 1 Oct, 20:14' },
];
