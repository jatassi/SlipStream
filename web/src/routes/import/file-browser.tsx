import { useState } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import { useGlobalLoading } from '@/hooks'
import { useBrowseForImport } from '@/hooks/use-filesystem'
import type { ScannedFile } from '@/types'

import { FileBrowserContent } from './file-browser-content'
import { FileBrowserHeader } from './file-browser-header'

type Props = {
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
}

export function FileBrowser(props: Props) {
  const [pathInput, setPathInput] = useState(props.currentPath)
  const globalLoading = useGlobalLoading()
  const { data, isLoading: queryLoading, refetch } = useBrowseForImport(props.currentPath || undefined)
  const isLoading = queryLoading || globalLoading
  const showScan = props.scannedFiles.length > 0

  const navigateTo = (path: string) => { props.onPathChange(path); setPathInput(path) }
  const navigateUp = () => { const t = data?.parent ?? ''; props.onPathChange(t); setPathInput(t) }

  return (
    <Card>
      <FileBrowserHeader
        showScanResults={showScan} scannedFiles={props.scannedFiles} selectedFiles={props.selectedFiles}
        isImporting={props.isImporting} isScanning={props.isScanning} isLoading={isLoading}
        currentPath={props.currentPath} pathInput={pathInput} onPathInputChange={setPathInput}
        onPathInputNavigate={() => pathInput && props.onPathChange(pathInput)}
        onImportSelected={props.onImportSelected} onClearScan={props.onClearScan}
        onRefresh={() => refetch()} onScanPath={props.onScanPath}
      />
      <CardContent>
        <FileBrowserContent
          showScanResults={showScan} isLoading={isLoading}
          scannedFiles={props.scannedFiles} selectedFiles={props.selectedFiles}
          onToggleFile={props.onToggleFile} onToggleAll={props.onToggleAll}
          onEditMatch={props.onEditMatch} onImportFile={props.onImportFile}
          data={data} currentPath={props.currentPath}
          onNavigateTo={navigateTo} onNavigateUp={navigateUp}
        />
      </CardContent>
    </Card>
  )
}
