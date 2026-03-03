import { useState } from 'react'

import { AlertCircle, Check, Play, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { useProfileMatch, useQualityProfiles } from '@/hooks'
import type { AttributeMatchResult, ProfileMatchOutput } from '@/types'

export function ProfileMatchTester() {
  const [releaseTitle, setReleaseTitle] = useState('')
  const [selectedProfileId, setSelectedProfileId] = useState<string>('')
  const [result, setResult] = useState<ProfileMatchOutput | null>(null)
  const [matchError, setMatchError] = useState<string | null>(null)

  const { data: profiles } = useQualityProfiles()
  const profileMatchMutation = useProfileMatch()

  const handleMatch = async () => {
    if (!releaseTitle.trim() || !selectedProfileId) {
      return
    }
    setMatchError(null)
    try {
      const data = await profileMatchMutation.mutateAsync({
        releaseTitle: releaseTitle.trim(),
        qualityProfileId: Number.parseInt(selectedProfileId, 10),
      })
      setResult(data)
    } catch (error) {
      setMatchError(error instanceof Error ? error.message : 'Failed to match profile')
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

      <ProfileErrorMessage message={matchError} />
      {result ? <ProfileMatchResultDisplay result={result} /> : null}
    </div>
  )
}

function ProfileErrorMessage({ message }: { message: string | null }) {
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
