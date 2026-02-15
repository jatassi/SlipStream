import { slotsApi } from '@/api'
import type {
  GeneratePreviewInput,
  MigrationPreview,
  MockMovie,
  MockTVShow,
} from '@/types'

type ReleaseTemplate = {
  resolution: string
  source: string
  hdr?: string
  videoCodec: string
  audioCodec: string
  audioChannels: string
  group: string
}

const RELEASE_TEMPLATES: ReleaseTemplate[] = [
  // 4K HDR variants
  {
    resolution: '2160p',
    source: 'UHD.BluRay.REMUX',
    hdr: 'DV.HDR10',
    videoCodec: 'HEVC',
    audioCodec: 'TrueHD.Atmos',
    audioChannels: '7.1',
    group: 'FraMeSToR',
  },
  {
    resolution: '2160p',
    source: 'UHD.BluRay.REMUX',
    hdr: 'DV.HDR10Plus',
    videoCodec: 'HEVC',
    audioCodec: 'DTS-HD.MA',
    audioChannels: '7.1',
    group: 'SiCFoI',
  },
  {
    resolution: '2160p',
    source: 'UHD.BluRay',
    hdr: 'HDR10',
    videoCodec: 'HEVC',
    audioCodec: 'DTS-HD.MA',
    audioChannels: '7.1',
    group: 'DON',
  },
  {
    resolution: '2160p',
    source: 'WEB-DL',
    hdr: 'DV.HDR10',
    videoCodec: 'HEVC',
    audioCodec: 'EAC3.Atmos',
    audioChannels: '5.1',
    group: 'FLUX',
  },
  {
    resolution: '2160p',
    source: 'WEBRip',
    hdr: 'HDR10',
    videoCodec: 'HEVC',
    audioCodec: 'EAC3',
    audioChannels: '5.1',
    group: 'NTb',
  },
  // 4K SDR variants
  {
    resolution: '2160p',
    source: 'UHD.BluRay.REMUX',
    videoCodec: 'HEVC',
    audioCodec: 'TrueHD.Atmos',
    audioChannels: '7.1',
    group: 'FraMeSToR',
  },
  {
    resolution: '2160p',
    source: 'WEB-DL',
    videoCodec: 'HEVC',
    audioCodec: 'EAC3',
    audioChannels: '5.1',
    group: 'NTG',
  },
  // 1080p variants
  {
    resolution: '1080p',
    source: 'BluRay.REMUX',
    videoCodec: 'AVC',
    audioCodec: 'DTS-HD.MA',
    audioChannels: '7.1',
    group: 'FraMeSToR',
  },
  {
    resolution: '1080p',
    source: 'BluRay',
    videoCodec: 'x264',
    audioCodec: 'DTS-HD.MA',
    audioChannels: '5.1',
    group: 'DON',
  },
  {
    resolution: '1080p',
    source: 'BluRay',
    videoCodec: 'x264',
    audioCodec: 'DTS',
    audioChannels: '5.1',
    group: 'EbP',
  },
  {
    resolution: '1080p',
    source: 'WEB-DL',
    videoCodec: 'x264',
    audioCodec: 'EAC3',
    audioChannels: '5.1',
    group: 'NTb',
  },
  {
    resolution: '1080p',
    source: 'WEB-DL',
    videoCodec: 'x264',
    audioCodec: 'AAC',
    audioChannels: '2.0',
    group: 'FLUX',
  },
  {
    resolution: '1080p',
    source: 'WEBRip',
    videoCodec: 'x264',
    audioCodec: 'AAC',
    audioChannels: '2.0',
    group: 'RARBG',
  },
  {
    resolution: '1080p',
    source: 'HDTV',
    videoCodec: 'x264',
    audioCodec: 'AC3',
    audioChannels: '5.1',
    group: 'LOL',
  },
  // 720p variants
  {
    resolution: '720p',
    source: 'BluRay',
    videoCodec: 'x264',
    audioCodec: 'DTS',
    audioChannels: '5.1',
    group: 'DON',
  },
  {
    resolution: '720p',
    source: 'WEB-DL',
    videoCodec: 'x264',
    audioCodec: 'AAC',
    audioChannels: '2.0',
    group: 'NTb',
  },
  {
    resolution: '720p',
    source: 'HDTV',
    videoCodec: 'x264',
    audioCodec: 'AC3',
    audioChannels: '5.1',
    group: 'LOL',
  },
  // SD variants (for no-match scenarios)
  {
    resolution: '480p',
    source: 'DVDRip',
    videoCodec: 'XviD',
    audioCodec: 'MP3',
    audioChannels: '2.0',
    group: 'aXXo',
  },
  {
    resolution: '576p',
    source: 'DVDRip',
    videoCodec: 'x264',
    audioCodec: 'AC3',
    audioChannels: '5.1',
    group: 'FGT',
  },
]

const MOVIE_TITLES = [
  'The Dark Knight',
  'Inception',
  'Interstellar',
  'The Matrix',
  'Pulp Fiction',
  'Fight Club',
  'Forrest Gump',
  'The Godfather',
  'Goodfellas',
  'The Shawshank Redemption',
  'Gladiator',
  'Braveheart',
  'Saving Private Ryan',
  'Schindlers List',
  'The Green Mile',
  'Se7en',
  'The Silence of the Lambs',
  'American History X',
  'The Departed',
  'Heat',
  'Casino',
  'Scarface',
  'The Prestige',
  'Memento',
  'Tenet',
  'Dunkirk',
  'Oppenheimer',
  'Dune',
  'Blade Runner 2049',
  'Arrival',
]

