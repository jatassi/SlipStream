import { useCallback, useState } from 'react'

import {
  AlertCircle,
  ChevronRight,
  ChevronUp,
  CornerDownRight,
  FileVideo,
  FolderOpen,
  HardDrive,
  Import,
  Loader2,
  Pencil,
  RefreshCw,
  Scan,
} from 'lucide-react'
import { toast } from 'sonner'

import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { useEpisodes, useGlobalLoading, useMovies, useSeries } from '@/hooks'
import { useBrowseForImport } from '@/hooks/useFilesystem'
import {
  useManualImport,
  usePendingImports,
  useRetryImport,
  useScanDirectory,
} from '@/hooks/useImport'
import { useMultiVersionSettings, useSlots } from '@/hooks/useSlots'
import type { ScannedFile, Slot } from '@/types'

function formatFileSize(bytes: number): string {
  if (bytes === 0) {
    return '0 B'
  }
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${Number.parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

function FileBrowser({
  currentPath,
  onPathChange,
  onScanPath,
  isScanning,
  scannedFiles,
  selectedFiles,
  onToggleFile,
  onToggleAll,
  onEditMatch,
  onImportFile,
  onClearScan,
  onImportSelected,
  isImporting,
}: {
  currentPath: string
  onPathChange: (path: string) => void
  onScanPath: (path: string) => void
  isScanning: boolean
  scannedFiles: ScannedFile[]
  selectedFiles: Set<string>
  onToggleFile: (path: string) => void
  onToggleAll: () => void
  onEditMatch: (file: ScannedFile) => void
  onImportFile: (file: ScannedFile) => void
  onClearScan: () => void
  onImportSelected: () => void
  isImporting: boolean
}) {
  const [pathInput, setPathInput] = useState(currentPath)
  const globalLoading = useGlobalLoading()
  const { data, isLoading: queryLoading, refetch } = useBrowseForImport(currentPath || undefined)
  const isLoading = queryLoading || globalLoading

  const showScanResults = scannedFiles.length > 0

  const navigateTo = (path: string) => {
    onPathChange(path)
    setPathInput(path)
  }

  const navigateUp = () => {
    if (data?.parent) {
      onPathChange(data.parent)
      setPathInput(data.parent)
    } else {
      onPathChange('')
      setPathInput('')
    }
  }

  const handlePathInputNavigate = () => {
    if (pathInput) {
      onPathChange(pathInput)
    }
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-base">
              {showScanResults ? 'Scanned Files' : 'File Browser'}
            </CardTitle>
            {showScanResults ? (
              <CardDescription>
                {scannedFiles.length} files found,{' '}
                {scannedFiles.filter((f) => f.suggestedMatch).length} ready to import
              </CardDescription>
            ) : null}
          </div>
          <div className="flex gap-2">
            {showScanResults ? (
              <>
                {selectedFiles.size > 0 && (
                  <Button size="sm" onClick={onImportSelected} disabled={isImporting}>
                    {isImporting ? 'Importing...' : `Import ${selectedFiles.size} Selected`}
                  </Button>
                )}
                <Button size="sm" variant="outline" onClick={onClearScan}>
                  Back to Browser
                </Button>
              </>
            ) : (
              <>
                <Button size="sm" variant="outline" onClick={() => refetch()} disabled={isLoading}>
                  <RefreshCw className="size-4" />
                </Button>
                {currentPath ? (
                  <Button
                    size="sm"
                    onClick={() => onScanPath(currentPath)}
                    disabled={isScanning || isLoading}
                  >
                    {isScanning ? (
                      <Loader2 className="mr-2 size-4 animate-spin" />
                    ) : (
                      <Scan className="mr-2 size-4" />
                    )}
                    Scan Directory
                  </Button>
                ) : null}
              </>
            )}
          </div>
        </div>
        {!showScanResults && (
          <div className="mt-2 flex gap-2">
            <Input
              placeholder="Enter path..."
              value={pathInput}
              onChange={(e) => setPathInput(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handlePathInputNavigate()}
              className="h-8 font-mono text-xs"
            />
            <Button
              size="sm"
              variant="outline"
              onClick={handlePathInputNavigate}
              className="h-8 px-2"
            >
              <ChevronRight className="size-4" />
            </Button>
          </div>
        )}
      </CardHeader>
      <CardContent>
        {showScanResults ? (
          <ScrollArea className="h-[500px]">
            <ScannedFilesList
              files={scannedFiles}
              selectedFiles={selectedFiles}
              onToggleFile={onToggleFile}
              onToggleAll={onToggleAll}
              onEditMatch={onEditMatch}
              onImportFile={onImportFile}
            />
          </ScrollArea>
        ) : isLoading ? (
          <div className="space-y-1">
            {[55, 40, 65, 48, 60, 42, 70, 50].map((w) => (
              <div key={w} className="flex items-center gap-2 p-2">
                <Skeleton className="size-4 shrink-0 rounded" />
                <Skeleton className="h-4" style={{ width: `${w}%` }} />
                {w % 3 === 0 && <Skeleton className="ml-auto size-4 shrink-0" />}
              </div>
            ))}
          </div>
        ) : (
          <ScrollArea className="h-[400px]">
            <div className="space-y-1">
              {/* Back button */}
              {currentPath || data?.parent ? (
                <button
                  onClick={navigateUp}
                  className="hover:bg-muted flex w-full items-center gap-2 rounded-md p-2 text-left"
                >
                  <ChevronUp className="text-muted-foreground size-4" />
                  <span className="text-sm">..</span>
                </button>
              ) : null}

              {/* Drives (Windows root) */}
              {data?.drives?.map((drive) => (
                <button
                  key={drive.letter}
                  onClick={() => navigateTo(`${drive.letter}\\`)}
                  className="hover:bg-muted flex w-full items-center gap-2 rounded-md p-2 text-left"
                >
                  <HardDrive className="text-muted-foreground size-4" />
                  <span className="text-sm font-medium">{drive.letter}</span>
                  {drive.label ? (
                    <span className="text-muted-foreground text-sm">({drive.label})</span>
                  ) : null}
                </button>
              ))}

              {/* Directories */}
              {data?.directories.map((dir) => (
                <button
                  key={dir.path}
                  onClick={() => navigateTo(dir.path)}
                  className="hover:bg-muted flex w-full items-center gap-2 rounded-md p-2 text-left"
                >
                  <FolderOpen className="size-4 text-yellow-600" />
                  <span className="text-sm">{dir.name}</span>
                  <ChevronRight className="text-muted-foreground ml-auto size-4" />
                </button>
              ))}

              {/* Video files */}
              {data?.files.map((file) => (
                <div
                  key={file.path}
                  className="hover:bg-muted flex w-full items-center gap-2 rounded-md p-2"
                >
                  <FileVideo className="size-4 text-blue-600" />
                  <span className="flex-1 truncate text-sm">{file.name}</span>
                  <span className="text-muted-foreground text-xs">{formatFileSize(file.size)}</span>
                </div>
              ))}

              {data &&
                !data.drives?.length &&
                !data.directories.length &&
                !data.files.length && (
                  <p className="text-muted-foreground py-4 text-center text-sm">No items found</p>
                )}
            </div>
          </ScrollArea>
        )}
      </CardContent>
    </Card>
  )
}

function EditMatchDialog({
  file,
  open,
  onClose,
  onConfirm,
}: {
  file: ScannedFile | null
  open: boolean
  onClose: () => void
  onConfirm: (
    file: ScannedFile,
    match: {
      mediaType: string
      mediaId: number
      seriesId?: number
      seasonNum?: number
      targetSlotId?: number
    },
  ) => void
}) {
  const { data: movies } = useMovies()
  const { data: allSeries } = useSeries()
  const { data: multiVersionSettings } = useMultiVersionSettings()
  const { data: slots } = useSlots()

  const initialType =
    file?.suggestedMatch?.mediaType === 'movie'
      ? 'movie'
      : file?.parsedInfo?.isTV
        ? 'episode'
        : 'movie'
  const [selectedType, setSelectedType] = useState<'movie' | 'episode'>(initialType)
  const [selectedMovieId, setSelectedMovieId] = useState<string>(
    file?.suggestedMatch?.mediaType === 'movie' ? String(file.suggestedMatch.mediaId) : '',
  )
  const [selectedSeriesId, setSelectedSeriesId] = useState<string>(
    file?.suggestedMatch?.seriesId ? String(file.suggestedMatch.seriesId) : '',
  )
  const [selectedEpisodeId, setSelectedEpisodeId] = useState<string>(
    file?.suggestedMatch?.mediaType === 'episode' ? String(file.suggestedMatch.mediaId) : '',
  )
  const [selectedSlotId, setSelectedSlotId] = useState<string>('')

  const seriesIdNum = selectedSeriesId ? Number.parseInt(selectedSeriesId) : 0
  const { data: episodes } = useEpisodes(seriesIdNum)

  const isMultiVersionEnabled = multiVersionSettings?.enabled ?? false
  const enabledSlots = slots?.filter((s: Slot) => s.enabled) ?? []

  if (!file) {
    return null
  }

  const parsed = file.parsedInfo

  const handleConfirm = () => {
    if (selectedType === 'movie') {
      if (!selectedMovieId) {
        toast.error('Please select a movie')
        return
      }
      onConfirm(file, {
        mediaType: 'movie',
        mediaId: Number.parseInt(selectedMovieId),
        targetSlotId: selectedSlotId ? Number.parseInt(selectedSlotId) : undefined,
      })
    } else {
      if (!selectedEpisodeId) {
        toast.error('Please select an episode')
        return
      }
      onConfirm(file, {
        mediaType: 'episode',
        mediaId: Number.parseInt(selectedEpisodeId),
        seriesId: selectedSeriesId ? Number.parseInt(selectedSeriesId) : undefined,
        seasonNum: parsed?.season,
        targetSlotId: selectedSlotId ? Number.parseInt(selectedSlotId) : undefined,
      })
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Edit Import Match</DialogTitle>
          <DialogDescription>
            Review parsed information and select the library item to import as
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="bg-muted rounded-lg p-3">
            <p className="text-sm font-medium break-all">{file.fileName}</p>
            <p className="text-muted-foreground mt-1 text-xs">{formatFileSize(file.fileSize)}</p>
          </div>

          {parsed ? (
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-3">
                <h4 className="text-muted-foreground text-sm font-medium">Parsed Information</h4>
                <div className="space-y-2 text-sm">
                  {parsed.title ? (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Title:</span>
                      <span className="font-medium">{parsed.title}</span>
                    </div>
                  ) : null}
                  {parsed.year ? (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Year:</span>
                      <span>{parsed.year}</span>
                    </div>
                  ) : null}
                  {parsed.isTV ? (
                    <>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Season:</span>
                        <span>{parsed.season}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Episode:</span>
                        <span>
                          {parsed.episode}
                          {parsed.endEpisode && parsed.endEpisode !== parsed.episode
                            ? `-${parsed.endEpisode}`
                            : ''}
                        </span>
                      </div>
                    </>
                  ) : null}
                </div>
              </div>

              <div className="space-y-3">
                <h4 className="text-muted-foreground text-sm font-medium">Quality Information</h4>
                <div className="space-y-2 text-sm">
                  {parsed.quality ? (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Quality:</span>
                      <Badge variant="secondary">{parsed.quality}</Badge>
                    </div>
                  ) : null}
                  {parsed.source ? (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Source:</span>
                      <span>{parsed.source}</span>
                    </div>
                  ) : null}
                  {parsed.codec ? (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Codec:</span>
                      <span>{parsed.codec}</span>
                    </div>
                  ) : null}
                  {parsed.audioCodecs && parsed.audioCodecs.length > 0 ? (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Audio:</span>
                      <span>{parsed.audioCodecs.join(', ')}</span>
                    </div>
                  ) : null}
                </div>
              </div>
            </div>
          ) : null}

          <div className="space-y-4 border-t pt-4">
            <h4 className="text-sm font-medium">Match to Library</h4>

            <div className="space-y-2">
              <Label htmlFor="media-type">Media Type</Label>
              <Select value={selectedType} onValueChange={(v) => v && setSelectedType(v)}>
                <SelectTrigger id="media-type">{selectedType === 'movie' ? 'Movie' : 'TV Episode'}</SelectTrigger>
                <SelectContent>
                  <SelectItem value="movie">Movie</SelectItem>
                  <SelectItem value="episode">TV Episode</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {selectedType === 'movie' && movies ? (
              <div className="space-y-2">
                <Label htmlFor="movie-select">Movie</Label>
                <Select value={selectedMovieId} onValueChange={(v) => v && setSelectedMovieId(v)}>
                  <SelectTrigger id="movie-select">
                    {selectedMovieId
                      ? (() => {
                          const movie = movies.find((m) => m.id.toString() === selectedMovieId)
                          return movie ? `${movie.title} (${movie.year})` : 'Select a movie'
                        })()
                      : 'Select a movie'}
                  </SelectTrigger>
                  <SelectContent>
                    {movies.map((movie) => (
                      <SelectItem key={movie.id} value={movie.id.toString()}>
                        {movie.title} ({movie.year})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}

            {selectedType === 'episode' && allSeries ? (
              <>
                <div className="space-y-2">
                  <Label htmlFor="series-select">Series</Label>
                  <Select
                    value={selectedSeriesId}
                    onValueChange={(v) => {
                      if (v) {
                        setSelectedSeriesId(v)
                        setSelectedEpisodeId('')
                      }
                    }}
                  >
                    <SelectTrigger id="series-select">
                      {selectedSeriesId
                        ? allSeries.find((s) => s.id.toString() === selectedSeriesId)?.title ||
                          'Select a series'
                        : 'Select a series'}
                    </SelectTrigger>
                    <SelectContent>
                      {allSeries.map((s) => (
                        <SelectItem key={s.id} value={s.id.toString()}>
                          {s.title}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {selectedSeriesId && episodes && episodes.length > 0 ? (
                  <div className="space-y-2">
                    <Label htmlFor="episode-select">Episode</Label>
                    <Select
                      value={selectedEpisodeId}
                      onValueChange={(v) => v && setSelectedEpisodeId(v)}
                    >
                      <SelectTrigger id="episode-select">
                        {selectedEpisodeId
                          ? (() => {
                              const ep = episodes.find((e) => e.id.toString() === selectedEpisodeId)
                              return ep
                                ? `S${String(ep.seasonNumber).padStart(2, '0')}E${String(ep.episodeNumber).padStart(2, '0')} - ${ep.title}`
                                : 'Select an episode'
                            })()
                          : 'Select an episode'}
                      </SelectTrigger>
                      <SelectContent>
                        {episodes
                          .toSorted((a, b) =>
                            a.seasonNumber === b.seasonNumber
                              ? a.episodeNumber - b.episodeNumber
                              : a.seasonNumber - b.seasonNumber,
                          )
                          .map((ep) => (
                            <SelectItem key={ep.id} value={ep.id.toString()}>
                              S{String(ep.seasonNumber).padStart(2, '0')}E
                              {String(ep.episodeNumber).padStart(2, '0')} - {ep.title}
                            </SelectItem>
                          ))}
                      </SelectContent>
                    </Select>
                  </div>
                ) : null}
              </>
            ) : null}

            {isMultiVersionEnabled && enabledSlots.length > 0 ? (
              <div className="mt-4 space-y-2 border-t pt-4">
                <h4 className="text-sm font-medium">Version Slot (Multi-Version)</h4>
                <p className="text-muted-foreground text-xs">
                  Optionally assign this file to a specific version slot. Leave blank for automatic
                  assignment.
                </p>
                <Select value={selectedSlotId} onValueChange={(v) => setSelectedSlotId(v || '')}>
                  <SelectTrigger>
                    {selectedSlotId
                      ? enabledSlots.find((s: Slot) => s.id.toString() === selectedSlotId)?.name ||
                        'Select a slot'
                      : 'Auto-assign (recommended)'}
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="">Auto-assign (recommended)</SelectItem>
                    {enabledSlots.map((slot: Slot) => (
                      <SelectItem key={slot.id} value={slot.id.toString()}>
                        {slot.name} {slot.qualityProfile ? `(${slot.qualityProfile.name})` : ''}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={handleConfirm}>
            <Import className="mr-2 size-4" />
            Import
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function ScannedFilesList({
  files,
  selectedFiles,
  onToggleFile,
  onToggleAll,
  onEditMatch,
  onImportFile,
}: {
  files: ScannedFile[]
  selectedFiles: Set<string>
  onToggleFile: (path: string) => void
  onToggleAll: () => void
  onEditMatch: (file: ScannedFile) => void
  onImportFile: (file: ScannedFile) => void
}) {
  const matchedFiles = files.filter((f) => f.suggestedMatch)
  const allSelected =
    matchedFiles.length > 0 && matchedFiles.every((f) => selectedFiles.has(f.path))

  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2 border-b px-2 py-1.5">
        <Checkbox checked={allSelected} onCheckedChange={onToggleAll} />
        <span className="text-muted-foreground text-xs">Select all matched files</span>
      </div>

      {files.map((file) => {
        const hasMatch = !!file.suggestedMatch
        const match = file.suggestedMatch

        return (
          <div
            key={file.path}
            className={`rounded-lg border p-3 ${hasMatch ? 'border-green-600' : ''}`}
          >
            <div className="flex items-start gap-3">
              <Checkbox
                checked={selectedFiles.has(file.path)}
                onCheckedChange={() => onToggleFile(file.path)}
                disabled={!hasMatch}
                className="mt-1"
              />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <FileVideo
                    className={`size-4 shrink-0 ${hasMatch ? 'text-blue-600' : 'text-muted-foreground'}`}
                  />
                  <span className="truncate text-sm font-medium" title={file.fileName}>
                    {file.fileName}
                  </span>
                  <span className="text-muted-foreground shrink-0 text-xs">
                    {formatFileSize(file.fileSize)}
                  </span>
                  {!file.valid && (
                    <Badge variant="destructive" className="shrink-0 text-xs">
                      <AlertCircle className="mr-1 size-3" />
                      Invalid
                    </Badge>
                  )}
                </div>

                {hasMatch && match ? (
                  <div className="mt-2 ml-6 flex items-center gap-2">
                    <CornerDownRight className="text-muted-foreground size-4 shrink-0" />
                    <div className="flex flex-wrap items-center gap-1.5">
                      {match.mediaType === 'episode' ? (
                        <>
                          <Badge className="bg-primary">
                            {match.seriesTitle || 'Unknown Series'}
                          </Badge>
                          <Badge variant="secondary">
                            S{String(match.seasonNum ?? 0).padStart(2, '0')}E
                            {String(match.episodeNum ?? 0).padStart(2, '0')} - &quot;{match.mediaTitle}&quot;
                          </Badge>
                        </>
                      ) : (
                        <>
                          <Badge variant="outline">{match.mediaTitle}</Badge>
                          {match.year ? <Badge variant="secondary">{match.year}</Badge> : null}
                        </>
                      )}
                    </div>
                    <div className="ml-auto flex shrink-0 gap-1">
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-6 px-2"
                        onClick={() => onEditMatch(file)}
                      >
                        <Pencil className="mr-1 size-3" />
                        Edit Match
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-6 px-2"
                        onClick={() => onImportFile(file)}
                      >
                        <Import className="mr-1 size-3" />
                        Import
                      </Button>
                    </div>
                  </div>
                ) : null}

                {!hasMatch && (
                  <div className="mt-2 ml-6 flex items-center gap-2">
                    <CornerDownRight className="text-muted-foreground size-4 shrink-0" />
                    <span className="text-muted-foreground text-sm italic">
                      No library match found
                    </span>
                    <Button
                      size="sm"
                      variant="outline"
                      className="ml-auto h-6 shrink-0 px-2"
                      onClick={() => onEditMatch(file)}
                    >
                      <Pencil className="mr-1 size-3" />
                      Set Match
                    </Button>
                  </div>
                )}
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}

function PendingImportsCard() {
  const globalLoading = useGlobalLoading()
  const { data: pending, isLoading: queryLoading } = usePendingImports()
  const retryMutation = useRetryImport()
  const isLoading = queryLoading || globalLoading

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-4 w-48" />
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {Array.from({ length: 3 }, (_, i) => (
              <div key={i} className="flex items-center justify-between rounded-lg border p-2">
                <div className="min-w-0 flex-1 space-y-1.5">
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-5 w-16 rounded-full" />
                </div>
                <Skeleton className="h-8 w-14 rounded-md" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!pending || pending.length === 0) {
    return null
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Pending Imports</CardTitle>
        <CardDescription>Files waiting to be imported</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {pending.map((item) => (
            <div
              key={item.id || item.filePath}
              className="flex items-center justify-between rounded-lg border p-2"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{item.fileName}</p>
                <div className="mt-1 flex items-center gap-2">
                  <Badge variant={item.status === 'failed' ? 'destructive' : 'outline'}>
                    {item.status}
                  </Badge>
                  {item.isProcessing ? <Loader2 className="size-3 animate-spin" /> : null}
                </div>
                {item.error ? <p className="mt-1 text-xs text-red-600">{item.error}</p> : null}
              </div>
              {item.status === 'failed' && item.id ? (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => item.id && retryMutation.mutate(item.id)}
                  disabled={retryMutation.isPending}
                >
                  Retry
                </Button>
              ) : null}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

export function ManualImportPage() {
  const [currentPath, setCurrentPath] = useState('')
  const [scannedFiles, setScannedFiles] = useState<ScannedFile[]>([])
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set())
  const [importDialogFile, setImportDialogFile] = useState<ScannedFile | null>(null)

  const scanMutation = useScanDirectory()
  const importMutation = useManualImport()

  const handleScanPath = useCallback(
    async (path: string) => {
      try {
        const result = await scanMutation.mutateAsync({ path })
        setScannedFiles(result.files)
        const matchedPaths = result.files.filter((f) => f.suggestedMatch).map((f) => f.path)
        setSelectedFiles(new Set(matchedPaths))
      } catch {
        toast.error('Failed to scan directory')
      }
    },
    [scanMutation],
  )

  const handleToggleFile = (path: string) => {
    const file = scannedFiles.find((f) => f.path === path)
    if (!file?.suggestedMatch) {
      return
    }

    setSelectedFiles((prev) => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }

  const handleToggleAll = () => {
    const matchedFiles = scannedFiles.filter((f) => f.suggestedMatch)
    const allMatchedSelected =
      matchedFiles.length > 0 && matchedFiles.every((f) => selectedFiles.has(f.path))

    if (allMatchedSelected) {
      setSelectedFiles(new Set())
    } else {
      setSelectedFiles(new Set(matchedFiles.map((f) => f.path)))
    }
  }

  const handleImportSelected = async () => {
    const filesToImport = scannedFiles.filter((f) => selectedFiles.has(f.path) && f.suggestedMatch)

    if (filesToImport.length === 0) {
      return
    }

    let successCount = 0
    let failCount = 0
    let lastError = ''

    for (const file of filesToImport) {
      const match = file.suggestedMatch
      if (!match) {
        continue
      }
      try {
        const result = await importMutation.mutateAsync({
          path: file.path,
          mediaType: match.mediaType as 'movie' | 'episode',
          mediaId: match.mediaId,
          seriesId: match.seriesId,
          seasonNum: match.seasonNum,
        })

        if (result.success) {
          successCount++
          setScannedFiles((prev) => prev.filter((f) => f.path !== file.path))
          setSelectedFiles((prev) => {
            const next = new Set(prev)
            next.delete(file.path)
            return next
          })
        } else {
          failCount++
          if (result.error) {
            lastError = result.error
          }
        }
      } catch {
        failCount++
      }
    }

    if (successCount > 0 && failCount === 0) {
      toast.success(`Imported ${successCount} file${successCount > 1 ? 's' : ''}`)
    } else if (successCount > 0 && failCount > 0) {
      toast.warning(`Imported ${successCount}, failed ${failCount}`)
    } else {
      toast.error(lastError || 'Failed to import files')
    }
  }

  const handleImportFile = (file: ScannedFile) => {
    setImportDialogFile(file)
  }

  const handleConfirmImport = async (
    file: ScannedFile,
    match: {
      mediaType: string
      mediaId: number
      seriesId?: number
      seasonNum?: number
      targetSlotId?: number
    },
  ) => {
    try {
      const result = await importMutation.mutateAsync({
        path: file.path,
        mediaType: match.mediaType as 'movie' | 'episode',
        mediaId: match.mediaId,
        seriesId: match.seriesId,
        seasonNum: match.seasonNum,
        targetSlotId: match.targetSlotId,
      })

      if (result.success) {
        toast.success(`Imported ${file.fileName}`)
        setScannedFiles((prev) => prev.filter((f) => f.path !== file.path))
      } else {
        toast.error(result.error || 'Import failed')
      }
    } catch {
      toast.error('Failed to import file')
    }

    setImportDialogFile(null)
  }

  const handleClearScan = () => {
    setScannedFiles([])
    setSelectedFiles(new Set())
  }

  const handleDirectImport = async (file: ScannedFile) => {
    if (!file.suggestedMatch) {
      return
    }

    const match = file.suggestedMatch
    try {
      const result = await importMutation.mutateAsync({
        path: file.path,
        mediaType: match.mediaType as 'movie' | 'episode',
        mediaId: match.mediaId,
        seriesId: match.seriesId,
        seasonNum: match.seasonNum,
      })

      if (result.success) {
        toast.success(`Imported ${file.fileName}`)
        setScannedFiles((prev) => prev.filter((f) => f.path !== file.path))
        setSelectedFiles((prev) => {
          const next = new Set(prev)
          next.delete(file.path)
          return next
        })
      } else {
        toast.error(result.error || 'Import failed')
      }
    } catch {
      toast.error('Failed to import file')
    }
  }

  return (
    <div>
      <PageHeader title="Manual Import" description="Browse and import media files manually" />

      <div className="space-y-6">
        <FileBrowser
          currentPath={currentPath}
          onPathChange={setCurrentPath}
          onScanPath={handleScanPath}
          isScanning={scanMutation.isPending}
          scannedFiles={scannedFiles}
          selectedFiles={selectedFiles}
          onToggleFile={handleToggleFile}
          onToggleAll={handleToggleAll}
          onEditMatch={handleImportFile}
          onImportFile={handleDirectImport}
          onClearScan={handleClearScan}
          onImportSelected={handleImportSelected}
          isImporting={importMutation.isPending}
        />

        <PendingImportsCard />
      </div>

      <EditMatchDialog
        key={importDialogFile?.path}
        file={importDialogFile}
        open={!!importDialogFile}
        onClose={() => setImportDialogFile(null)}
        onConfirm={handleConfirmImport}
      />
    </div>
  )
}
