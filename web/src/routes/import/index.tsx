import { PageHeader } from '@/components/layout/page-header'

import { EditMatchDialog } from './edit-match-dialog'
import { FileBrowser } from './file-browser'
import { PendingImportsCard } from './pending-imports-card'
import { useImportPage } from './use-import-page'

export function ManualImportPage() {
  const {
    currentPath,
    setCurrentPath,
    scannedFiles,
    selectedFiles,
    importDialogFile,
    isScanning,
    isImporting,
    handleScanPath,
    handleToggleFile,
    handleToggleAll,
    handleImportSelected,
    handleEditMatch,
    handleConfirmImport,
    handleClearScan,
    handleDirectImport,
    closeDialog,
  } = useImportPage()

  return (
    <div>
      <PageHeader title="Manual Import" description="Browse and import media files manually" />

      <div className="space-y-6">
        <FileBrowser
          currentPath={currentPath}
          onPathChange={setCurrentPath}
          onScanPath={handleScanPath}
          isScanning={isScanning}
          scannedFiles={scannedFiles}
          selectedFiles={selectedFiles}
          onToggleFile={handleToggleFile}
          onToggleAll={handleToggleAll}
          onEditMatch={handleEditMatch}
          onImportFile={handleDirectImport}
          onClearScan={handleClearScan}
          onImportSelected={handleImportSelected}
          isImporting={isImporting}
        />

        <PendingImportsCard />
      </div>

      <EditMatchDialog
        key={importDialogFile?.path}
        file={importDialogFile}
        open={!!importDialogFile}
        onClose={closeDialog}
        onConfirm={handleConfirmImport}
      />
    </div>
  )
}
