import { FormProvider } from 'react-hook-form'

import { TestTube } from 'lucide-react'

import { FormActions, SheetPresenter } from '@/components/presenter'
import { LoadingButton } from '@/components/ui/loading-button'
import type { Indexer } from '@/types'

import { ConfigureStep } from './configure-step'
import { DefinitionSearchTable } from './definition-search-table'
import { useIndexerDialog } from './use-indexer-dialog'

type IndexerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  indexer?: Indexer | null
}

type HookValues = ReturnType<typeof useIndexerDialog>

function titleFor(hook: HookValues): string {
  if (hook.step === 'select') {
    return 'Add Indexer'
  }
  return hook.isEditing ? 'Edit Indexer' : 'Configure Indexer'
}

export function IndexerDialog({ open, onOpenChange, indexer }: IndexerDialogProps) {
  const hook = useIndexerDialog(open, indexer, onOpenChange)
  const isSelect = hook.step === 'select'

  return (
    <FormProvider {...hook.form}>
      <SheetPresenter
        open={open}
        onOpenChange={onOpenChange}
        title={titleFor(hook)}
        description={
          isSelect ? 'Select an indexer from the list below.' : 'Configure the indexer settings.'
        }
        wideClassName={isSelect ? 'h-[600px] sm:max-w-3xl' : 'h-[80vh] sm:max-w-2xl'}
        footer={isSelect ? undefined : <FooterActions hook={hook} onOpenChange={onOpenChange} />}
      >
        {isSelect ? (
          <DefinitionSearchTable
            definitions={hook.definitions}
            isLoading={hook.isLoadingDefinitions}
            onSelect={hook.handleDefinitionSelect}
          />
        ) : (
          <ConfigureStep hook={hook} onBack={hook.isEditing ? undefined : hook.handleBack} />
        )}
      </SheetPresenter>
    </FormProvider>
  )
}

function FooterActions({
  hook,
  onOpenChange,
}: {
  hook: HookValues
  onOpenChange: (open: boolean) => void
}) {
  return (
    <FormActions
      leading={
        <LoadingButton
          className="min-h-tap"
          loading={hook.isTesting}
          icon={TestTube}
          variant="outline"
          onClick={hook.handleTest}
        >
          Test
        </LoadingButton>
      }
      onCancel={() => onOpenChange(false)}
      confirmLabel={hook.isEditing ? 'Save' : 'Add'}
      onConfirm={hook.handleSubmit}
      loading={hook.isPending}
    />
  )
}
