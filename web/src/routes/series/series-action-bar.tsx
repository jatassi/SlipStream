import { Edit, RefreshCw, Trash2 } from 'lucide-react'

import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { MediaSearchMonitorControls } from '@/components/search'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { Series } from '@/types'

type SeriesActionBarProps = {
  series: Series
  isRefreshing: boolean
  onToggleMonitored: (monitored?: boolean) => void
  onRefresh: () => void
  onEdit: () => void
  onDelete: () => void
}

export function SeriesActionBar({
  series,
  isRefreshing,
  onToggleMonitored,
  onRefresh,
  onEdit,
  onDelete,
}: SeriesActionBarProps) {
  return (
    <div className="bg-card flex flex-wrap gap-2 border-b px-6 py-4">
      <MediaSearchMonitorControls
        mediaType="series"
        seriesId={series.id}
        title={series.title}
        theme="tv"
        size="responsive"
        monitored={series.monitored}
        onMonitoredChange={onToggleMonitored}
        qualityProfileId={series.qualityProfileId}
        tvdbId={series.tvdbId}
        tmdbId={series.tmdbId}
        imdbId={series.imdbId}
      />
      <div className="ml-auto flex gap-2">
        <ResponsiveButton
          icon={RefreshCw}
          label="Refresh"
          onClick={onRefresh}
          disabled={isRefreshing}
        />
        <ResponsiveButton
          icon={Edit}
          label="Edit"
          onClick={onEdit}
        />
        <DeleteButton series={series} onDelete={onDelete} />
      </div>
    </div>
  )
}

type ResponsiveButtonProps = {
  icon: React.ComponentType<{ className?: string }>
  label: string
  onClick: () => void
  disabled?: boolean
}

function ResponsiveButton({ icon: Icon, label, onClick, disabled }: ResponsiveButtonProps) {
  return (
    <>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="outline"
              size="icon"
              className="min-[820px]:hidden"
              onClick={onClick}
              disabled={disabled}
            />
          }
        >
          <Icon className="size-4" />
        </TooltipTrigger>
        <TooltipContent>{label}</TooltipContent>
      </Tooltip>
      <Button
        variant="outline"
        className="hidden min-[820px]:inline-flex"
        onClick={onClick}
        disabled={disabled}
      >
        <Icon className="mr-2 size-4" />
        {label}
      </Button>
    </>
  )
}

type DeleteButtonProps = {
  series: Series
  onDelete: () => void
}

function DeleteButton({ series, onDelete }: DeleteButtonProps) {
  return (
    <ConfirmDialog
      trigger={
        <>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button variant="destructive" size="icon" className="min-[820px]:hidden" />
              }
            >
              <Trash2 className="size-4" />
            </TooltipTrigger>
            <TooltipContent>Delete</TooltipContent>
          </Tooltip>
          <Button variant="destructive" className="hidden min-[820px]:inline-flex">
            <Trash2 className="mr-2 size-4" />
            Delete
          </Button>
        </>
      }
      title="Delete series"
      description={`Are you sure you want to delete "${series.title}"? This action cannot be undone.`}
      confirmLabel="Delete"
      variant="destructive"
      onConfirm={onDelete}
    />
  )
}
