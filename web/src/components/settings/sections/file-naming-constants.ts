export const VALIDATION_LEVELS = [
  { value: 'basic', label: 'Basic', description: 'File exists and size > 0' },
  { value: 'standard', label: 'Standard', description: 'Size > minimum, valid extension' },
  { value: 'full', label: 'Full', description: 'Size + extension + MediaInfo probe' },
]

export const MATCH_CONFLICT_OPTIONS = [
  {
    value: 'trust_queue',
    label: 'Trust Queue',
    description: 'Trust the queue record over filename parsing',
  },
  {
    value: 'trust_parse',
    label: 'Trust Parse',
    description: 'Trust filename parsing over queue record',
  },
  { value: 'fail', label: 'Fail with Warning', description: 'Fail import when conflict detected' },
]

export const UNKNOWN_MEDIA_OPTIONS = [
  { value: 'ignore', label: 'Ignore', description: "Skip files that don't match library items" },
  {
    value: 'auto_add',
    label: 'Auto Add',
    description: 'Automatically add to library and fetch metadata',
  },
]

export const COLON_REPLACEMENT_OPTIONS = [
  { value: 'delete', label: 'Delete', example: 'Title Subtitle' },
  { value: 'dash', label: 'Replace with Dash', example: 'Title- Subtitle' },
  { value: 'space_dash', label: 'Space Dash', example: 'Title - Subtitle' },
  { value: 'space_dash_space', label: 'Space Dash Space', example: 'Title - Subtitle' },
  { value: 'smart', label: 'Smart Replace', example: 'Context-aware replacement' },
  { value: 'custom', label: 'Custom', example: 'User-defined replacement' },
]

export const MULTI_EPISODE_STYLES = [
  { value: 'extend', label: 'Extend', example: 'S01E01-02-03' },
  { value: 'duplicate', label: 'Duplicate', example: 'S01E01.S01E02' },
  { value: 'repeat', label: 'Repeat', example: 'S01E01E02E03' },
  { value: 'scene', label: 'Scene', example: 'S01E01-E02-E03' },
  { value: 'range', label: 'Range', example: 'S01E01-03' },
  { value: 'prefixed_range', label: 'Prefixed Range', example: 'S01E01-E03' },
]

export const TOKEN_REFERENCE = {
  series: [
    { token: '{Series Title}', description: 'Full series title', example: "The Series Title's!" },
    {
      token: '{Series TitleYear}',
      description: 'Title with year',
      example: 'The Series Title (2024)',
    },
    {
      token: '{Series CleanTitle}',
      description: 'Title without special chars',
      example: 'The Series Titles',
    },
    {
      token: '{Series CleanTitleYear}',
      description: 'Clean title with year',
      example: 'The Series Titles 2024',
    },
  ],
  season: [
    { token: '{season:0}', description: 'Season number (no padding)', example: '1' },
    { token: '{season:00}', description: 'Season number (2-digit pad)', example: '01' },
  ],
  episode: [
    { token: '{episode:0}', description: 'Episode number (no padding)', example: '1' },
    { token: '{episode:00}', description: 'Episode number (2-digit pad)', example: '01' },
    { token: '{Episode Title}', description: 'Episode title', example: 'Episode Title' },
    {
      token: '{Episode CleanTitle}',
      description: 'Clean episode title',
      example: 'Episodes Title',
    },
  ],
  quality: [
    {
      token: '{Quality Full}',
      description: 'Quality with revision',
      example: 'WEBDL-1080p Proper',
    },
    { token: '{Quality Title}', description: 'Quality only', example: 'WEBDL-1080p' },
  ],
  mediaInfo: [
    { token: '{MediaInfo Simple}', description: 'Basic codec info', example: 'x264 DTS' },
    {
      token: '{MediaInfo Full}',
      description: 'Full codec info with languages',
      example: 'x264 DTS [EN]',
    },
    { token: '{MediaInfo VideoCodec}', description: 'Video codec', example: 'x264' },
    { token: '{MediaInfo VideoBitDepth}', description: 'Video bit depth', example: '10' },
    { token: '{MediaInfo VideoDynamicRange}', description: 'HDR indicator', example: 'HDR' },
    { token: '{MediaInfo VideoDynamicRangeType}', description: 'HDR type', example: 'DV HDR10' },
    { token: '{MediaInfo AudioCodec}', description: 'Audio codec', example: 'DTS' },
    { token: '{MediaInfo AudioChannels}', description: 'Audio channels', example: '5.1' },
    {
      token: '{MediaInfo AudioLanguages}',
      description: 'Audio language codes',
      example: '[EN+DE]',
    },
    {
      token: '{MediaInfo SubtitleLanguages}',
      description: 'Subtitle language codes',
      example: '[EN+ES]',
    },
  ],
  other: [
    { token: '{Air-Date}', description: 'Air date with dashes', example: '2024-03-20' },
    { token: '{Air Date}', description: 'Air date with spaces', example: '2024 03 20' },
    { token: '{Release Group}', description: 'Release group name', example: 'SPARKS' },
    { token: '{Revision}', description: 'Release revision', example: 'Proper' },
    { token: '{Custom Formats}', description: 'Matched custom formats', example: 'Remux HDR' },
    {
      token: '{Original Title}',
      description: 'Original release title',
      example: 'The.Series.S01E01',
    },
    {
      token: '{Original Filename}',
      description: 'Original filename',
      example: 'The.Series.S01E01.mkv',
    },
  ],
  movie: [
    { token: '{Movie Title}', description: 'Movie title', example: 'The Movie Title' },
    {
      token: '{Movie TitleYear}',
      description: 'Title with year',
      example: 'The Movie Title (2024)',
    },
    { token: '{Movie CleanTitle}', description: 'Clean movie title', example: 'The Movie Title' },
    {
      token: '{Movie CleanTitleYear}',
      description: 'Clean title with year',
      example: 'The Movie Title 2024',
    },
    { token: '{Year}', description: 'Release year', example: '2024' },
    { token: '{Edition Tags}', description: 'Edition info', example: 'Directors Cut' },
  ],
  anime: [
    { token: '{absolute:0}', description: 'Absolute episode (no padding)', example: '1' },
    { token: '{absolute:000}', description: 'Absolute episode (3-digit pad)', example: '001' },
    { token: '{version}', description: 'Release version', example: 'v2' },
  ],
}

export type TokenCategory = keyof typeof TOKEN_REFERENCE

export type TokenContext = 'episode' | 'movie' | 'series-folder' | 'season-folder' | 'movie-folder'

export const TOKEN_CATEGORIES_BY_CONTEXT: Record<TokenContext, TokenCategory[]> = {
  episode: ['series', 'season', 'episode', 'anime', 'quality', 'mediaInfo', 'other'],
  movie: ['movie', 'quality', 'mediaInfo', 'other'],
  'series-folder': ['series'],
  'season-folder': ['series', 'season'],
  'movie-folder': ['movie'],
}
