import { useState, useCallback, useRef } from 'react'
import {
  Clock,
  CheckCircle,
  XCircle,
  Download,
  Loader2,
  FlaskConical,
  Search,
  Zap,
  Trash2,
} from 'lucide-react'
import { PosterImage } from '@/components/media/PosterImage'
import { PageHeader } from '@/components/layout/PageHeader'
import { RequestsNav } from './RequestsNav'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Checkbox } from '@/components/ui/checkbox'
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { SearchModal } from '@/components/search/SearchModal'
import {
  useAdminRequests,
  useApproveRequest,
  useDenyRequest,
  useBatchDenyRequests,
  useDeleteRequest,
  useBatchDeleteRequests,
  useRootFolders,
  useDeveloperMode,
  useAddMovie,
  useAddSeries,
  useRequestSettings,
  useQualityProfiles,
  useAutoSearchMovie,
  useAutoSearchSeason,
  usePortalEnabled,
} from '@/hooks'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { AlertCircle } from 'lucide-react'
import { toast } from 'sonner'
import type { Request, RequestStatus, AddMovieInput, AddSeriesInput, SeasonInput } from '@/types'
import { formatDistanceToNow } from 'date-fns'

interface SearchModalState {
  open: boolean
  mediaType: 'movie' | 'series'
  mediaId: number
  mediaTitle: string
  tmdbId?: number
  imdbId?: string
  tvdbId?: number
  qualityProfileId: number
  year?: number
  season?: number
  pendingSeasons?: number[]
}

const STATUS_CONFIG: Record<RequestStatus, { label: string; icon: React.ReactNode; color: string }> = {
  pending: { label: 'Pending', icon: <Clock className="size-4" />, color: 'bg-yellow-500' },
  approved: { label: 'Approved', icon: <CheckCircle className="size-4" />, color: 'bg-blue-500' },
  denied: { label: 'Denied', icon: <XCircle className="size-4" />, color: 'bg-red-500' },
  downloading: { label: 'Downloading', icon: <Download className="size-4" />, color: 'bg-purple-500' },
  failed: { label: 'Failed', icon: <XCircle className="size-4" />, color: 'bg-red-700' },
  available: { label: 'Available', icon: <CheckCircle className="size-4" />, color: 'bg-green-500' },
  cancelled: { label: 'Cancelled', icon: <XCircle className="size-4" />, color: 'bg-gray-500' },
}

