import { useNavigate, useRouter, useSearch } from '@tanstack/react-router'

import { usePushBack } from '@/components/layout/use-push-back'
import { useBrowseForImport, useDirectoryScan } from '@/hooks'
import { useUIStore } from '@/stores'
import type { ScannedFile } from '@/types'

import { folderName } from './folder-path'

export type BrowserFolder = { name: string; path: string }
export type BrowserFile = { name: string; path: string; size: number }

type BrowseData = {
  parent?: string
  drives?: { letter: string; label?: string }[]
  directories?: { name: string; path: string }[]
  files?: BrowserFile[]
}

type BackControl = { label: string; onClick: () => void }

function driveFolder(drive: { letter: string; label?: string }): BrowserFolder {
  return {
    name: drive.label === undefined ? drive.letter : `${drive.letter} (${drive.label})`,
    path: `${drive.letter}\\`,
  }
}

function foldersOf(data: BrowseData | undefined): BrowserFolder[] {
  const drives = data?.drives ?? []
  const directories = data?.directories ?? []
  return [
    ...drives.map((drive) => driveFolder(drive)),
    ...directories.map((dir) => ({ name: dir.name, path: dir.path })),
  ]
}

function scannedByPath(files: ScannedFile[]): Map<string, ScannedFile> {
  return new Map(files.map((file) => [file.path, file]))
}

function useSearchPath(): string {
  const search = useSearch({ strict: false })
  return typeof search.path === 'string' ? search.path : ''
}

function useBackControl(parent: string): BackControl | undefined {
  const router = useRouter()
  const pushBack = usePushBack()
  if (parent === '') {
    return pushBack
  }
  return {
    label: folderName(parent),
    onClick: () => {
      router.history.back()
    },
  }
}

export function useImportBrowser() {
  const path = useSearchPath()
  const navigate = useNavigate()
  const globalLoading = useUIStore((s) => s.globalLoading)
  const browse = useBrowseForImport(path === '' ? undefined : path)
  const data: BrowseData | undefined = browse.data
  const files = data?.files ?? []
  const scan = useDirectoryScan(path, files.length > 0)
  const back = useBackControl(data?.parent ?? '')

  return {
    path,
    title: path === '' ? 'Manual Import' : folderName(path),
    back,
    folders: foldersOf(data),
    files,
    matches: scannedByPath(scan.data?.files ?? []),
    matching: scan.isFetching,
    isLoading: browse.isLoading || globalLoading,
    open: (next: string) => {
      void navigate({ to: '/import', search: { path: next } })
    },
  }
}
