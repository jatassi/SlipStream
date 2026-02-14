import { useEffect, useRef, useState } from 'react'

import { Code2, Loader2, Pencil } from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useImportSettings, usePreviewNamingPattern, useUpdateImportSettings } from '@/hooks'
import { useDebounce } from '@/hooks/useDebounce'
import type { ImportSettings, MissingTokenInfo } from '@/types'

type ResolveNamingModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  missingMovieTokens?: MissingTokenInfo[]
  missingEpisodeTokens?: MissingTokenInfo[]
  onResolved: () => void
}

const TOKEN_REFERENCE = {
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
}

export function ResolveNamingModal({
  open,
  onOpenChange,
  missingMovieTokens,
  missingEpisodeTokens,
  onResolved,
}: ResolveNamingModalProps) {
  const { data: settings } = useImportSettings()
  const updateMutation = useUpdateImportSettings()

  const [form, setForm] = useState<Partial<ImportSettings>>({})
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (open && settings) {
      setForm({
        standardEpisodeFormat: settings.standardEpisodeFormat,
        dailyEpisodeFormat: settings.dailyEpisodeFormat,
        animeEpisodeFormat: settings.animeEpisodeFormat,
        movieFileFormat: settings.movieFileFormat,
      })
    }
  }, [open, settings])

  const handleSave = async () => {
    if (!settings) {
      return
    }
    setSaving(true)
    try {
      await updateMutation.mutateAsync({
        ...settings,
        ...form,
      })
      toast.success('Naming formats updated')
      onOpenChange(false)
      onResolved()
    } catch {
      toast.error('Failed to update naming formats')
    } finally {
      setSaving(false)
    }
  }

  // Calculate which tokens are STILL missing for EACH format independently
  const requiredEpisodeTokens = (missingEpisodeTokens || []).map((t) => t.suggestedToken)
  const requiredMovieTokens = (missingMovieTokens || []).map((t) => t.suggestedToken)

  // Check each format independently
  const missingInStandard = requiredEpisodeTokens.filter(
    (token) => !(form.standardEpisodeFormat || '').includes(token),
  )
  const missingInDaily = requiredEpisodeTokens.filter(
    (token) => !(form.dailyEpisodeFormat || '').includes(token),
  )
  const missingInAnime = requiredEpisodeTokens.filter(
    (token) => !(form.animeEpisodeFormat || '').includes(token),
  )
  const missingInMovie = requiredMovieTokens.filter(
    (token) => !(form.movieFileFormat || '').includes(token),
  )

  // Tokens still missing anywhere (for highlighting Differentiator Tokens section)
  const stillMissingTokens = new Set<string>([
    ...missingInStandard,
    ...missingInDaily,
    ...missingInAnime,
    ...missingInMovie,
  ])

  // All issues resolved when ALL formats have their required tokens
  const allResolved =
    missingInStandard.length === 0 &&
    missingInDaily.length === 0 &&
    missingInAnime.length === 0 &&
    missingInMovie.length === 0

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Resolve Naming Format Issues</DialogTitle>
          <DialogDescription>
            Add the missing tokens to your filename formats to differentiate files in different
            slots.
            {stillMissingTokens.size > 0 && (
              <span className="mt-2 block font-medium text-orange-600 dark:text-orange-400">
                Suggested tokens to add: {[...stillMissingTokens].join(', ')}
              </span>
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-6 py-4 md:grid-cols-2">
          {/* Episode Formats */}
          <div className="space-y-4">
            <h3 className="border-b pb-2 text-sm font-medium">Episode Formats</h3>

            <PatternEditorCompact
              label="Standard Episode Format"
              value={form.standardEpisodeFormat || ''}
              onChange={(v) => setForm((prev) => ({ ...prev, standardEpisodeFormat: v }))}
              mediaType="episode"
              highlightTokens={missingInStandard}
            />

            <PatternEditorCompact
              label="Daily Episode Format"
              value={form.dailyEpisodeFormat || ''}
              onChange={(v) => setForm((prev) => ({ ...prev, dailyEpisodeFormat: v }))}
              mediaType="episode"
              highlightTokens={missingInDaily}
            />

            <PatternEditorCompact
              label="Anime Episode Format"
              value={form.animeEpisodeFormat || ''}
              onChange={(v) => setForm((prev) => ({ ...prev, animeEpisodeFormat: v }))}
              mediaType="episode"
              highlightTokens={missingInAnime}
            />
          </div>

          {/* Movie Format */}
          <div className="space-y-4">
            <h3 className="border-b pb-2 text-sm font-medium">Movie Format</h3>

            <PatternEditorCompact
              label="Movie File Format"
              value={form.movieFileFormat || ''}
              onChange={(v) => setForm((prev) => ({ ...prev, movieFileFormat: v }))}
              mediaType="movie"
              highlightTokens={missingInMovie}
            />

            {/* Token Reference */}
            <div className="mt-6 space-y-3">
              <h4 className="text-muted-foreground text-xs font-medium">Differentiator Tokens</h4>
              <div className="grid gap-2">
                {[...TOKEN_REFERENCE.quality, ...TOKEN_REFERENCE.mediaInfo.slice(0, 5)].map((t) => (
                  <div
                    key={t.token}
                    className={`flex items-center justify-between rounded border p-2 text-xs ${
                      stillMissingTokens.has(t.token)
                        ? 'border-orange-400 bg-orange-50 dark:bg-orange-950/30'
                        : 'bg-muted/30'
                    }`}
                  >
                    <code className="font-mono">{t.token}</code>
                    <span className="text-muted-foreground">{t.example}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving || !allResolved}>
            {saving ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type PatternEditorCompactProps = {
  label: string
  value: string
  onChange: (value: string) => void
  mediaType: 'episode' | 'movie'
  highlightTokens?: string[]
}

function PatternEditorCompact({
  label,
  value,
  onChange,
  mediaType,
  highlightTokens = [],
}: PatternEditorCompactProps) {
  const [localValue, setLocalValue] = useState(value)
  const [tokenDialogOpen, setTokenDialogOpen] = useState(false)
  const debouncedValue = useDebounce(localValue, 500)
  const previewMutation = usePreviewNamingPattern()
  const previewMutate = previewMutation.mutate

  useEffect(() => {
    setLocalValue(value)
  }, [value])

  useEffect(() => {
    if (debouncedValue) {
      previewMutate({ pattern: debouncedValue, mediaType })
    }
  }, [debouncedValue, mediaType, previewMutate])

  const handleChange = (newValue: string) => {
    setLocalValue(newValue)
    onChange(newValue)
  }

  const preview = previewMutation.data

  // Check if pattern is missing suggested tokens
  const isMissingTokens = highlightTokens.some((token) => !localValue.includes(token))

  return (
    <div className="space-y-2">
      <Label className="text-xs">{label}</Label>
      <button
        type="button"
        onClick={() => setTokenDialogOpen(true)}
        className={`bg-muted/50 hover:bg-muted flex w-full cursor-pointer items-start gap-2 rounded-md border p-2 text-left font-mono text-xs transition-colors ${
          isMissingTokens ? 'border-orange-400' : ''
        }`}
      >
        <Pencil className="text-muted-foreground mt-0.5 size-3 shrink-0" />
        <span className="break-all">{localValue || '(not configured)'}</span>
      </button>
      {preview ? (
        <div className="bg-muted/50 rounded-md p-2 text-xs">
          <span className="text-muted-foreground">Preview: </span>
          {preview.valid ? (
            <span className="font-mono break-all text-green-600 dark:text-green-400">
              {preview.preview}
            </span>
          ) : (
            <span className="text-red-600 dark:text-red-400">{preview.error}</span>
          )}
        </div>
      ) : null}
      <TokenBuilderDialogCompact
        open={tokenDialogOpen}
        onOpenChange={setTokenDialogOpen}
        value={localValue}
        onChange={handleChange}
        mediaType={mediaType}
        highlightTokens={highlightTokens}
      />
    </div>
  )
}

type TokenBuilderDialogCompactProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  value: string
  onChange: (value: string) => void
  mediaType: 'episode' | 'movie'
  highlightTokens: string[]
}

function TokenBuilderDialogCompact({
  open,
  onOpenChange,
  value,
  onChange,
  mediaType,
  highlightTokens,
}: TokenBuilderDialogCompactProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [localValue, setLocalValue] = useState(value)
  const [cursorPosition, setCursorPosition] = useState<number | null>(null)
  const [prevOpen, setPrevOpen] = useState(open)
  const [prevValue, setPrevValue] = useState(value)

  // Reset state when dialog opens or value changes (React-recommended pattern)
  if (open !== prevOpen || value !== prevValue) {
    setPrevOpen(open)
    setPrevValue(value)
    if (open) {
      setLocalValue(value)
      setCursorPosition(null)
    }
  }

  const handleInsertToken = (token: string) => {
    const textarea = textareaRef.current
    if (!textarea) {
      setLocalValue((prev) => prev + token)
      return
    }

    const start = cursorPosition ?? textarea.selectionStart
    const newValue = localValue.slice(0, start) + token + localValue.slice(start)
    setLocalValue(newValue)

    const newPosition = start + token.length
    setCursorPosition(newPosition)

    setTimeout(() => {
      textarea.focus()
      textarea.setSelectionRange(newPosition, newPosition)
    }, 0)
  }

  const handleApply = () => {
    onChange(localValue)
    onOpenChange(false)
  }

  const categories =
    mediaType === 'episode'
      ? (['episode', 'quality', 'mediaInfo'] as const)
      : (['movie', 'quality', 'mediaInfo'] as const)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[70vh] flex-col sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Edit Pattern</DialogTitle>
          <DialogDescription>
            Click a token to insert it. Highlighted tokens are recommended.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 space-y-3 overflow-y-auto py-2">
          {categories.map((category) => {
            // Only highlight tokens in differentiator categories (quality, mediaInfo)
            const canHighlight = category === 'quality' || category === 'mediaInfo'
            return (
              <div key={category} className="space-y-1.5">
                <h4 className="text-muted-foreground text-xs font-medium capitalize">
                  {category.replaceAll(/([A-Z])/g, ' $1').trim()}
                </h4>
                <div className="flex flex-wrap gap-1.5">
                  {TOKEN_REFERENCE[category].map((t) => (
                    <button
                      key={t.token}
                      type="button"
                      onClick={() => handleInsertToken(t.token)}
                      className={`inline-flex cursor-pointer items-center gap-1 rounded border px-2 py-1 font-mono text-[10px] transition-colors ${
                        canHighlight && highlightTokens.includes(t.token)
                          ? 'border-orange-400 bg-orange-100 hover:bg-orange-200 dark:bg-orange-950 dark:hover:bg-orange-900'
                          : 'bg-muted hover:bg-muted/80'
                      }`}
                      title={`${t.description}\nExample: ${t.example}`}
                    >
                      <Code2 className="text-muted-foreground size-2.5" />
                      {t.token}
                    </button>
                  ))}
                </div>
              </div>
            )
          })}
        </div>

        <div className="space-y-2 border-t pt-2">
          <Label className="text-xs">Format Pattern</Label>
          <Textarea
            ref={textareaRef}
            value={localValue}
            onChange={(e) => setLocalValue(e.target.value)}
            onSelect={(e) => setCursorPosition(e.currentTarget.selectionStart)}
            onClick={(e) => setCursorPosition(e.currentTarget.selectionStart)}
            onKeyUp={(e) => setCursorPosition(e.currentTarget.selectionStart)}
            className="min-h-[60px] font-mono text-xs"
            placeholder="Click tokens above to build your format pattern..."
          />
        </div>

        <DialogFooter showCloseButton>
          <Button onClick={handleApply}>Apply</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