export function RequestQueuePage() {
  const [activeTab, setActiveTab] = useState<string>('pending')
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [showDenyDialog, setShowDenyDialog] = useState(false)
  const [denyReason, setDenyReason] = useState('')
  const [pendingDenyId, setPendingDenyId] = useState<number | null>(null)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [pendingDeleteId, setPendingDeleteId] = useState<number | null>(null)
  const [showBatchDeleteDialog, setShowBatchDeleteDialog] = useState(false)
  const [searchModal, setSearchModal] = useState<SearchModalState | null>(null)
  const [processingRequest, setProcessingRequest] = useState<number | null>(null)
  const pendingSeasonsRef = useRef<number[]>([])

  const { data: requests = [], isLoading, isError, refetch } = useAdminRequests()
  const { data: rootFolders } = useRootFolders()
  const { data: requestSettings } = useRequestSettings()
  const { data: qualityProfiles } = useQualityProfiles()
  const developerMode = useDeveloperMode()
  const portalEnabled = usePortalEnabled()

  const approveMutation = useApproveRequest()
  const denyMutation = useDenyRequest()
  const batchDenyMutation = useBatchDenyRequests()
  const deleteMutation = useDeleteRequest()
  const batchDeleteMutation = useBatchDeleteRequests()
  const addMovieMutation = useAddMovie()
  const addSeriesMutation = useAddSeries()
  const autoSearchMovieMutation = useAutoSearchMovie()
  const autoSearchSeasonMutation = useAutoSearchSeason()

  const getDefaultRootFolderId = useCallback((mediaType: string) => {
    if (requestSettings?.defaultRootFolderId) {
      return requestSettings.defaultRootFolderId
    }
    const matchingFolder = rootFolders?.find(f =>
      (mediaType === 'movie' && f.mediaType === 'movie') ||
      (mediaType === 'series' && f.mediaType === 'tv')
    )
    return matchingFolder?.id || rootFolders?.[0]?.id || 0
  }, [requestSettings, rootFolders])

  const getDefaultQualityProfileId = useCallback(() => {
    return qualityProfiles?.[0]?.id || 0
  }, [qualityProfiles])

  const filteredRequests = requests.filter((r) => {
    if (activeTab === 'all') return true
    return r.status === activeTab
  })

  const pendingCount = requests.filter((r) => r.status === 'pending').length

  const isAllSelected = filteredRequests.length > 0 && filteredRequests.every((r) => selectedIds.has(r.id))
  const isSomeSelected = selectedIds.size > 0

  const toggleSelectAll = () => {
    if (isAllSelected) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(filteredRequests.map((r) => r.id)))
    }
  }

  const toggleSelect = (id: number) => {
    const newSelected = new Set(selectedIds)
    if (newSelected.has(id)) {
      newSelected.delete(id)
    } else {
      newSelected.add(id)
    }
    setSelectedIds(newSelected)
  }

  const openDenyDialog = (id: number | null = null) => {
    setPendingDenyId(id)
    setDenyReason('')
    setShowDenyDialog(true)
  }

  const openDeleteDialog = (id: number) => {
    setPendingDeleteId(id)
    setShowDeleteDialog(true)
  }

  const handleDelete = async () => {
    if (!pendingDeleteId) return
    try {
      await deleteMutation.mutateAsync(pendingDeleteId)
      toast.success('Request deleted')
      setShowDeleteDialog(false)
      setPendingDeleteId(null)
    } catch {
      toast.error('Failed to delete request')
    }
  }

  const handleBatchDelete = async () => {
    if (selectedIds.size === 0) return
    try {
      const result = await batchDeleteMutation.mutateAsync(Array.from(selectedIds))
      toast.success(`${result.deleted} request${result.deleted !== 1 ? 's' : ''} deleted`)
      setShowBatchDeleteDialog(false)
      setSelectedIds(new Set())
    } catch {
      toast.error('Failed to delete requests')
    }
  }

  const addToLibrary = async (request: Request) => {
    const rootFolderId = getDefaultRootFolderId(request.mediaType)
    const qualityProfileId = getDefaultQualityProfileId()

    if (!rootFolderId || !qualityProfileId) {
      throw new Error('Missing root folder or quality profile configuration')
    }

    if (request.mediaType === 'movie') {
      const input: AddMovieInput = {
        title: request.title,
        year: request.year || undefined,
        tmdbId: request.tmdbId || undefined,
        rootFolderId,
        qualityProfileId,
        monitored: true,
        posterUrl: request.posterUrl || undefined,
        searchOnAdd: false,
      }
      const movie = await addMovieMutation.mutateAsync(input)
      // Return movie data including tmdbId/imdbId from metadata fetch
      return {
        mediaId: movie.id,
        qualityProfileId: movie.qualityProfileId,
        tmdbId: movie.tmdbId,
        imdbId: movie.imdbId,
        year: movie.year,
      }
    } else {
      const seasons: SeasonInput[] = []
      if (request.requestedSeasons && request.requestedSeasons.length > 0) {
        for (const seasonNum of request.requestedSeasons) {
          seasons.push({ seasonNumber: seasonNum, monitored: true })
        }
      }
      if (request.monitorFuture) {
        seasons.push({ seasonNumber: -1, monitored: true })
      }

      const input: AddSeriesInput = {
        title: request.title,
        year: request.year || undefined,
        tmdbId: request.tmdbId || undefined,
        tvdbId: request.tvdbId || undefined,
        rootFolderId,
        qualityProfileId,
        monitored: true,
        seasonFolder: true,
        posterUrl: request.posterUrl || undefined,
        searchOnAdd: 'no',
        monitorOnAdd: 'none',
        seasons: seasons.length > 0 ? seasons : undefined,
      }
      const series = await addSeriesMutation.mutateAsync(input)
      // Return series data including tvdbId from metadata fetch
      return {
        mediaId: series.id,
        qualityProfileId: series.qualityProfileId,
        tvdbId: series.tvdbId,
        year: series.year,
      }
    }
  }

  const handleApproveOnly = async (request: Request) => {
    setProcessingRequest(request.id)
    try {
      await approveMutation.mutateAsync({
        id: request.id,
        input: { action: 'approve_only' },
      })
      await addToLibrary(request)
      toast.success('Request approved and added to library')
    } catch (error) {
      toast.error('Failed to approve request', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    } finally {
      setProcessingRequest(null)
    }
  }

  const handleApproveAndManualSearch = async (request: Request) => {
    setProcessingRequest(request.id)
    try {
      await approveMutation.mutateAsync({
        id: request.id,
        input: { action: 'manual_search' },
      })
      const libraryMedia = await addToLibrary(request)

      console.log('[RequestQueue] Opening search modal:', {
        mediaId: libraryMedia.mediaId,
        qualityProfileId: libraryMedia.qualityProfileId,
        title: request.title,
        tmdbId: libraryMedia.tmdbId,
        imdbId: 'imdbId' in libraryMedia ? libraryMedia.imdbId : undefined,
        tvdbId: 'tvdbId' in libraryMedia ? libraryMedia.tvdbId : undefined,
        year: libraryMedia.year,
        mediaType: request.mediaType,
      })

      if (request.mediaType === 'movie' && 'imdbId' in libraryMedia) {
        setSearchModal({
          open: true,
          mediaType: 'movie',
          mediaId: libraryMedia.mediaId,
          mediaTitle: request.title,
          tmdbId: libraryMedia.tmdbId,
          imdbId: libraryMedia.imdbId,
          qualityProfileId: libraryMedia.qualityProfileId,
          year: libraryMedia.year,
        })
      } else {
        const seasonsToSearch = request.requestedSeasons && request.requestedSeasons.length > 0
          ? [...request.requestedSeasons].sort((a, b) => a - b)
          : [1]

        pendingSeasonsRef.current = seasonsToSearch.slice(1)
        setSearchModal({
          open: true,
          mediaType: 'series',
          mediaId: libraryMedia.mediaId,
          mediaTitle: request.title,
          tvdbId: 'tvdbId' in libraryMedia ? libraryMedia.tvdbId : undefined,
          qualityProfileId: libraryMedia.qualityProfileId,
          season: seasonsToSearch[0],
          pendingSeasons: seasonsToSearch.slice(1),
        })
      }
      toast.success('Request approved')
    } catch (error) {
      toast.error('Failed to approve request', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    } finally {
      setProcessingRequest(null)
    }
  }

  const handleApproveAndAutoSearch = async (request: Request) => {
    setProcessingRequest(request.id)
    try {
      await approveMutation.mutateAsync({
        id: request.id,
        input: { action: 'auto_search' },
      })
      const { mediaId } = await addToLibrary(request)

      if (request.mediaType === 'movie') {
        const result = await autoSearchMovieMutation.mutateAsync(mediaId)
        if (result.downloaded) {
          toast.success('Request approved and download started')
        } else if (result.found) {
          toast.success('Request approved, release found but not grabbed')
        } else {
          toast.success('Request approved, no releases found')
        }
      } else {
        const seasonsToSearch = request.requestedSeasons && request.requestedSeasons.length > 0
          ? [...request.requestedSeasons].sort((a, b) => a - b)
          : []

        let totalDownloaded = 0
        let totalFound = 0
        for (const seasonNum of seasonsToSearch) {
          try {
            const result = await autoSearchSeasonMutation.mutateAsync({
              seriesId: mediaId,
              seasonNumber: seasonNum,
            })
            totalDownloaded += result.downloaded
            totalFound += result.found
          } catch {
            // Continue searching other seasons
          }
        }
        if (totalDownloaded > 0) {
          toast.success(`Request approved, ${totalDownloaded} download(s) started`)
        } else if (totalFound > 0) {
          toast.success('Request approved, releases found but not grabbed')
        } else {
          toast.success('Request approved, no releases found')
        }
      }
    } catch (error) {
      toast.error('Failed to process request', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    } finally {
      setProcessingRequest(null)
    }
  }

  const handleSearchModalClose = () => {
    if (searchModal?.pendingSeasons && searchModal.pendingSeasons.length > 0) {
      const nextSeason = searchModal.pendingSeasons[0]
      const remainingSeasons = searchModal.pendingSeasons.slice(1)
      setSearchModal({
        ...searchModal,
        season: nextSeason,
        pendingSeasons: remainingSeasons,
      })
    } else {
      setSearchModal(null)
    }
  }

  const handleDeny = async () => {
    try {
      if (pendingDenyId) {
        await denyMutation.mutateAsync({
          id: pendingDenyId,
          input: denyReason ? { reason: denyReason } : undefined,
        })
        toast.success('Request denied')
      } else if (selectedIds.size > 0) {
        await batchDenyMutation.mutateAsync({
          ids: Array.from(selectedIds),
          reason: denyReason || undefined,
        })
        toast.success(`${selectedIds.size} requests denied`)
        setSelectedIds(new Set())
      }
      setShowDenyDialog(false)
    } catch {
      toast.error('Failed to deny request(s)')
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Request Queue" />
        <div className="max-w-6xl mx-auto pt-6 px-6">
          <LoadingState variant="list" count={5} />
        </div>
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Request Queue" />
        <div className="max-w-6xl mx-auto pt-6 px-6">
          <ErrorState onRetry={refetch} />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="External Requests"
        description="Manage portal users and content requests"
        breadcrumbs={[
          { label: 'Settings', href: '/settings/media' },
          { label: 'External Requests' },
        ]}
        actions={
          developerMode && (
            <Button
              variant="outline"
              onClick={() => toast.info('Test request feature coming soon', {
                description: 'This will allow creating test requests for debugging.'
              })}
            >
              <FlaskConical className="size-4 mr-2" />
              Test Request
            </Button>
          )
        }
      />

      <RequestsNav />

      {!portalEnabled && (
        <Alert>
          <AlertCircle className="size-4" />
          <AlertDescription>
            The external requests portal is currently disabled. Portal users cannot submit new requests or access the portal.
            You can re-enable it in the <a href="/settings/requests/settings" className="underline font-medium">Settings</a> tab.
          </AlertDescription>
        </Alert>
      )}

      <Tabs value={activeTab} onValueChange={(value) => { setActiveTab(value); setSelectedIds(new Set()) }}>
        <div className="flex items-center justify-between mb-4">
          <TabsList>
            <TabsTrigger value="pending">
              Pending {pendingCount > 0 && <Badge variant="secondary" className="ml-1">{pendingCount}</Badge>}
            </TabsTrigger>
            <TabsTrigger value="approved">Approved</TabsTrigger>
            <TabsTrigger value="downloading">Downloading</TabsTrigger>
            <TabsTrigger value="available">Available</TabsTrigger>
            <TabsTrigger value="denied">Denied</TabsTrigger>
            <TabsTrigger value="all">All</TabsTrigger>
          </TabsList>

          {isSomeSelected && (
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">{selectedIds.size} selected</span>
              <Button size="sm" variant="destructive" onClick={() => openDenyDialog()}>
                <XCircle className="size-4 mr-1" />
                Deny
              </Button>
              <Button size="sm" variant="outline" onClick={() => setShowBatchDeleteDialog(true)}>
                <Trash2 className="size-4 mr-1" />
                Delete
              </Button>
            </div>
          )}
        </div>

        <TabsContent value={activeTab} className="mt-0">
          {filteredRequests.length === 0 ? (
            <EmptyState
              icon={<Clock className="size-8" />}
              title={`No ${activeTab === 'all' ? '' : activeTab} requests`}
              description={activeTab === 'pending' ? 'No requests waiting for approval' : `No requests with status "${activeTab}"`}
            />
          ) : (
            <div className="rounded-md border">
              <div className="flex items-center gap-4 p-3 border-b bg-muted/40">
                <Checkbox
                  checked={isAllSelected}
                  onCheckedChange={toggleSelectAll}
                />
                <span className="text-sm text-muted-foreground">
                  {filteredRequests.length} request{filteredRequests.length !== 1 ? 's' : ''}
                </span>
              </div>
              <div className="divide-y">
                {filteredRequests.map((request) => (
                  <RequestRow
                    key={request.id}
                    request={request}
                    selected={selectedIds.has(request.id)}
                    isProcessing={processingRequest === request.id}
                    onToggleSelect={() => toggleSelect(request.id)}
                    onApproveOnly={() => handleApproveOnly(request)}
                    onApproveAndManualSearch={() => handleApproveAndManualSearch(request)}
                    onApproveAndAutoSearch={() => handleApproveAndAutoSearch(request)}
                    onDeny={() => openDenyDialog(request.id)}
                    onDelete={() => openDeleteDialog(request.id)}
                  />
                ))}
              </div>
            </div>
          )}
        </TabsContent>
      </Tabs>

      <Dialog open={showDenyDialog} onOpenChange={setShowDenyDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Deny Request{selectedIds.size > 1 ? 's' : ''}</DialogTitle>
            <DialogDescription>
              {pendingDenyId
                ? 'Optionally provide a reason for denying this request.'
                : `Deny ${selectedIds.size} selected request${selectedIds.size !== 1 ? 's' : ''}.`}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Reason (Optional)</Label>
              <Textarea
                placeholder="e.g., Content not available in region, already in library, etc."
                value={denyReason}
                onChange={(e) => setDenyReason(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDenyDialog(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeny}
              disabled={denyMutation.isPending || batchDenyMutation.isPending}
            >
              {(denyMutation.isPending || batchDenyMutation.isPending) && (
                <Loader2 className="size-4 mr-2 animate-spin" />
              )}
              Deny
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Request</DialogTitle>
            <DialogDescription>
              Are you sure you want to permanently delete this request? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeleteDialog(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && (
                <Loader2 className="size-4 mr-2 animate-spin" />
              )}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showBatchDeleteDialog} onOpenChange={setShowBatchDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete {selectedIds.size} Request{selectedIds.size !== 1 ? 's' : ''}</DialogTitle>
            <DialogDescription>
              Are you sure you want to permanently delete {selectedIds.size} selected request{selectedIds.size !== 1 ? 's' : ''}? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowBatchDeleteDialog(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleBatchDelete}
              disabled={batchDeleteMutation.isPending}
            >
              {batchDeleteMutation.isPending && (
                <Loader2 className="size-4 mr-2 animate-spin" />
              )}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {searchModal && (
        <SearchModal
          open={searchModal.open}
          onOpenChange={(open) => {
            if (!open) {
              handleSearchModalClose()
            }
          }}
          qualityProfileId={searchModal.qualityProfileId}
          movieId={searchModal.mediaType === 'movie' ? searchModal.mediaId : undefined}
          movieTitle={searchModal.mediaType === 'movie' ? searchModal.mediaTitle : undefined}
          tmdbId={searchModal.tmdbId}
          imdbId={searchModal.imdbId}
          year={searchModal.year}
          seriesId={searchModal.mediaType === 'series' ? searchModal.mediaId : undefined}
          seriesTitle={searchModal.mediaType === 'series' ? searchModal.mediaTitle : undefined}
          tvdbId={searchModal.tvdbId}
          season={searchModal.season}
        />
      )}
    </div>
  )
}

interface RequestRowProps {
  request: Request
  selected: boolean
  isProcessing: boolean
  onToggleSelect: () => void
  onApproveOnly: () => void
  onApproveAndManualSearch: () => void
  onApproveAndAutoSearch: () => void
  onDeny: () => void
  onDelete: () => void
}

function RequestRow({
  request,
  selected,
  isProcessing,
  onToggleSelect,
  onApproveOnly,
  onApproveAndManualSearch,
  onApproveAndAutoSearch,
  onDeny,
  onDelete,
}: RequestRowProps) {
  const statusConfig = STATUS_CONFIG[request.status]
  const isPending = request.status === 'pending'

  return (
    <div className="flex items-center gap-4 p-4 hover:bg-muted/40">
      <Checkbox checked={selected} onCheckedChange={onToggleSelect} />

      <div className="flex-shrink-0 w-10 h-15 rounded overflow-hidden">
        <PosterImage
          url={request.posterUrl}
          alt={request.title}
          type={request.mediaType === 'movie' ? 'movie' : 'series'}
          className="w-full h-full"
        />
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium truncate">{request.title}</span>
          {request.year && <span className="text-sm text-muted-foreground">({request.year})</span>}
        </div>
        <div className="flex items-center gap-2 text-sm text-muted-foreground flex-wrap">
          <Badge variant="outline" className="text-xs capitalize">{request.mediaType}</Badge>
          {request.mediaType === 'series' && (
            <>
              {request.requestedSeasons && request.requestedSeasons.length > 0 ? (
                <span>
                  {request.requestedSeasons.length <= 3
                    ? `S${request.requestedSeasons.join(', S')}`
                    : `${request.requestedSeasons.length} seasons`}
                </span>
              ) : (
                <span className="text-muted-foreground/70">No seasons</span>
              )}
              {request.monitorFuture && (
                <Badge variant="secondary" className="text-xs">Future</Badge>
              )}
            </>
          )}
          {request.seasonNumber && request.mediaType !== 'series' && <span>Season {request.seasonNumber}</span>}
          {request.episodeNumber && <span>Episode {request.episodeNumber}</span>}
          <span>•</span>
          <span>{formatDistanceToNow(new Date(request.createdAt), { addSuffix: true })}</span>
          {request.user && (
            <>
              <span>•</span>
              <span>by {request.user.displayName || request.user.username}</span>
            </>
          )}
        </div>
        {request.deniedReason && (
          <p className="text-sm text-red-500 mt-1">Reason: {request.deniedReason}</p>
        )}
      </div>

      <Badge className={`${statusConfig.color} text-white`}>
        {statusConfig.icon}
        <span className="ml-1">{statusConfig.label}</span>
      </Badge>

      <TooltipProvider>
        <div className="flex items-center gap-1">
          {isPending && (
            <>
              <Tooltip>
                <TooltipTrigger render={<Button variant="ghost" size="icon" onClick={onApproveOnly} disabled={isProcessing} />}>
                  {isProcessing ? <Loader2 className="size-4 animate-spin" /> : <CheckCircle className="size-4" />}
                </TooltipTrigger>
                <TooltipContent>Approve (add to library)</TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger render={<Button variant="ghost" size="icon" onClick={onApproveAndManualSearch} disabled={isProcessing} />}>
                  <Search className="size-4" />
                </TooltipTrigger>
                <TooltipContent>Approve & Manual Search</TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger render={<Button variant="ghost" size="icon" onClick={onApproveAndAutoSearch} disabled={isProcessing} />}>
                  <Zap className="size-4" />
                </TooltipTrigger>
                <TooltipContent>Approve & Auto Search</TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger render={<Button variant="ghost" size="icon" onClick={onDeny} disabled={isProcessing} />}>
                  <XCircle className="size-4 text-destructive" />
                </TooltipTrigger>
                <TooltipContent>Deny</TooltipContent>
              </Tooltip>
            </>
          )}

          <Tooltip>
            <TooltipTrigger render={<Button variant="ghost" size="icon" onClick={onDelete} />}>
              <Trash2 className="size-4 text-muted-foreground hover:text-destructive" />
            </TooltipTrigger>
            <TooltipContent>Delete permanently</TooltipContent>
          </Tooltip>
        </div>
      </TooltipProvider>
    </div>
  )
}
