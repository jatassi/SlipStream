import { useState } from 'react'

import { Import } from 'lucide-react'
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
import type { ScannedFile } from '@/types'

import { formatFileSize } from './format-file-size'
import { MatchToLibraryForm } from './match-to-library-form'
import { ParsedInfoPanel } from './parsed-info-panel'
import type { MatchParams } from './use-import-page'

function resolveInitialType(file: ScannedFile): 'movie' | 'episode' {
  if (file.suggestedMatch?.mediaType === 'movie') {
    return 'movie'
  }
  if (file.parsedInfo?.isTV) {
    return 'episode'
  }
  return 'movie'
}

type FormState = {
  type: 'movie' | 'episode'
  movieId: string
  episodeId: string
  seriesId: string
  slotId: string
}

function buildMatchParams(state: FormState, season?: number): MatchParams | null {
  const parsedSlotId = state.slotId ? Number.parseInt(state.slotId) : undefined

  if (state.type === 'movie') {
    if (!state.movieId) {
      toast.error('Please select a movie')
      return null
    }
    return { mediaType: 'movie', mediaId: Number.parseInt(state.movieId), targetSlotId: parsedSlotId }
  }

  if (!state.episodeId) {
    toast.error('Please select an episode')
    return null
  }
  return {
    mediaType: 'episode',
    mediaId: Number.parseInt(state.episodeId),
    seriesId: state.seriesId ? Number.parseInt(state.seriesId) : undefined,
    seasonNum: season,
    targetSlotId: parsedSlotId,
  }
}

function useDialogFormState(file: ScannedFile) {
  const [selectedType, setSelectedType] = useState<'movie' | 'episode'>(resolveInitialType(file))
  const [selectedMovieId, setSelectedMovieId] = useState(
    file.suggestedMatch?.mediaType === 'movie' ? String(file.suggestedMatch.mediaId) : '',
  )
  const [selectedSeriesId, setSelectedSeriesId] = useState(
    file.suggestedMatch?.seriesId ? String(file.suggestedMatch.seriesId) : '',
  )
  const [selectedEpisodeId, setSelectedEpisodeId] = useState(
    file.suggestedMatch?.mediaType === 'episode' ? String(file.suggestedMatch.mediaId) : '',
  )
  const [selectedSlotId, setSelectedSlotId] = useState('')

  return {
    selectedType, setSelectedType,
    selectedMovieId, setSelectedMovieId,
    selectedSeriesId, setSelectedSeriesId,
    selectedEpisodeId, setSelectedEpisodeId,
    selectedSlotId, setSelectedSlotId,
  }
}

function EditMatchDialogContent({ file, onClose, onConfirm }: {
  file: ScannedFile
  onClose: () => void
  onConfirm: (file: ScannedFile, match: MatchParams) => void
}) {
  const s = useDialogFormState(file)

  const handleConfirm = () => {
    const params = buildMatchParams(
      { type: s.selectedType, movieId: s.selectedMovieId, episodeId: s.selectedEpisodeId, seriesId: s.selectedSeriesId, slotId: s.selectedSlotId },
      file.parsedInfo?.season,
    )
    if (params) {
      onConfirm(file, params)
    }
  }

  return (
    <>
      <div className="space-y-4 py-4">
        <div className="bg-muted rounded-lg p-3">
          <p className="text-sm font-medium break-all">{file.fileName}</p>
          <p className="text-muted-foreground mt-1 text-xs">{formatFileSize(file.fileSize)}</p>
        </div>
        {file.parsedInfo ? <ParsedInfoPanel parsed={file.parsedInfo} /> : null}
        <MatchToLibraryForm
          selectedType={s.selectedType} onTypeChange={s.setSelectedType}
          selectedMovieId={s.selectedMovieId} onMovieChange={s.setSelectedMovieId}
          selectedSeriesId={s.selectedSeriesId} onSeriesChange={s.setSelectedSeriesId}
          selectedEpisodeId={s.selectedEpisodeId} onEpisodeChange={s.setSelectedEpisodeId}
          selectedSlotId={s.selectedSlotId} onSlotChange={s.setSelectedSlotId}
        />
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Cancel</Button>
        <Button onClick={handleConfirm}><Import className="mr-2 size-4" />Import</Button>
      </DialogFooter>
    </>
  )
}

export function EditMatchDialog({ file, open, onClose, onConfirm }: {
  file: ScannedFile | null
  open: boolean
  onClose: () => void
  onConfirm: (file: ScannedFile, match: MatchParams) => void
}) {
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Edit Import Match</DialogTitle>
          <DialogDescription>Review parsed information and select the library item to import as</DialogDescription>
        </DialogHeader>
        {file ? <EditMatchDialogContent file={file} onClose={onClose} onConfirm={onConfirm} /> : null}
      </DialogContent>
    </Dialog>
  )
}
