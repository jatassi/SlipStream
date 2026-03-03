import { useState } from 'react'

import { AlertCircle, Play } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useParseRelease } from '@/hooks'
import type { ParseReleaseOutput } from '@/types'

export function ParseReleaseTester() {
  const [releaseTitle, setReleaseTitle] = useState('')
  const [result, setResult] = useState<ParseReleaseOutput | null>(null)
  const [parseError, setParseError] = useState<string | null>(null)

  const parseReleaseMutation = useParseRelease()

  const handleParse = async () => {
    if (!releaseTitle.trim()) {
      return
    }
    setParseError(null)
    try {
      const data = await parseReleaseMutation.mutateAsync({ releaseTitle: releaseTitle.trim() })
      setResult(data)
    } catch (error) {
      setParseError(error instanceof Error ? error.message : 'Failed to parse release')
      setResult(null)
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h4 className="mb-2 font-medium">Parse Release Title</h4>
        <p className="text-muted-foreground mb-3 text-sm">
          See how SlipStream parses a release title to extract quality attributes.
        </p>
      </div>

      <div className="flex gap-2">
        <Input
          placeholder="e.g., Movie.2024.2160p.BluRay.DV.HDR10.x265.TrueHD.Atmos.7.1.mkv"
          value={releaseTitle}
          onChange={(e) => setReleaseTitle(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleParse()}
        />
        <Button
          onClick={handleParse}
          disabled={!releaseTitle.trim() || parseReleaseMutation.isPending}
        >
          <Play className="mr-2 size-4" />
          Parse
        </Button>
      </div>

      <ParseErrorMessage message={parseError} />
      {result ? <ParseResultDisplay result={result} /> : null}
    </div>
  )
}

function ParseErrorMessage({ message }: { message: string | null }) {
  if (!message) {
    return null
  }
  return (
    <div className="text-destructive flex items-center gap-2 text-sm">
      <AlertCircle className="size-4" />
      {message}
    </div>
  )
}

function buildParseFields(result: ParseReleaseOutput) {
  return [
    { label: 'Title', value: result.title },
    { label: 'Year', value: result.year?.toString() },
    { label: 'Resolution', value: result.quality },
    { label: 'Source', value: result.source },
    { label: 'Video Codec', value: result.videoCodec },
    { label: 'HDR', value: result.hdrFormats?.join(', ') },
    { label: 'Audio', value: result.audioCodecs?.join(', ') },
    { label: 'Channels', value: result.audioChannels?.join(', ') },
  ].filter((f) => f.value)
}

function ParseResultDisplay({ result }: { result: ParseReleaseOutput }) {
  const fields = buildParseFields(result)

  return (
    <div className="bg-muted/50 space-y-3 rounded-lg p-4">
      <div className="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
        {fields.map((field) => (
          <div key={field.label}>
            <span className="text-muted-foreground">{field.label}:</span>
            <p className="font-medium">{field.value}</p>
          </div>
        ))}
      </div>
      <div className="border-t pt-2">
        <span className="text-muted-foreground text-sm">Quality Score: </span>
        <Badge variant="secondary">{result.qualityScore}</Badge>
      </div>
    </div>
  )
}
