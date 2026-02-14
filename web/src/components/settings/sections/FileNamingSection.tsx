import { useCallback, useEffect, useRef, useState } from 'react'

import {
  AlertTriangle,
  CheckCircle2,
  Code2,
  ExternalLink,
  Pencil,
  Plus,
  Save,
  X,
} from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/ErrorState'
import { LoadingState } from '@/components/data/LoadingState'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  useImportSettings,
  useMediainfoAvailable,
  useParseFilename,
  usePreviewNamingPattern,
  useUpdateImportSettings,
} from '@/hooks'
import { useDebounce } from '@/hooks/useDebounce'
import type { ImportSettings, ParsedTokenDetail, TokenBreakdown } from '@/types'

const VALIDATION_LEVELS = [
  { value: 'basic', label: 'Basic', description: 'File exists and size > 0' },
  { value: 'standard', label: 'Standard', description: 'Size > minimum, valid extension' },
  { value: 'full', label: 'Full', description: 'Size + extension + MediaInfo probe' },
]

const MATCH_CONFLICT_OPTIONS = [
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

const UNKNOWN_MEDIA_OPTIONS = [
  { value: 'ignore', label: 'Ignore', description: "Skip files that don't match library items" },
  {
    value: 'auto_add',
    label: 'Auto Add',
    description: 'Automatically add to library and fetch metadata',
  },
]

const COLON_REPLACEMENT_OPTIONS = [
  { value: 'delete', label: 'Delete', example: 'Title Subtitle' },
  { value: 'dash', label: 'Replace with Dash', example: 'Title- Subtitle' },
  { value: 'space_dash', label: 'Space Dash', example: 'Title - Subtitle' },
  { value: 'space_dash_space', label: 'Space Dash Space', example: 'Title - Subtitle' },
  { value: 'smart', label: 'Smart Replace', example: 'Context-aware replacement' },
  { value: 'custom', label: 'Custom', example: 'User-defined replacement' },
]

const MULTI_EPISODE_STYLES = [
  { value: 'extend', label: 'Extend', example: 'S01E01-02-03' },
  { value: 'duplicate', label: 'Duplicate', example: 'S01E01.S01E02' },
  { value: 'repeat', label: 'Repeat', example: 'S01E01E02E03' },
  { value: 'scene', label: 'Scene', example: 'S01E01-E02-E03' },
  { value: 'range', label: 'Range', example: 'S01E01-03' },
  { value: 'prefixed_range', label: 'Prefixed Range', example: 'S01E01-E03' },
]

const TOKEN_REFERENCE = {
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

type TokenCategory = keyof typeof TOKEN_REFERENCE
type TokenContext = 'episode' | 'movie' | 'series-folder' | 'season-folder' | 'movie-folder'

const TOKEN_CATEGORIES_BY_CONTEXT: Record<TokenContext, TokenCategory[]> = {
  episode: ['series', 'season', 'episode', 'anime', 'quality', 'mediaInfo', 'other'],
  movie: ['movie', 'quality', 'mediaInfo', 'other'],
  'series-folder': ['series'],
  'season-folder': ['series', 'season'],
  'movie-folder': ['movie'],
}

function TokenBuilderDialog({
  open,
  onOpenChange,
  value,
  onChange,
  tokenContext,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  value: string
  onChange: (value: string) => void
  tokenContext: TokenContext
}) {
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

  const handleTextareaChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setLocalValue(e.target.value)
  }

  const handleTextareaSelect = (e: React.SyntheticEvent<HTMLTextAreaElement>) => {
    setCursorPosition(e.currentTarget.selectionStart)
  }

  const handleApply = () => {
    onChange(localValue)
    onOpenChange(false)
  }

  const categories = TOKEN_CATEGORIES_BY_CONTEXT[tokenContext]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[80vh] flex-col sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Token Builder</DialogTitle>
          <DialogDescription>Click a token to insert it into your format pattern</DialogDescription>
        </DialogHeader>

        <div className="flex-1 space-y-4 overflow-y-auto py-2">
          {categories.map((category) => (
            <div key={category} className="space-y-2">
              <h4 className="text-muted-foreground text-sm font-medium capitalize">
                {category.replaceAll(/([A-Z])/g, ' $1').trim()}
              </h4>
              <div className="flex flex-wrap gap-2">
                {TOKEN_REFERENCE[category].map((t) => (
                  <button
                    key={t.token}
                    type="button"
                    onClick={() => handleInsertToken(t.token)}
                    className="bg-muted hover:bg-muted/80 inline-flex cursor-pointer items-center gap-1.5 rounded-md border px-2.5 py-1.5 font-mono text-xs transition-colors"
                    title={`${t.description}\nExample: ${t.example}`}
                  >
                    <Code2 className="text-muted-foreground size-3" />
                    {t.token}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div className="space-y-2 border-t pt-2">
          <Label>Format Pattern</Label>
          <Textarea
            ref={textareaRef}
            value={localValue}
            onChange={handleTextareaChange}
            onSelect={handleTextareaSelect}
            onClick={handleTextareaSelect}
            onKeyUp={handleTextareaSelect}
            className="min-h-[80px] font-mono text-sm"
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

function PatternEditor({
  label,
  value,
  onChange,
  description,
  mediaType = 'episode',
  tokenContext,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  description?: string
  mediaType?: 'episode' | 'movie' | 'folder'
  tokenContext: TokenContext
}) {
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

  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <button
        type="button"
        onClick={() => setTokenDialogOpen(true)}
        className="bg-muted/50 hover:bg-muted flex w-full cursor-pointer items-start gap-3 rounded-md border p-3 text-left font-mono text-sm transition-colors"
      >
        <Pencil className="text-muted-foreground mt-0.5 size-4 shrink-0" />
        <span className="break-all">{localValue || '(not configured)'}</span>
      </button>
      {description ? <p className="text-muted-foreground text-xs">{description}</p> : null}
      {preview ? (
        <div className="bg-muted/50 space-y-2 rounded-md p-3">
          <div className="flex items-start gap-2">
            <span className="shrink-0 text-xs font-medium">Preview:</span>
            {preview.valid ? (
              <span className="font-mono text-sm break-all text-green-600 dark:text-green-400">
                {preview.preview}
              </span>
            ) : (
              <span className="text-sm text-red-600 dark:text-red-400">{preview.error}</span>
            )}
          </div>
          {preview.tokens && preview.tokens.length > 0 ? (
            <details className="text-xs">
              <summary className="text-muted-foreground hover:text-foreground cursor-pointer">
                Token breakdown
              </summary>
              <div className="mt-2 space-y-1">
                {preview.tokens.map((t: TokenBreakdown) => (
                  <div key={`${t.token}-${t.value}`} className="flex items-center gap-2 font-mono">
                    <span className="text-muted-foreground">{t.token}</span>
                    <span className="text-muted-foreground">→</span>
                    <span className={t.empty ? 'text-yellow-600' : ''}>{t.value || '(empty)'}</span>
                  </div>
                ))}
              </div>
            </details>
          ) : null}
        </div>
      ) : null}
      <TokenBuilderDialog
        open={tokenDialogOpen}
        onOpenChange={setTokenDialogOpen}
        value={localValue}
        onChange={handleChange}
        tokenContext={tokenContext}
      />
    </div>
  )
}

function ExtensionManager({
  extensions,
  onChange,
}: {
  extensions: string[]
  onChange: (extensions: string[]) => void
}) {
  const [newExt, setNewExt] = useState('')

  const addExtension = () => {
    if (!newExt) {
      return
    }
    let ext = newExt.trim().toLowerCase()
    if (!ext.startsWith('.')) {
      ext = `.${ext}`
    }
    if (!extensions.includes(ext)) {
      onChange([...extensions, ext])
    }
    setNewExt('')
  }

  const removeExtension = (ext: string) => {
    onChange(extensions.filter((e) => e !== ext))
  }

  return (
    <div className="space-y-3">
      <Label>Allowed Video Extensions</Label>
      <div className="flex flex-wrap gap-2">
        {extensions.map((ext) => (
          <Badge key={ext} variant="secondary" className="gap-1">
            {ext}
            <button
              type="button"
              onClick={() => removeExtension(ext)}
              className="hover:text-destructive ml-1"
            >
              <X className="size-3" />
            </button>
          </Badge>
        ))}
      </div>
      <div className="flex gap-2">
        <Input
          value={newExt}
          onChange={(e) => setNewExt(e.target.value)}
          placeholder=".mkv"
          className="w-24"
          onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addExtension())}
        />
        <Button type="button" size="sm" variant="outline" onClick={addExtension}>
          <Plus className="mr-1 size-4" />
          Add
        </Button>
      </div>
    </div>
  )
}

function FilenameTester({
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
  const showResult = filename.trim() && result

  const defaultPlaceholder =
    mediaType === 'tv'
      ? 'Breaking.Bad.S01E02.720p.BluRay.x264-DEMAND.mkv'
      : 'The.Matrix.1999.1080p.BluRay.x264-GROUP.mkv'

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Test Filename Parsing</CardTitle>
        <CardDescription>Paste a filename to see how it will be parsed</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Input
          value={filename}
          onChange={(e) => setFilename(e.target.value)}
          placeholder={placeholder || defaultPlaceholder}
          className="font-mono text-sm"
        />
        {showResult ? (
          <div className="bg-muted/30 space-y-3 rounded-md border p-4">
            {result.parsedInfo ? (
              <>
                <div className="flex items-center gap-2">
                  <Badge variant={result.parsedInfo.isTV ? 'default' : 'secondary'}>
                    {result.parsedInfo.isTV ? 'TV Show' : 'Movie'}
                  </Badge>
                  {result.parsedInfo.isSeasonPack ? (
                    <Badge variant="outline">Season Pack</Badge>
                  ) : null}
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
            ) : (
              <p className="text-muted-foreground text-sm">Could not parse this filename</p>
            )}
          </div>
        ) : null}
        {parseMutation.isPending ? (
          <p className="text-muted-foreground text-sm">Parsing...</p>
        ) : null}
      </CardContent>
    </Card>
  )
}

function MediaInfoStatus() {
  const isAvailable = useMediainfoAvailable()

  if (isAvailable) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-green-500/20 bg-green-500/10 p-3">
        <CheckCircle2 className="size-4 shrink-0 text-green-500" />
        <span className="text-sm">MediaInfo is installed and available for file probing</span>
      </div>
    )
  }

  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border border-amber-500/20 bg-amber-500/10 p-3">
      <div className="flex items-center gap-2">
        <AlertTriangle className="size-4 shrink-0 text-amber-500" />
        <span className="text-sm">
          MediaInfo not found - file probing will use filename parsing only
        </span>
      </div>
      <a
        href="https://mediaarea.net/en/MediaInfo/Download"
        target="_blank"
        rel="noopener noreferrer"
        className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex shrink-0 items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
      >
        Download
        <ExternalLink className="size-3" />
      </a>
    </div>
  )
}

export function FileNamingSection() {
  const { data: settings, isLoading, isError, refetch } = useImportSettings()
  const updateMutation = useUpdateImportSettings()

  const [form, setForm] = useState<ImportSettings | null>(null)
  const [activeTab, setActiveTab] = useState('validation')
  const [prevSettings, setPrevSettings] = useState(settings)

  // Sync form state when settings change (React-recommended pattern)
  if (settings !== prevSettings) {
    setPrevSettings(settings)
    if (settings) {
      setForm({ ...settings })
    }
  }

  const updateField = useCallback(
    <K extends keyof ImportSettings>(field: K, value: ImportSettings[K]) => {
      setForm((prev) => (prev ? { ...prev, [field]: value } : null))
    },
    [],
  )

  const hasChanges = form && settings && JSON.stringify(form) !== JSON.stringify(settings)

  // Auto-save with debounce
  const debouncedForm = useDebounce(form, 1000)
  const lastSavedRef = useRef<string | null>(null)

  useEffect(() => {
    if (!debouncedForm || !settings) {
      return
    }
    const formJson = JSON.stringify(debouncedForm)
    const settingsJson = JSON.stringify(settings)
    // Only save if form has changed from settings AND we haven't already saved this exact form
    if (formJson !== settingsJson && formJson !== lastSavedRef.current) {
      lastSavedRef.current = formJson
      updateMutation.mutate(debouncedForm, {
        onError: () => {
          toast.error('Failed to auto-save settings')
          lastSavedRef.current = null
        },
      })
    }
  }, [debouncedForm, settings, updateMutation])

  if (isLoading) {
    return <LoadingState variant="list" count={3} />
  }

  if (isError || !form) {
    return <ErrorState onRetry={refetch} />
  }

  return (
    <div className="space-y-6">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <div className="flex items-center justify-between gap-4">
          <TabsList>
            <TabsTrigger value="validation">Validation</TabsTrigger>
            <TabsTrigger value="matching">Matching</TabsTrigger>
            <TabsTrigger value="tv-naming">TV Naming</TabsTrigger>
            <TabsTrigger value="movie-naming">Movie Naming</TabsTrigger>
            <TabsTrigger value="tokens">Token Reference</TabsTrigger>
          </TabsList>
          {updateMutation.isPending ? (
            <span className="text-muted-foreground flex items-center gap-2 text-sm">
              <Save className="size-4 animate-pulse" />
              Saving...
            </span>
          ) : hasChanges ? (
            <span className="text-muted-foreground flex items-center gap-2 text-sm">
              <Save className="size-4" />
              Unsaved changes
            </span>
          ) : (
            <span className="text-muted-foreground flex items-center gap-2 text-sm">
              <Save className="size-4" />
              All changes saved
            </span>
          )}
        </div>

        <TabsContent value="validation" className="mt-6 max-w-2xl space-y-6">
          <MediaInfoStatus />

          <Card>
            <CardHeader>
              <CardTitle>File Validation</CardTitle>
              <CardDescription>Configure how files are validated before import</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-3">
                <Label>Validation Level</Label>
                <Select
                  value={form.validationLevel}
                  onValueChange={(v) => v && updateField('validationLevel', v)}
                >
                  <SelectTrigger>
                    {VALIDATION_LEVELS.find((l) => l.value === form.validationLevel)?.label}
                  </SelectTrigger>
                  <SelectContent>
                    {VALIDATION_LEVELS.map((level) => (
                      <SelectItem key={level.value} value={level.value}>
                        {level.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-muted-foreground text-xs">
                  {VALIDATION_LEVELS.find((l) => l.value === form.validationLevel)?.description}
                </p>
              </div>

              <div className="space-y-3">
                <div className="flex justify-between">
                  <Label>Minimum File Size</Label>
                  <span className="text-muted-foreground text-sm">{form.minimumFileSizeMB} MB</span>
                </div>
                <Slider
                  value={[form.minimumFileSizeMB]}
                  onValueChange={(v) =>
                    updateField(
                      'minimumFileSizeMB',
                      Array.isArray(v) && typeof v[0] === 'number' ? v[0] : form.minimumFileSizeMB,
                    )
                  }
                  min={0}
                  max={500}
                  step={10}
                />
                <p className="text-muted-foreground text-xs">
                  Files smaller than this will be rejected (helps filter sample files)
                </p>
              </div>

              <ExtensionManager
                extensions={form.videoExtensions}
                onChange={(exts) => updateField('videoExtensions', exts)}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="matching" className="mt-6 max-w-2xl space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Match Behavior</CardTitle>
              <CardDescription>Configure how files are matched to library items</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-3">
                <Label>Match Conflict Behavior</Label>
                <Select
                  value={form.matchConflictBehavior}
                  onValueChange={(v) => v && updateField('matchConflictBehavior', v)}
                >
                  <SelectTrigger>
                    {
                      MATCH_CONFLICT_OPTIONS.find((o) => o.value === form.matchConflictBehavior)
                        ?.label
                    }
                  </SelectTrigger>
                  <SelectContent>
                    {MATCH_CONFLICT_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-muted-foreground text-xs">
                  {
                    MATCH_CONFLICT_OPTIONS.find((o) => o.value === form.matchConflictBehavior)
                      ?.description
                  }
                </p>
              </div>

              <div className="space-y-3">
                <Label>Unknown Media Handling</Label>
                <Select
                  value={form.unknownMediaBehavior}
                  onValueChange={(v) => v && updateField('unknownMediaBehavior', v)}
                >
                  <SelectTrigger>
                    {
                      UNKNOWN_MEDIA_OPTIONS.find((o) => o.value === form.unknownMediaBehavior)
                        ?.label
                    }
                  </SelectTrigger>
                  <SelectContent>
                    {UNKNOWN_MEDIA_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-muted-foreground text-xs">
                  {
                    UNKNOWN_MEDIA_OPTIONS.find((o) => o.value === form.unknownMediaBehavior)
                      ?.description
                  }
                </p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="tv-naming" className="mt-6 max-w-3xl space-y-6">
          <FilenameTester mediaType="tv" />

          <Card>
            <CardHeader>
              <CardTitle>Episode Renaming</CardTitle>
              <CardDescription>Configure how TV episodes are renamed during import</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Rename Episodes</Label>
                  <p className="text-muted-foreground text-sm">
                    Rename files according to format patterns
                  </p>
                </div>
                <Switch
                  checked={form.renameEpisodes}
                  onCheckedChange={(v) => updateField('renameEpisodes', v)}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Replace Illegal Characters</Label>
                  <p className="text-muted-foreground text-sm">
                    Replace filesystem-illegal characters with safe alternatives
                  </p>
                </div>
                <Switch
                  checked={form.replaceIllegalCharacters}
                  onCheckedChange={(v) => updateField('replaceIllegalCharacters', v)}
                />
              </div>

              <div className="space-y-3">
                <Label>Colon Replacement</Label>
                <Select
                  value={form.colonReplacement}
                  onValueChange={(v) => v && updateField('colonReplacement', v)}
                >
                  <SelectTrigger>
                    {
                      COLON_REPLACEMENT_OPTIONS.find((o) => o.value === form.colonReplacement)
                        ?.label
                    }
                  </SelectTrigger>
                  <SelectContent>
                    {COLON_REPLACEMENT_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-muted-foreground text-xs">
                  Example:{' '}
                  {
                    COLON_REPLACEMENT_OPTIONS.find((o) => o.value === form.colonReplacement)
                      ?.example
                  }
                </p>
                {form.colonReplacement === 'custom' && (
                  <Input
                    value={form.customColonReplacement ?? ''}
                    onChange={(e) => updateField('customColonReplacement', e.target.value)}
                    placeholder="Enter custom replacement character"
                  />
                )}
              </div>

              <div className="space-y-3">
                <Label>Multi-Episode Style</Label>
                <Select
                  value={form.multiEpisodeStyle}
                  onValueChange={(v) => v && updateField('multiEpisodeStyle', v)}
                >
                  <SelectTrigger>
                    {MULTI_EPISODE_STYLES.find((s) => s.value === form.multiEpisodeStyle)?.label}
                  </SelectTrigger>
                  <SelectContent>
                    {MULTI_EPISODE_STYLES.map((style) => (
                      <SelectItem key={style.value} value={style.value}>
                        {style.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-muted-foreground font-mono text-xs">
                  Example:{' '}
                  {MULTI_EPISODE_STYLES.find((s) => s.value === form.multiEpisodeStyle)?.example}
                </p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Episode Format Patterns</CardTitle>
              <CardDescription>Define naming patterns for different episode types</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <PatternEditor
                label="Standard Episode Format"
                value={form.standardEpisodeFormat}
                onChange={(v) => updateField('standardEpisodeFormat', v)}
                description="For regular TV series"
                mediaType="episode"
                tokenContext="episode"
              />

              <PatternEditor
                label="Daily Episode Format"
                value={form.dailyEpisodeFormat}
                onChange={(v) => updateField('dailyEpisodeFormat', v)}
                description="For daily/date-based shows"
                mediaType="episode"
                tokenContext="episode"
              />

              <PatternEditor
                label="Anime Episode Format"
                value={form.animeEpisodeFormat}
                onChange={(v) => updateField('animeEpisodeFormat', v)}
                description="For anime series"
                mediaType="episode"
                tokenContext="episode"
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Folder Format Patterns</CardTitle>
              <CardDescription>
                Define folder naming patterns for series organization
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <PatternEditor
                label="Series Folder Format"
                value={form.seriesFolderFormat}
                onChange={(v) => updateField('seriesFolderFormat', v)}
                description="Root folder for each series"
                mediaType="folder"
                tokenContext="series-folder"
              />

              <PatternEditor
                label="Season Folder Format"
                value={form.seasonFolderFormat}
                onChange={(v) => updateField('seasonFolderFormat', v)}
                description="Subfolder for each season"
                mediaType="folder"
                tokenContext="season-folder"
              />

              <PatternEditor
                label="Specials Folder Format"
                value={form.specialsFolderFormat}
                onChange={(v) => updateField('specialsFolderFormat', v)}
                description="Folder for specials (Season 0)"
                mediaType="folder"
                tokenContext="series-folder"
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="movie-naming" className="mt-6 max-w-3xl space-y-6">
          <FilenameTester mediaType="movie" />

          <Card>
            <CardHeader>
              <CardTitle>Movie Renaming</CardTitle>
              <CardDescription>Configure how movies are renamed during import</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Rename Movies</Label>
                  <p className="text-muted-foreground text-sm">
                    Rename files according to format patterns
                  </p>
                </div>
                <Switch
                  checked={form.renameMovies}
                  onCheckedChange={(v) => updateField('renameMovies', v)}
                />
              </div>

              <PatternEditor
                label="Movie Folder Format"
                value={form.movieFolderFormat}
                onChange={(v) => updateField('movieFolderFormat', v)}
                description="Folder name for each movie"
                mediaType="folder"
                tokenContext="movie-folder"
              />

              <PatternEditor
                label="Movie File Format"
                value={form.movieFileFormat}
                onChange={(v) => updateField('movieFileFormat', v)}
                description="Filename pattern for movie files"
                mediaType="movie"
                tokenContext="movie"
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="tokens" className="mt-6 max-w-4xl space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Token Reference</CardTitle>
              <CardDescription>Available tokens for naming patterns</CardDescription>
            </CardHeader>
            <CardContent>
              <Accordion>
                {Object.entries(TOKEN_REFERENCE).map(([category, tokens]) => (
                  <AccordionItem key={category} value={category}>
                    <AccordionTrigger className="capitalize">
                      {category.replaceAll(/([A-Z])/g, ' $1').trim()} Tokens
                    </AccordionTrigger>
                    <AccordionContent>
                      <div className="space-y-2">
                        {tokens.map((t) => (
                          <div
                            key={t.token}
                            className="flex items-start gap-4 border-b py-2 last:border-0"
                          >
                            <code className="bg-muted min-w-[180px] rounded px-2 py-1 font-mono text-sm">
                              {t.token}
                            </code>
                            <div className="flex-1 text-sm">
                              <p>{t.description}</p>
                              <p className="text-muted-foreground mt-1">
                                Example: <span className="font-mono">{t.example}</span>
                              </p>
                            </div>
                          </div>
                        ))}
                      </div>
                    </AccordionContent>
                  </AccordionItem>
                ))}
              </Accordion>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Token Modifiers</CardTitle>
              <CardDescription>Additional formatting options for tokens</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4 text-sm">
              <div>
                <h4 className="mb-2 font-medium">Separator Control</h4>
                <p className="text-muted-foreground mb-2">Control word separation within tokens:</p>
                <ul className="text-muted-foreground list-inside list-disc space-y-1">
                  <li>
                    <code>{'{Series Title}'}</code> - Space separator (default)
                  </li>
                  <li>
                    <code>{'{Series.Title}'}</code> - Period separator
                  </li>
                  <li>
                    <code>{'{Series-Title}'}</code> - Dash separator
                  </li>
                  <li>
                    <code>{'{Series_Title}'}</code> - Underscore separator
                  </li>
                </ul>
              </div>

              <div>
                <h4 className="mb-2 font-medium">Truncation</h4>
                <p className="text-muted-foreground mb-2">
                  Limit token length to prevent path issues:
                </p>
                <ul className="text-muted-foreground list-inside list-disc space-y-1">
                  <li>
                    <code>{'{Episode Title:30}'}</code> - Truncate to 30 chars from end
                  </li>
                  <li>
                    <code>{'{Episode Title:-30}'}</code> - Truncate to 30 chars from start
                  </li>
                </ul>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
