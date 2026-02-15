import { ChevronRight, Loader2, RefreshCw, Scan } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import type { ScannedFile } from '@/types'

function ScanResultActions({ selectedCount, isImporting, onImportSelected, onClearScan }: {
  selectedCount: number
  isImporting: boolean
  onImportSelected: () => void
  onClearScan: () => void
}) {
  return (
    <>
      {selectedCount > 0 && (
        <Button size="sm" onClick={onImportSelected} disabled={isImporting}>
          {isImporting ? 'Importing...' : `Import ${selectedCount} Selected`}
        </Button>
      )}
      <Button size="sm" variant="outline" onClick={onClearScan}>Back to Browser</Button>
    </>
  )
}

function BrowseActions({ isLoading, isScanning, currentPath, onRefresh, onScanPath }: {
  isLoading: boolean
  isScanning: boolean
  currentPath: string
  onRefresh: () => void
  onScanPath: (path: string) => void
}) {
  const scanIcon = isScanning
    ? <Loader2 className="mr-2 size-4 animate-spin" />
    : <Scan className="mr-2 size-4" />

  return (
    <>
      <Button size="sm" variant="outline" onClick={onRefresh} disabled={isLoading}>
        <RefreshCw className="size-4" />
      </Button>
      {currentPath ? (
        <Button size="sm" onClick={() => onScanPath(currentPath)} disabled={isScanning || isLoading}>
          {scanIcon}
          Scan Directory
        </Button>
      ) : null}
    </>
  )
}

function PathInput({ pathInput, onPathInputChange, onPathInputNavigate }: {
  pathInput: string
  onPathInputChange: (value: string) => void
  onPathInputNavigate: () => void
}) {
  return (
    <div className="mt-2 flex gap-2">
      <Input
        placeholder="Enter path..."
        value={pathInput}
        onChange={(e) => onPathInputChange(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && onPathInputNavigate()}
        className="h-8 font-mono text-xs"
      />
      <Button size="sm" variant="outline" onClick={onPathInputNavigate} className="h-8 px-2">
        <ChevronRight className="size-4" />
      </Button>
    </div>
  )
}

function HeaderTitle({ showScanResults, scannedFiles }: {
  showScanResults: boolean
  scannedFiles: ScannedFile[]
}) {
  const readyCount = scannedFiles.filter((f) => f.suggestedMatch).length
  return (
    <div>
      <CardTitle className="text-base">{showScanResults ? 'Scanned Files' : 'File Browser'}</CardTitle>
      {showScanResults ? (
        <CardDescription>{scannedFiles.length} files found, {readyCount} ready to import</CardDescription>
      ) : null}
    </div>
  )
}

export function FileBrowserHeader({
  showScanResults, scannedFiles, selectedFiles, isImporting, isScanning,
  isLoading, currentPath, pathInput, onPathInputChange, onPathInputNavigate,
  onImportSelected, onClearScan, onRefresh, onScanPath,
}: {
  showScanResults: boolean
  scannedFiles: ScannedFile[]
  selectedFiles: Set<string>
  isImporting: boolean
  isScanning: boolean
  isLoading: boolean
  currentPath: string
  pathInput: string
  onPathInputChange: (value: string) => void
  onPathInputNavigate: () => void
  onImportSelected: () => void
  onClearScan: () => void
  onRefresh: () => void
  onScanPath: (path: string) => void
}) {
  return (
    <CardHeader className="pb-3">
      <div className="flex items-center justify-between">
        <HeaderTitle showScanResults={showScanResults} scannedFiles={scannedFiles} />
        <div className="flex gap-2">
          {showScanResults ? (
            <ScanResultActions selectedCount={selectedFiles.size} isImporting={isImporting} onImportSelected={onImportSelected} onClearScan={onClearScan} />
          ) : (
            <BrowseActions isLoading={isLoading} isScanning={isScanning} currentPath={currentPath} onRefresh={onRefresh} onScanPath={onScanPath} />
          )}
        </div>
      </div>
      {!showScanResults && <PathInput pathInput={pathInput} onPathInputChange={onPathInputChange} onPathInputNavigate={onPathInputNavigate} />}
    </CardHeader>
  )
}
