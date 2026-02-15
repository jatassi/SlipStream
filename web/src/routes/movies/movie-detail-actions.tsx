import { Edit, RefreshCw, Trash2 } from 'lucide-react'

import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { MediaSearchMonitorControls } from '@/components/search'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { Movie } from '@/types'

type MovieDetailActionsProps = {
  movie: Movie
  isRefreshing: boolean
  onToggleMonitored: (monitored?: boolean) => void
  onRefresh: () => void
  onEdit: () => void
  onDelete: () => void
}

export function MovieDetailActions({
  movie,
  isRefreshing,
  onToggleMonitored,
  onRefresh,
  onEdit,
  onDelete,
}: MovieDetailActionsProps) {
  return (
    <div className="bg-card flex flex-wrap gap-2 border-b px-6 py-4">
      <MediaSearchMonitorControls
        mediaType="movie"
        movieId={movie.id}
        title={movie.title}
        theme="movie"
        size="responsive"
        monitored={movie.monitored}
        onMonitoredChange={onToggleMonitored}
        qualityProfileId={movie.qualityProfileId}
        tmdbId={movie.tmdbId}
        imdbId={movie.imdbId}
        year={movie.year}
      />
      <div className="ml-auto flex gap-2">
        <ResponsiveButton
          icon={<RefreshCw className="size-4" />}
          label="Refresh"
          onClick={onRefresh}
          disabled={isRefreshing}
        />
        <ResponsiveButton icon={<Edit className="size-4" />} label="Edit" onClick={onEdit} />
        <DeleteButton title={movie.title} onConfirm={onDelete} />
      </div>
    </div>
  )
}

function DeleteButton({ title, onConfirm }: { title: string; onConfirm: () => void }) {
  return (
    <ConfirmDialog
      trigger={
        <>
          <Tooltip>
            <TooltipTrigger
              render={<Button variant="destructive" size="icon" className="min-[820px]:hidden" />}
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
      title="Delete movie"
      description={`Are you sure you want to delete "${title}"? This action cannot be undone.`}
      confirmLabel="Delete"
      variant="destructive"
      onConfirm={onConfirm}
    />
  )
}

function ResponsiveButton({
  icon,
  label,
  onClick,
  disabled,
}: {
  icon: React.ReactNode
  label: string
  onClick: () => void
  disabled?: boolean
}) {
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
          {icon}
        </TooltipTrigger>
        <TooltipContent>{label}</TooltipContent>
      </Tooltip>
      <Button
        variant="outline"
        className="hidden min-[820px]:inline-flex"
        onClick={onClick}
        disabled={disabled}
      >
        <span className="mr-2">{icon}</span>
        {label}
      </Button>
    </>
  )
}
