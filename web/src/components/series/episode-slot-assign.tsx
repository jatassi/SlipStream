import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { seriesKeys, useAssignEpisodeFile } from '@/hooks'
import type { Episode, Slot } from '@/types'

import { useSeriesInfo } from './series-context'

function episodeCode(episode: Episode): string {
  return `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`
}

function useAssignSlot(episode: Episode) {
  const { seriesId } = useSeriesInfo()
  const queryClient = useQueryClient()
  const mutation = useAssignEpisodeFile()

  const assign = async (fileId: number, slotId: number) => {
    try {
      await mutation.mutateAsync({ episodeId: episode.id, slotId, data: { fileId } })
      void queryClient.invalidateQueries({ queryKey: [...seriesKeys.detail(seriesId), 'episodes'] })
      toast.success('File assigned to slot')
    } catch {
      toast.error('Failed to assign file to slot')
    }
  }

  return { assign, isPending: mutation.isPending }
}

/** The episode file's version-slot assignment, inside the episode row's slot panel. */
export function EpisodeSlotAssign({ episode, slots }: { episode: Episode; slots: Slot[] }) {
  const { assign, isPending } = useAssignSlot(episode)
  const file = episode.episodeFile

  if (!file) {
    return null
  }

  return (
    <div className="min-h-tap flex items-center justify-between gap-3 pb-2">
      <span className="text-footnote text-muted-foreground">File slot</span>
      <Select
        value={file.slotId?.toString() ?? 'unassigned'}
        onValueChange={(value) => {
          if (value !== null && value !== 'unassigned') {
            void assign(file.id, Number.parseInt(value, 10))
          }
        }}
        disabled={isPending}
      >
        <SelectTrigger className="h-8 w-32" aria-label={`Slot for ${episodeCode(episode)}`}>
          {slots.find((slot) => slot.id === file.slotId)?.name ?? (
            <span className="text-muted-foreground">Unassigned</span>
          )}
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="unassigned" disabled>
            Unassigned
          </SelectItem>
          {slots.map((slot) => (
            <SelectItem key={slot.id} value={slot.id.toString()}>
              {slot.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
