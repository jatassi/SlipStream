import type { HealthIssue, HistoryEvent, MediaItem, QueueItem, SettingsSection } from './types'

export const MOVIES: MediaItem[] = [
  { id: 1, kind: 'movie', title: 'Dune: Part Two', year: 2024, status: 'downloading', quality: 'Bluray-1080p', sizeGb: 14.2, length: '2h 46m', rating: 8.5, genres: ['Sci-Fi', 'Adventure'], studio: 'Legendary', profile: 'Ultra HD', monitored: true, art: ['#c2703a', '#2b1a12'], added: '3h ago', overview: 'Paul Atreides unites with Chani and the Fremen while seeking revenge against the conspirators who destroyed his family, facing a choice between the love of his life and the fate of the known universe.' },
  { id: 2, kind: 'movie', title: 'Oppenheimer', year: 2023, status: 'available', quality: 'Remux-2160p', sizeGb: 58.6, length: '3h 0m', rating: 8.3, genres: ['Drama', 'History'], studio: 'Universal', profile: 'Ultra HD', monitored: true, art: ['#f2a33a', '#0f0f10'], added: '2mo ago', overview: 'The story of J. Robert Oppenheimer and his role in the development of the atomic bomb.' },
  { id: 3, kind: 'movie', title: 'Poor Things', year: 2023, status: 'available', quality: 'Bluray-1080p', sizeGb: 11.4, length: '2h 21m', rating: 7.9, genres: ['Comedy', 'Drama'], studio: 'Searchlight', profile: 'HD-1080p', monitored: true, art: ['#2aa7a1', '#d96ba1'], added: '12m ago', overview: 'Brought back to life by an unorthodox scientist, a young woman runs off with a lawyer on a whirlwind adventure across the continents.' },
  { id: 4, kind: 'movie', title: 'The Holdovers', year: 2023, status: 'upgradable', quality: 'WEBDL-1080p', sizeGb: 6.1, length: '2h 13m', rating: 7.9, genres: ['Comedy', 'Drama'], studio: 'Focus Features', profile: 'Ultra HD', monitored: true, art: ['#7a4b2a', '#e6d7b7'], added: '2h ago', overview: 'A curmudgeonly instructor at a New England prep school remains on campus during the holidays with a troubled student.' },
  { id: 5, kind: 'movie', title: 'Past Lives', year: 2023, status: 'available', quality: 'Bluray-1080p', sizeGb: 9.7, length: '1h 46m', rating: 7.8, genres: ['Drama', 'Romance'], studio: 'A24', profile: 'HD-1080p', monitored: true, art: ['#5b7fb8', '#f0c9a8'], added: '3w ago', overview: 'Nora and Hae Sung, two childhood friends, are reunited in New York for one fateful week as they confront destiny, love and the choices that make a life.' },
  { id: 6, kind: 'movie', title: 'Anatomy of a Fall', year: 2023, status: 'missing', quality: '—', sizeGb: 0, length: '2h 31m', rating: 7.7, genres: ['Drama', 'Thriller'], studio: 'Le Pacte', profile: 'HD-1080p', monitored: true, art: ['#dfe4ea', '#3a3f47'], added: '5d ago', overview: 'A woman is suspected of her husband\u2019s murder, and their blind son faces a moral dilemma as the sole witness.' },
  { id: 7, kind: 'movie', title: 'Civil War', year: 2024, status: 'available', quality: 'WEBDL-2160p', sizeGb: 19.3, length: '1h 49m', rating: 7, genres: ['Action', 'Thriller'], studio: 'A24', profile: 'Ultra HD', monitored: true, art: ['#b62a2a', '#1c1c1f'], added: '1w ago', overview: 'In the near future, a group of war journalists attempt to survive while reporting the truth as the United States stands on the brink of civil war.' },
  { id: 8, kind: 'movie', title: 'Challengers', year: 2024, status: 'downloading', quality: '—', sizeGb: 0, length: '2h 11m', rating: 7.3, genres: ['Drama', 'Romance'], studio: 'MGM', profile: 'HD-1080p', monitored: true, art: ['#3fbf5f', '#e8e3c8'], added: '1d ago', overview: 'Tashi, a former tennis prodigy turned coach, is married to a champion on a losing streak. Her strategy takes a surprising turn when he must face his former best friend.' },
  { id: 9, kind: 'movie', title: 'The Zone of Interest', year: 2023, status: 'available', quality: 'Bluray-1080p', sizeGb: 8.8, length: '1h 45m', rating: 7.4, genres: ['Drama', 'History'], studio: 'A24', profile: 'HD-1080p', monitored: false, art: ['#7d8a6d', '#1a1c17'], added: '1mo ago', overview: 'The commandant of Auschwitz and his wife strive to build a dream life for their family in a house and garden next to the camp.' },
  { id: 10, kind: 'movie', title: 'Perfect Days', year: 2023, status: 'missing', quality: '—', sizeGb: 0, length: '2h 4m', rating: 7.9, genres: ['Drama'], studio: 'Neon', profile: 'HD-1080p', monitored: true, art: ['#7ab8c8', '#3a5a40'], added: '2d ago', overview: 'Hirayama seems utterly content with his simple life as a cleaner of toilets in Tokyo. A series of unexpected encounters reveal more of his past.' },
  { id: 11, kind: 'movie', title: 'Furiosa: A Mad Max Saga', year: 2024, status: 'downloading', quality: '—', sizeGb: 0, length: '2h 28m', rating: 7.6, genres: ['Action', 'Sci-Fi'], studio: 'Warner Bros.', profile: 'Ultra HD', monitored: true, art: ['#e0842c', '#3b2a6d'], added: '6h ago', overview: 'As the world falls, young Furiosa is snatched from the Green Place of Many Mothers and falls into the hands of a great biker horde led by the warlord Dementus.' },
  { id: 12, kind: 'movie', title: 'Godzilla Minus One', year: 2023, status: 'available', quality: 'Bluray-2160p', sizeGb: 41.2, length: '2h 5m', rating: 7.7, genres: ['Sci-Fi', 'Drama'], studio: 'Toho', profile: 'Ultra HD', monitored: true, art: ['#2b3a67', '#c7c9cf'], added: '3mo ago', overview: 'Postwar Japan is at its lowest point when a new crisis emerges in the form of a giant monster, baptized in the horrific power of the atomic bomb.' },
]

