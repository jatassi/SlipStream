import { useState } from 'react'

import { AlertCircle, Bug, Check, ChevronDown, ChevronUp, Play, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { useParseRelease, useProfileMatch, useQualityProfiles } from '@/hooks'
import type { AttributeMatchResult, ParseReleaseOutput, ProfileMatchOutput } from '@/types'

export function SlotDebugPanel() {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <Card className="border-dashed border-orange-500/50 bg-orange-500/5">
        <CollapsibleTrigger>
          <CardHeader className="cursor-pointer">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Bug className="size-5 text-orange-500" />
                <CardTitle className="text-orange-500">Debug Tools</CardTitle>
                <Badge variant="outline" className="border-orange-500 text-orange-500">
                  Developer Mode
                </Badge>
              </div>
              {isOpen ? (
                <ChevronUp className="text-muted-foreground size-4" />
              ) : (
                <ChevronDown className="text-muted-foreground size-4" />
              )}
            </div>
            <CardDescription>
              Test release parsing and profile matching without affecting your library
            </CardDescription>
          </CardHeader>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <CardContent className="space-y-6">
            <ParseReleaseTester />
            <ProfileMatchTester />
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  )
}

function ParseReleaseTester() {
  const [releaseTitle, setReleaseTitle] = useState('')
  const [result, setResult] = useState<ParseReleaseOutput | null>(null)
  const [error, setError] = useState<string | null>(null)

  const parseReleaseMutation = useParseRelease()

  const handleParse = async () => {
    if (!releaseTitle.trim()) {
      return
    }
    setError(null)
    try {
      const data = await parseReleaseMutation.mutateAsync({ releaseTitle: releaseTitle.trim() })
      setResult(data)
    } catch (error_) {
      setError(error_ instanceof Error ? error_.message : 'Failed to parse release')
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

      <ErrorMessage message={error} />
      {result ? <ParseResultDisplay result={result} /> : null}
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

function ProfileMatchTester() {
  const [releaseTitle, setReleaseTitle] = useState('')
  const [selectedProfileId, setSelectedProfileId] = useState<string>('')
  const [result, setResult] = useState<ProfileMatchOutput | null>(null)
  const [error, setError] = useState<string | null>(null)

  const { data: profiles } = useQualityProfiles()
  const profileMatchMutation = useProfileMatch()

  const handleMatch = async () => {
    if (!releaseTitle.trim() || !selectedProfileId) {
      return
    }
    setError(null)
    try {
      const data = await profileMatchMutation.mutateAsync({
        releaseTitle: releaseTitle.trim(),
        qualityProfileId: Number.parseInt(selectedProfileId, 10),
      })
      setResult(data)
    } catch (error_) {
      setError(error_ instanceof Error ? error_.message : 'Failed to match profile')
      setResult(null)
    }
  }

  return (
    <div className="space-y-4 border-t pt-4">
      <div>
        <h4 className="mb-2 font-medium">Profile Matching Tester</h4>
        <p className="text-muted-foreground mb-3 text-sm">
          Test whether a release matches a quality profile&apos;s attribute requirements.
        </p>
      </div>

      <ProfileMatchForm
        releaseTitle={releaseTitle}
        onReleaseTitleChange={setReleaseTitle}
        selectedProfileId={selectedProfileId}
        onProfileChange={setSelectedProfileId}
        profiles={profiles}
        onSubmit={handleMatch}
        isPending={profileMatchMutation.isPending}
      />

      <ErrorMessage message={error} />
      {result ? <ProfileMatchResultDisplay result={result} /> : null}
    </div>
  )
}

type ProfileMatchFormProps = {
  releaseTitle: string
  onReleaseTitleChange: (value: string) => void
  selectedProfileId: string
  onProfileChange: (value: string) => void
  profiles: { id: number; name: string }[] | undefined
  onSubmit: () => void
  isPending: boolean
}

function ProfileMatchForm(props: ProfileMatchFormProps) {
  const { releaseTitle, selectedProfileId, profiles, isPending } = props

  return (
    <div className="grid gap-4 md:grid-cols-3">
      <div className="md:col-span-2">
        <Label htmlFor="profile-match-input" className="sr-only">
          Release Title
        </Label>
        <Input
          id="profile-match-input"
          placeholder="e.g., Movie.2024.2160p.BluRay.DV.HDR10.x265.TrueHD.Atmos.7.1.mkv"
          value={releaseTitle}
          onChange={(e) => props.onReleaseTitleChange(e.target.value)}
        />
      </div>
      <div className="flex gap-2">
        <Select value={selectedProfileId} onValueChange={(v) => v && props.onProfileChange(v)}>
          <SelectTrigger className="flex-1">
            {profiles?.find((p) => p.id.toString() === selectedProfileId)?.name ??
              'Select profile...'}
          </SelectTrigger>
          <SelectContent>
            {profiles?.map((profile) => (
              <SelectItem key={profile.id} value={profile.id.toString()}>
                {profile.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          onClick={props.onSubmit}
          disabled={!releaseTitle.trim() || !selectedProfileId || isPending}
        >
          <Play className="mr-2 size-4" />
          Test
        </Button>
      </div>
    </div>
  )
}

function ProfileMatchResultDisplay({ result }: { result: ProfileMatchOutput }) {
  return (
    <div className="bg-muted/50 space-y-4 rounded-lg p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="font-medium">Overall Result:</span>
          {result.allAttributesMatch ? (
            <Badge className="bg-green-500">
              <Check className="mr-1 size-3" /> Matches
            </Badge>
          ) : (
            <Badge variant="destructive">
              <X className="mr-1 size-3" /> Does Not Match
            </Badge>
          )}
        </div>
        <div className="flex gap-4 text-sm">
          <span>
            Quality: <Badge variant="secondary">{result.qualityScore}</Badge>
          </span>
          <span>
            Attributes: <Badge variant="secondary">{result.totalScore}</Badge>
          </span>
          <span>
            Combined: <Badge>{result.combinedScore}</Badge>
          </span>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <AttributeMatchCard label="HDR" result={result.hdrMatch} />
        <AttributeMatchCard label="Video Codec" result={result.videoCodecMatch} />
        <AttributeMatchCard label="Audio Codec" result={result.audioCodecMatch} />
        <AttributeMatchCard label="Audio Channels" result={result.audioChannelMatch} />
      </div>
    </div>
  )
}

function ErrorMessage({ message }: { message: string | null }) {
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

const MODE_COLORS: Record<string, string> = {
  required: 'text-red-500',
  preferred: 'text-blue-500',
}

function AttributeMatchCard({ label, result }: { label: string; result: AttributeMatchResult }) {
  return (
    <div className="bg-background rounded-md border p-3">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-sm font-medium">{label}</span>
        <div className="flex items-center gap-2">
          <Badge variant="outline" className={MODE_COLORS[result.mode] ?? 'text-muted-foreground'}>
            {result.mode}
          </Badge>
          {result.matches ? (
            <Check className="size-4 text-green-500" />
          ) : (
            <X className="size-4 text-red-500" />
          )}
        </div>
      </div>
      <div className="space-y-1 text-xs">
        {result.profileValues.length > 0 && (
          <div>
            <span className="text-muted-foreground">Profile: </span>
            <span>{result.profileValues.join(', ')}</span>
          </div>
        )}
        <div>
          <span className="text-muted-foreground">Release: </span>
          <span>{result.releaseValue || '(empty)'}</span>
        </div>
        {result.score > 0 && (
          <div>
            <span className="text-muted-foreground">Score bonus: </span>
            <span className="text-green-500">+{result.score}</span>
          </div>
        )}
        {result.reason ? <div className="text-red-500">{result.reason}</div> : null}
      </div>
    </div>
  )
}