const TV_SHOW_TITLES = [
  'Breaking Bad',
  'Game of Thrones',
  'The Sopranos',
  'The Wire',
  'Mad Men',
  'Better Call Saul',
  'True Detective',
  'Fargo',
  'Westworld',
  'Succession',
  'House of the Dragon',
  'The Last of Us',
  'Severance',
  'Andor',
  'The Mandalorian',
]

function sanitizeTitle(title: string): string {
  return title.replaceAll(/[^a-zA-Z0-9\s]/g, '').replaceAll(/\s+/g, '.')
}

function templateParts(template: ReleaseTemplate): string[] {
  const parts = [template.resolution, template.source]
  if (template.hdr) {
    parts.push(template.hdr)
  }
  parts.push(template.videoCodec, template.audioCodec, template.audioChannels, template.group)
  return parts
}

function randomTemplate(): ReleaseTemplate {
  return RELEASE_TEMPLATES[Math.floor(Math.random() * RELEASE_TEMPLATES.length)]
}

function buildReleaseTitle(title: string, year: number, template: ReleaseTemplate): string {
  return [sanitizeTitle(title), year.toString(), ...templateParts(template)].join('.')
}

type EpisodeReleaseTitleOpts = {
  show: string
  season: number
  episode: number
  template: ReleaseTemplate
}

function buildEpisodeReleaseTitle(opts: EpisodeReleaseTitleOpts): string {
  const sStr = String(opts.season).padStart(2, '0')
  const eStr = String(opts.episode).padStart(2, '0')
  return [sanitizeTitle(opts.show), `S${sStr}E${eStr}`, ...templateParts(opts.template)].join('.')
}

function randomFileSize(baseGB: number, rangeGB: number): number {
  return Math.floor(Math.random() * rangeGB + baseGB) * 1024 * 1024 * 1024
}

function generateMockMovies(startFileId: number): { movies: MockMovie[]; nextFileId: number } {
  let fileId = startFileId
  const movies: MockMovie[] = []

  for (let i = 0; i < 40; i++) {
    const title = MOVIE_TITLES[i % MOVIE_TITLES.length]
    const year = 2000 + (i % 24)
    const fileCount = Math.random() > 0.7 ? 2 : 1

    const movie: MockMovie = { movieId: i + 1, title, year, files: [] }

    for (let j = 0; j < fileCount; j++) {
      const template = randomTemplate()
      const releaseTitle = buildReleaseTitle(title, year, template)
      movie.files.push({
        fileId: fileId++,
        path: `/media/movies/${title} (${year})/${releaseTitle}.mkv`,
        quality: `${template.resolution} ${template.source}`,
        size: randomFileSize(5, 50),
      })
    }

    movies.push(movie)
  }

  return { movies, nextFileId: fileId }
}

function generateEpisodeFiles(opts: {
  startFileId: number
  show: string
  season: number
  episode: number
}): { files: MockTVShow['seasons'][0]['episodes'][0]['files']; nextFileId: number } {
  const fileCount = Math.random() > 0.8 ? 2 : 1
  const files: MockTVShow['seasons'][0]['episodes'][0]['files'] = []
  let fileId = opts.startFileId

  for (let f = 0; f < fileCount; f++) {
    const template = randomTemplate()
    const releaseTitle = buildEpisodeReleaseTitle({
      show: opts.show,
      season: opts.season,
      episode: opts.episode,
      template,
    })
    files.push({
      fileId: fileId++,
      path: `/media/tv/${opts.show}/Season ${opts.season}/${releaseTitle}.mkv`,
      quality: `${template.resolution} ${template.source}`,
      size: randomFileSize(1, 5),
    })
  }

  return { files, nextFileId: fileId }
}

function generateMockTVShows(startFileId: number): { tvShows: MockTVShow[]; nextFileId: number } {
  let fileId = startFileId
  const tvShows: MockTVShow[] = []

  for (let i = 0; i < 10; i++) {
    const title = TV_SHOW_TITLES[i % TV_SHOW_TITLES.length]
    const seasonCount = 2 + (i % 3)
    const show: MockTVShow = { seriesId: i + 1, title, seasons: [] }

    for (let s = 1; s <= seasonCount; s++) {
      const episodeCount = 5 + (i % 6)
      const season = {
        seasonNumber: s,
        episodes: [] as MockTVShow['seasons'][0]['episodes'],
      }

      for (let e = 1; e <= episodeCount; e++) {
        const { files, nextFileId } = generateEpisodeFiles({
          startFileId: fileId,
          show: title,
          season: s,
          episode: e,
        })
        fileId = nextFileId
        season.episodes.push({
          episodeId: fileId,
          episodeNumber: e,
          title: `Episode ${e}`,
          files,
        })
      }

      show.seasons.push(season)
    }

    tvShows.push(show)
  }

  return { tvShows, nextFileId: fileId }
}

export async function generateDebugPreview(): Promise<MigrationPreview> {
  const { movies, nextFileId } = generateMockMovies(1)
  const { tvShows } = generateMockTVShows(nextFileId)

  const input: GeneratePreviewInput = { movies, tvShows }
  return slotsApi.generatePreview(input)
}
