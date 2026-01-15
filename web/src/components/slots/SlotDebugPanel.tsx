import { useState } from 'react'
import { Bug, ChevronDown, ChevronUp, Play, Check, X, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { useParseRelease, useProfileMatch, useQualityProfiles } from '@/hooks'
import type {
  ParseReleaseOutput,
  ProfileMatchOutput,
  AttributeMatchResult,
} from '@/types'

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
                <Badge variant="outline" className="text-orange-500 border-orange-500">
                  Developer Mode
                </Badge>
              </div>
              {isOpen ? (
                <ChevronUp className="size-4 text-muted-foreground" />
              ) : (
                <ChevronDown className="size-4 text-muted-foreground" />
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
    if (!releaseTitle.trim()) return
    setError(null)
    try {
      const data = await parseReleaseMutation.mutateAsync({ releaseTitle: releaseTitle.trim() })
      setResult(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to parse release')
      setResult(null)
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h4 className="font-medium mb-2">Parse Release Title</h4>
        <p className="text-sm text-muted-foreground mb-3">
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
          <Play className="size-4 mr-2" />
          Parse
        </Button>
      </div>

      {error && (
        <div className="flex items-center gap-2 text-destructive text-sm">
          <AlertCircle className="size-4" />
          {error}
        </div>
      )}

      {result && (
        <div className="bg-muted/50 rounded-lg p-4 space-y-3">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="text-muted-foreground">Title:</span>
              <p className="font-medium">{result.title}</p>
            </div>
            {result.year && (
              <div>
                <span className="text-muted-foreground">Year:</span>
                <p className="font-medium">{result.year}</p>
              </div>
            )}
            {result.quality && (
              <div>
                <span className="text-muted-foreground">Resolution:</span>
                <p className="font-medium">{result.quality}</p>
              </div>
            )}
            {result.source && (
              <div>
                <span className="text-muted-foreground">Source:</span>
                <p className="font-medium">{result.source}</p>
              </div>
            )}
            {result.videoCodec && (
              <div>
                <span className="text-muted-foreground">Video Codec:</span>
                <p className="font-medium">{result.videoCodec}</p>
              </div>
            )}
            {result.hdrFormats && result.hdrFormats.length > 0 && (
              <div>
                <span className="text-muted-foreground">HDR:</span>
                <p className="font-medium">{result.hdrFormats.join(', ')}</p>
              </div>
            )}
            {result.audioCodecs && result.audioCodecs.length > 0 && (
              <div>
                <span className="text-muted-foreground">Audio:</span>
                <p className="font-medium">{result.audioCodecs.join(', ')}</p>
              </div>
            )}
            {result.audioChannels && result.audioChannels.length > 0 && (
              <div>
                <span className="text-muted-foreground">Channels:</span>
                <p className="font-medium">{result.audioChannels.join(', ')}</p>
              </div>
            )}
          </div>
          <div className="pt-2 border-t">
            <span className="text-muted-foreground text-sm">Quality Score: </span>
            <Badge variant="secondary">{result.qualityScore}</Badge>
          </div>
        </div>
      )}
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
    if (!releaseTitle.trim() || !selectedProfileId) return
    setError(null)
    try {
      const data = await profileMatchMutation.mutateAsync({
        releaseTitle: releaseTitle.trim(),
        qualityProfileId: parseInt(selectedProfileId, 10),
      })
      setResult(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to match profile')
      setResult(null)
    }
  }

  return (
    <div className="space-y-4 pt-4 border-t">
      <div>
        <h4 className="font-medium mb-2">Profile Matching Tester</h4>
        <p className="text-sm text-muted-foreground mb-3">
          Test whether a release matches a quality profile's attribute requirements.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <div className="md:col-span-2">
          <Label htmlFor="profile-match-input" className="sr-only">Release Title</Label>
          <Input
            id="profile-match-input"
            placeholder="e.g., Movie.2024.2160p.BluRay.DV.HDR10.x265.TrueHD.Atmos.7.1.mkv"
            value={releaseTitle}
            onChange={(e) => setReleaseTitle(e.target.value)}
          />
        </div>
        <div className="flex gap-2">
          <Select value={selectedProfileId} onValueChange={(v) => v && setSelectedProfileId(v)}>
            <SelectTrigger className="flex-1">
              {profiles?.find(p => p.id.toString() === selectedProfileId)?.name ?? 'Select profile...'}
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
            onClick={handleMatch}
            disabled={!releaseTitle.trim() || !selectedProfileId || profileMatchMutation.isPending}
          >
            <Play className="size-4 mr-2" />
            Test
          </Button>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 text-destructive text-sm">
          <AlertCircle className="size-4" />
          {error}
        </div>
      )}

      {result && (
        <div className="bg-muted/50 rounded-lg p-4 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="font-medium">Overall Result:</span>
              {result.allAttributesMatch ? (
                <Badge className="bg-green-500">
                  <Check className="size-3 mr-1" /> Matches
                </Badge>
              ) : (
                <Badge variant="destructive">
                  <X className="size-3 mr-1" /> Does Not Match
                </Badge>
              )}
            </div>
            <div className="flex gap-4 text-sm">
              <span>Quality: <Badge variant="secondary">{result.qualityScore}</Badge></span>
              <span>Attributes: <Badge variant="secondary">{result.totalScore}</Badge></span>
              <span>Combined: <Badge>{result.combinedScore}</Badge></span>
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            <AttributeMatchCard label="HDR" result={result.hdrMatch} />
            <AttributeMatchCard label="Video Codec" result={result.videoCodecMatch} />
            <AttributeMatchCard label="Audio Codec" result={result.audioCodecMatch} />
            <AttributeMatchCard label="Audio Channels" result={result.audioChannelMatch} />
          </div>
        </div>
      )}
    </div>
  )
}

function AttributeMatchCard({ label, result }: { label: string; result: AttributeMatchResult }) {
  const getModeColor = (mode: string) => {
    switch (mode) {
      case 'required':
        return 'text-red-500'
      case 'preferred':
        return 'text-blue-500'
      default:
        return 'text-muted-foreground'
    }
  }

  return (
    <div className="bg-background rounded-md p-3 border">
      <div className="flex items-center justify-between mb-2">
        <span className="font-medium text-sm">{label}</span>
        <div className="flex items-center gap-2">
          <Badge variant="outline" className={getModeColor(result.mode)}>
            {result.mode}
          </Badge>
          {result.matches ? (
            <Check className="size-4 text-green-500" />
          ) : (
            <X className="size-4 text-red-500" />
          )}
        </div>
      </div>
      <div className="text-xs space-y-1">
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
        {result.reason && (
          <div className="text-red-500">{result.reason}</div>
        )}
      </div>
    </div>
  )
}
