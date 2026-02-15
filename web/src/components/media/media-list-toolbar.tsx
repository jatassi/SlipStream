import { Eye, EyeOff, Trash2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { QualityProfile } from '@/types'

type Props = {
  selectedCount: number
  totalCount: number
  qualityProfiles: QualityProfile[] | undefined
  isBulkUpdating: boolean
  onSelectAll: () => void
  onMonitor: (monitored: boolean) => void
  onChangeQualityProfile: (id: number) => void
  onDelete: () => void
  theme: 'movie' | 'tv'
}

export function MediaListToolbar(props: Props) {
  const { selectedCount, totalCount, qualityProfiles, isBulkUpdating, theme } = props
  const disabled = selectedCount === 0 || isBulkUpdating

  return (
    <div className={`mb-4 flex items-center gap-4 rounded-lg border p-3 ${theme === 'movie' ? 'bg-movie-500/10 border-movie-500/20' : 'bg-tv-500/10 border-tv-500/20'}`}>
      <div className="flex items-center gap-2">
        <Checkbox checked={selectedCount === totalCount && totalCount > 0} onCheckedChange={props.onSelectAll} />
        <span className="text-muted-foreground text-sm">{selectedCount} of {totalCount} selected</span>
      </div>
      <div className="ml-auto flex items-center gap-2">
        <Button variant="outline" size="sm" disabled={disabled} onClick={() => props.onMonitor(true)}>
          <Eye className="mr-1 size-4" />Monitor
        </Button>
        <Button variant="outline" size="sm" disabled={disabled} onClick={() => props.onMonitor(false)}>
          <EyeOff className="mr-1 size-4" />Unmonitor
        </Button>
        <Select value="" onValueChange={(v) => v && props.onChangeQualityProfile(Number(v))} disabled={disabled}>
          <SelectTrigger className="h-8 w-40 text-sm">Set Quality Profile</SelectTrigger>
          <SelectContent>
            {qualityProfiles?.map((p) => (
              <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="destructive" size="sm" disabled={selectedCount === 0} onClick={props.onDelete}>
          <Trash2 className="mr-1 size-4" />Delete
        </Button>
      </div>
    </div>
  )
}
