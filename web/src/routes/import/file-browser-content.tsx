import { ScrollArea } from '@/components/ui/scroll-area'
import type { ScannedFile } from '@/types'

import type { BrowseData } from './directory-listing'
import { DirectoryListing } from './directory-listing'
import { FileBrowserSkeleton } from './file-browser-skeleton'
import { ScannedFilesList } from './scanned-files-list'

type Props = {
  showScanResults: boolean
  isLoading: boolean
  scannedFiles: ScannedFile[]
  selectedFiles: Set<string>
  onToggleFile: (path: string) => void
  onToggleAll: () => void
  onEditMatch: (file: ScannedFile) => void
  onImportFile: (file: ScannedFile) => void
  data: BrowseData | undefined
  currentPath: string
  onNavigateTo: (path: string) => void
  onNavigateUp: () => void
}

export function FileBrowserContent(props: Props) {
  if (props.showScanResults) {
    return (
      <ScrollArea className="h-[500px]">
        <ScannedFilesList
          files={props.scannedFiles}
          selectedFiles={props.selectedFiles}
          onToggleFile={props.onToggleFile}
          onToggleAll={props.onToggleAll}
          onEditMatch={props.onEditMatch}
          onImportFile={props.onImportFile}
        />
      </ScrollArea>
    )
  }

  if (props.isLoading) {
    return <FileBrowserSkeleton />
  }

  return (
    <DirectoryListing
      data={props.data}
      currentPath={props.currentPath}
      onNavigateTo={props.onNavigateTo}
      onNavigateUp={props.onNavigateUp}
    />
  )
}
