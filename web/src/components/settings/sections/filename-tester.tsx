import { useEffect, useState } from 'react'

import { Group } from '@/components/grouped-list'
import { StackedRow } from '@/components/settings/control-row'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { useParseFilename } from '@/hooks'
import { useDebounce } from '@/hooks/use-debounce'
import type { ParsedTokenDetail } from '@/types'

function ParseResult({ result }: { result: { parsedInfo?: { isTV: boolean; isSeasonPack?: boolean }; tokens: ParsedTokenDetail[] } }) {
  if (!result.parsedInfo) {
    return <p className="text-muted-foreground text-sm">Could not parse this filename</p>
  }

  return (
    <>
      <div className="flex items-center gap-2">
        <Badge variant={result.parsedInfo.isTV ? 'default' : 'secondary'}>
          {result.parsedInfo.isTV ? 'TV Show' : 'Movie'}
        </Badge>
        {Boolean(result.parsedInfo.isSeasonPack) && <Badge variant="outline">Season Pack</Badge>}
      </div>
      <div className="grid gap-2">
        {result.tokens.map((token: ParsedTokenDetail) => (
          <div key={`${token.name}-${token.value}`} className="flex items-center gap-3 text-sm">
            <span className="text-muted-foreground min-w-[80px]">{token.name}</span>
            <span className="bg-background rounded border px-2 py-0.5 font-mono">
              {token.value}
            </span>
          </div>
        ))}
      </div>
      {result.tokens.length === 0 && (
        <p className="text-muted-foreground text-sm">
          No metadata could be extracted from this filename
        </p>
      )}
    </>
  )
}

const PLACEHOLDERS = {
  tv: 'Breaking.Bad.S01E02.720p.BluRay.x264-DEMAND.mkv',
  movie: 'The.Matrix.1999.1080p.BluRay.x264-GROUP.mkv',
} as const

export function FilenameTester({
  mediaType,
  placeholder,
}: {
  mediaType: 'tv' | 'movie'
  placeholder?: string
}) {
  const [filename, setFilename] = useState('')
  const debouncedFilename = useDebounce(filename, 300)
  const parseMutation = useParseFilename()
  const parseMutate = parseMutation.mutate

  useEffect(() => {
    if (debouncedFilename.trim()) {
      parseMutate({ filename: debouncedFilename })
    }
  }, [debouncedFilename, parseMutate])

  const result = parseMutation.data
  const showResult = filename.trim().length > 0

  return (
    <Group header="Test Filename Parsing" footer="Paste a filename to see how it will be parsed.">
      <StackedRow>
        <Input
          aria-label="Test filename"
          value={filename}
          onChange={(e) => setFilename(e.target.value)}
          placeholder={placeholder ?? PLACEHOLDERS[mediaType]}
          className="h-11 font-mono text-base"
        />
      </StackedRow>
      {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
      {showResult && result !== undefined && <div className="space-y-3 px-4 py-3">
          <ParseResult result={result} />
        </div>}
      {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
      {parseMutation.isPending && <div className="text-footnote px-4 py-3 text-muted-foreground">Parsing...</div>}
    </Group>
  )
}
