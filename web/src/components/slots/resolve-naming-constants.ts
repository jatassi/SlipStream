export const TOKEN_REFERENCE = {
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
    { token: '{MediaInfo Full}', description: 'Full codec info', example: 'x264 DTS [EN]' },
    { token: '{MediaInfo VideoCodec}', description: 'Video codec', example: 'x264' },
    { token: '{MediaInfo VideoDynamicRange}', description: 'HDR indicator', example: 'HDR' },
    { token: '{MediaInfo VideoDynamicRangeType}', description: 'HDR type', example: 'DV HDR10' },
    { token: '{MediaInfo AudioCodec}', description: 'Audio codec', example: 'DTS' },
    { token: '{MediaInfo AudioChannels}', description: 'Audio channels', example: '5.1' },
  ],
  episode: [
    { token: '{Series Title}', description: 'Series title', example: 'Breaking Bad' },
    { token: '{season:00}', description: 'Season number', example: '01' },
    { token: '{episode:00}', description: 'Episode number', example: '05' },
    { token: '{Episode Title}', description: 'Episode title', example: 'Pilot' },
  ],
  movie: [
    { token: '{Movie Title}', description: 'Movie title', example: 'The Matrix' },
    { token: '{Year}', description: 'Release year', example: '1999' },
  ],
} as const

export type TokenCategory = keyof typeof TOKEN_REFERENCE