export const SERIES: MediaItem[] = [
  { id: 101, kind: 'series', title: 'Shōgun', year: 2024, status: 'downloading', quality: 'WEBDL-1080p', sizeGb: 24.6, length: '1 season · 10 ep', rating: 8.7, genres: ['Drama', 'History'], studio: 'FX', profile: 'HD-1080p', monitored: true, art: ['#8b1e1e', '#111111'], added: '24m ago', overview: 'When a mysterious European ship is found marooned in a nearby fishing village, Lord Yoshii Toranaga discovers secrets that could tip the scales of power.' },
  { id: 102, kind: 'series', title: 'Severance', year: 2022, status: 'available', quality: 'WEBDL-2160p', sizeGb: 61.3, length: '2 seasons · 19 ep', rating: 8.7, genres: ['Drama', 'Mystery'], studio: 'Apple TV+', profile: 'Ultra HD', monitored: true, art: ['#0f3d3e', '#c9e4e4'], added: '1d ago', overview: 'Mark leads a team of office workers whose memories have been surgically divided between their work and personal lives.' },
  { id: 103, kind: 'series', title: 'The Bear', year: 2022, status: 'downloading', quality: 'WEBDL-1080p', sizeGb: 33.8, length: '3 seasons · 28 ep', rating: 8.5, genres: ['Comedy', 'Drama'], studio: 'FX', profile: 'HD-1080p', monitored: true, art: ['#1f3a93', '#f5f5f5'], added: '2mo ago', overview: 'A young chef from the fine dining world returns to Chicago to run his family\u2019s sandwich shop.' },
  { id: 104, kind: 'series', title: 'Slow Horses', year: 2022, status: 'available', quality: 'WEBDL-2160p', sizeGb: 48.1, length: '4 seasons · 24 ep', rating: 8.2, genres: ['Thriller', 'Drama'], studio: 'Apple TV+', profile: 'Ultra HD', monitored: true, art: ['#5a5a5a', '#e0c070'], added: '4mo ago', overview: 'A team of British intelligence agents serve in a dumping ground department of MI5 due to their career-ending mistakes.' },
  { id: 105, kind: 'series', title: 'Fallout', year: 2024, status: 'available', quality: 'WEBDL-2160p', sizeGb: 39.4, length: '1 season · 8 ep', rating: 8.4, genres: ['Sci-Fi', 'Adventure'], studio: 'Prime Video', profile: 'Ultra HD', monitored: true, art: ['#f7c948', '#1f4e79'], added: '5mo ago', overview: 'In a future, post-apocalyptic Los Angeles, citizens must live in underground bunkers to protect themselves from radiation, mutants and bandits.' },
  { id: 106, kind: 'series', title: 'Fargo', year: 2014, status: 'upgradable', quality: 'WEBDL-1080p', sizeGb: 92.7, length: '5 seasons · 51 ep', rating: 8.9, genres: ['Crime', 'Drama'], studio: 'FX', profile: 'Ultra HD', monitored: false, art: ['#c8d6e5', '#3d0c02'], added: '1y ago', overview: 'Various chronicles of deception, intrigue and murder in and around frozen Minnesota. Yet all of these tales mysteriously lead back one way or another to Fargo.' },
  { id: 107, kind: 'series', title: 'Blue Eye Samurai', year: 2023, status: 'available', quality: 'WEBDL-1080p', sizeGb: 12.9, length: '1 season · 8 ep', rating: 8.6, genres: ['Animation', 'Action'], studio: 'Netflix', profile: 'HD-1080p', monitored: true, art: ['#0b3d91', '#ff6b35'], added: '8mo ago', overview: 'Driven by a dream of revenge against those who made her an outcast in Edo-period Japan, a young warrior cuts a bloody path toward her destiny.' },
  { id: 108, kind: 'series', title: 'Silo', year: 2023, status: 'missing', quality: 'WEBDL-1080p', sizeGb: 27.5, length: '2 seasons · 20 ep', rating: 8.1, genres: ['Sci-Fi', 'Drama'], studio: 'Apple TV+', profile: 'HD-1080p', monitored: true, art: ['#3a3f44', '#c56a2a'], added: '5h ago', overview: 'In a ruined and toxic future, a community exists in a giant underground silo that plunges hundreds of stories deep.' },
  { id: 109, kind: 'series', title: 'Ripley', year: 2024, status: 'available', quality: 'WEBDL-2160p', sizeGb: 44, length: '1 season · 8 ep', rating: 8, genres: ['Crime', 'Drama'], studio: 'Netflix', profile: 'Ultra HD', monitored: true, art: ['#1a1a1a', '#e8e8e8'], added: '6mo ago', overview: 'Tom Ripley, a grifter scraping by in early 1960s New York, is hired by a wealthy man to travel to Italy to try to convince his vagabond son to return home.' },
  { id: 110, kind: 'series', title: 'Baby Reindeer', year: 2024, status: 'available', quality: 'WEBDL-1080p', sizeGb: 9.2, length: '1 season · 7 ep', rating: 7.8, genres: ['Drama', 'Thriller'], studio: 'Netflix', profile: 'HD-1080p', monitored: true, art: ['#7a1f3d', '#efe7dd'], added: '5mo ago', overview: 'When a struggling comedian shows one act of kindness to a vulnerable woman, it sparks a suffocating obsession which threatens to wreck both their lives.' },
]

