import type { ScannedFile } from '@/types'

export function matchLabel(file: ScannedFile | undefined, matching: boolean): string {
  if (file === undefined) {
    return matching ? 'Matching…' : 'Not importable'
  }
  const match = file.suggestedMatch
  if (!match) {
    return 'No library match'
  }
  if (match.mediaType === 'episode') {
    const season = String(match.seasonNum ?? 0).padStart(2, '0')
    const episode = String(match.episodeNum ?? 0).padStart(2, '0')
    return `${match.seriesTitle ?? 'Unknown series'} · S${season}E${episode}`
  }
  if (match.year === undefined) {
    return match.mediaTitle
  }
  return `${match.mediaTitle} (${match.year.toString()})`
}

export function importableFile(file: ScannedFile | undefined): ScannedFile | undefined {
  if (file?.suggestedMatch === undefined) {
    return undefined
  }
  return file
}
