import { HardDrive } from 'lucide-react'

import { Group, IconTile } from '@/components/grouped-list'
import { Skeleton } from '@/components/ui/skeleton'
import { useRootFolders } from '@/hooks/use-root-folders'
import { useStorage } from '@/hooks/use-storage'
import { formatBytes } from '@/lib/formatters'
import { useUIStore } from '@/stores'
import type { RootFolder } from '@/types/root-folder'
import type { RootFolderRef, StorageInfo } from '@/types/storage'

function uniqueFolders(volumes: StorageInfo[]): RootFolderRef[] {
  const seen = new Set<number>()
  const folders: RootFolderRef[] = []
  for (const volume of volumes) {
    for (const folder of volume.rootFolders ?? []) {
      if (seen.has(folder.id)) {
        continue
      }
      seen.add(folder.id)
      folders.push(folder)
    }
  }
  return folders
}

function totals(volumes: StorageInfo[]): { used: number; total: number } {
  return volumes.reduce(
    (acc, volume) => ({ used: acc.used + volume.usedSpace, total: acc.total + volume.totalSpace }),
    { used: 0, total: 0 },
  )
}

function splitPercents(folders: RootFolderRef[], usedPct: number): { movie: number; tv: number } {
  const movieCount = folders.filter((f) => f.mediaType === 'movie').length
  const tvCount = folders.filter((f) => f.mediaType === 'tv').length
  const weight = movieCount + tvCount
  if (weight === 0) {
    return { movie: 0, tv: 0 }
  }
  return {
    movie: usedPct * (movieCount / weight),
    tv: usedPct * (tvCount / weight),
  }
}

function SplitBar({ movie, tv }: { movie: number; tv: number }) {
  return (
    <div className="mt-2 flex h-1.5 w-full gap-0.5 overflow-hidden rounded-full bg-foreground/10">
      {movie > 0 && <div className="h-full rounded-l-full bg-movie-500" style={{ width: `${movie}%` }} />}
      {tv > 0 && <div className="h-full rounded-r-full bg-tv-500" style={{ width: `${tv}%` }} />}
    </div>
  )
}

function FolderLegend({
  folders,
  freeSpaceById,
  usedPct,
}: {
  folders: RootFolderRef[]
  freeSpaceById: Record<number, number>
  usedPct: number
}) {
  return (
    <div className="text-caption mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-muted-foreground">
      {folders.map((folder) => (
        <span key={folder.id}>
          <span className={folder.mediaType === 'movie' ? 'text-movie-400' : 'text-tv-400'}>●</span>{' '}
          {folder.mediaType === 'movie' ? 'Movies' : 'Series'} {folder.name}
          <span className="nums"> {formatBytes(freeSpaceById[folder.id] ?? 0)} free</span>
        </span>
      ))}
      <span className="nums ml-auto">{usedPct.toFixed(0)}%</span>
    </div>
  )
}

function StorageBody({
  volumes,
  freeSpaceById,
}: {
  volumes: StorageInfo[]
  freeSpaceById: Record<number, number>
}) {
  const { used, total } = totals(volumes)
  const folders = uniqueFolders(volumes)
  const usedPct = total > 0 ? (used / total) * 100 : 0
  const split = splitPercents(folders, usedPct)
  return (
    <div className="px-4 py-3">
      <div className="flex items-center gap-3">
        <IconTile className="bg-zinc-600">
          <HardDrive />
        </IconTile>
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between gap-3">
            <span className="text-body font-medium">{formatBytes(used)} used</span>
            <span className="nums text-footnote text-muted-foreground">of {formatBytes(total)}</span>
          </div>
          <SplitBar movie={split.movie} tv={split.tv} />
          <FolderLegend folders={folders} freeSpaceById={freeSpaceById} usedPct={usedPct} />
        </div>
      </div>
    </div>
  )
}

function StorageSkeleton() {
  return (
    <div role="status" aria-label="Loading" className="px-4 py-3">
      <div className="flex items-center gap-3">
        <Skeleton className="size-7 rounded-[7px]" />
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between">
            <Skeleton className="h-5 w-28" />
            <Skeleton className="h-[18px] w-16" />
          </div>
          <Skeleton className="mt-2 h-1.5 w-full rounded-full" />
          <Skeleton className="mt-1.5 h-4 w-48" />
        </div>
      </div>
    </div>
  )
}

function freeSpaceByFolderId(folders: RootFolder[] | undefined): Record<number, number> {
  const map: Record<number, number> = {}
  for (const folder of folders ?? []) {
    map[folder.id] = folder.freeSpace
  }
  return map
}

export function StorageGroup() {
  const globalLoading = useUIStore((s) => s.globalLoading)
  const { data, isLoading } = useStorage()
  const { data: rootFolders, isLoading: foldersLoading } = useRootFolders()

  if (isLoading || foldersLoading || globalLoading) {
    return (
      <Group header="Storage" inset={false} className="mb-0">
        <StorageSkeleton />
      </Group>
    )
  }

  return (
    <Group header="Storage" inset={false} className="mb-0">
      <StorageBody volumes={data ?? []} freeSpaceById={freeSpaceByFolderId(rootFolders)} />
    </Group>
  )
}