export const LIBRARY: MediaItem[] = [...MOVIES, ...SERIES]

export const QUEUE: QueueItem[] = [
  { id: 'q1', mediaId: 1, release: 'Dune.Part.Two.2024.2160p.UHD.BluRay.REMUX.DV.HDR.TrueHD.7.1.Atmos-FraMeSToR', progress: 72.4, sizeGb: 64.2, speedMbps: 48.2, etaMin: 6, state: 'downloading', protocol: 'torrent', client: 'qBittorrent' },
  { id: 'q2', mediaId: 101, episode: 'S01E09 · Crimson Sky', release: 'Shogun.2024.S01E09.Crimson.Sky.1080p.DSNP.WEB-DL.DDP5.1.H.264-NTb', progress: 41, sizeGb: 3.1, speedMbps: 22.6, etaMin: 1, state: 'downloading', protocol: 'usenet', client: 'SABnzbd' },
  { id: 'q3', mediaId: 103, episode: 'Season 3 · 10 episodes', release: 'The.Bear.S03.1080p.HULU.WEB-DL.DDP5.1.H.264-FLUX', progress: 12.8, sizeGb: 18.7, speedMbps: 0, etaMin: 0, state: 'paused', protocol: 'torrent', client: 'qBittorrent' },
  { id: 'q4', mediaId: 8, release: 'Challengers.2024.1080p.BluRay.x264-VETO', progress: 100, sizeGb: 9.8, speedMbps: 0, etaMin: 0, state: 'importing', protocol: 'torrent', client: 'qBittorrent' },
  { id: 'q5', mediaId: 11, release: 'Furiosa.A.Mad.Max.Saga.2024.2160p.MA.WEB-DL.DDP5.1.Atmos.DV.HDR.H.265-FLUX', progress: 0, sizeGb: 22.4, speedMbps: 0, etaMin: 0, state: 'queued', protocol: 'usenet', client: 'SABnzbd' },
]

export const HISTORY: HistoryEvent[] = [
  { id: 'h1', mediaId: 3, event: 'Imported', detail: 'Bluray-1080p · 11.4 GB', ago: '12m' },
  { id: 'h2', mediaId: 101, event: 'Grabbed', detail: 'S01E09 · WEBDL-1080p · NTb', ago: '24m' },
  { id: 'h3', mediaId: 4, event: 'Upgraded', detail: 'WEBDL-1080p → Bluray-1080p', ago: '2h' },
  { id: 'h4', mediaId: 1, event: 'Grabbed', detail: 'Remux-2160p · FraMeSToR', ago: '3h' },
  { id: 'h5', mediaId: 108, event: 'Failed', detail: 'S02E01 · No results from 4 indexers', ago: '5h' },
  { id: 'h6', mediaId: 102, event: 'Imported', detail: 'S02E03 · WEBDL-2160p', ago: '1d' },
  { id: 'h7', mediaId: 7, event: 'Imported', detail: 'WEBDL-2160p · 19.3 GB', ago: '1w' },
]

export const HEALTH: HealthIssue[] = [
  { id: 'i1', level: 'warning', source: 'Indexer', message: 'TorrentLeech failed 3 consecutive queries — backing off 30 min' },
  { id: 'i2', level: 'warning', source: 'Root folder', message: '/mnt/media/tv is 91% full' },
]

export const STORAGE = {
  usedTb: 8.14,
  totalTb: 12,
  folders: [
    { path: '/mnt/media/movies', usedTb: 5.2, kind: 'movie' as const },
    { path: '/mnt/media/tv', usedTb: 2.94, kind: 'series' as const },
  ],
}

export const COUNTS = { movies: 412, series: 38, episodes: 1764, missingMovies: 11, missingEpisodes: 6 }

export const SETTINGS: SettingsSection[] = [
  { id: 'media', title: 'Media', items: [
    { title: 'Root Folders', detail: '2 folders' },
    { title: 'Quality Profiles', detail: 'Ultra HD, HD-1080p' },
    { title: 'Version Slots', detail: '3 slots' },
    { title: 'File Naming', detail: 'Movie · Series' },
    { title: 'Import from Radarr/Sonarr', detail: '' },
  ] },
  { id: 'pipeline', title: 'Download Pipeline', items: [
    { title: 'Indexers', detail: '4 · 1 warning' },
    { title: 'Download Clients', detail: 'qBittorrent, SABnzbd' },
    { title: 'Auto Search', detail: 'Every 6h' },
    { title: 'RSS Sync', detail: 'Every 15m' },
  ] },
  { id: 'general', title: 'General', items: [
    { title: 'Server', detail: ':8080' },
    { title: 'Authentication', detail: 'Passkey' },
    { title: 'Notifications', detail: 'Discord, Pushover' },
  ] },
]

export const TASKS = [
  { id: 't1', name: 'RSS Sync', last: '4m ago', next: 'in 11m', status: 'idle' },
  { id: 't2', name: 'Auto Search · Missing', last: '2h ago', next: 'in 4h', status: 'idle' },
  { id: 't3', name: 'Refresh Metadata', last: 'running · 38%', next: '—', status: 'running' },
  { id: 't4', name: 'Housekeeping', last: '1d ago', next: 'in 23h', status: 'idle' },
]