import { FormProvider } from 'react-hook-form'

import { ArrowLeft, TestTube } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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

export function IndexerDialog({ open, onOpenChange, indexer }: IndexerDialogProps) {
  const hook = useIndexerDialog(open, indexer, onOpenChange)

  return (
    <FormProvider {...hook.form}>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent
          className={hook.step === 'select' ? 'h-[600px] sm:max-w-3xl' : 'h-[80vh] sm:max-w-2xl'}
        >
          <DialogHeader>
            <HeaderContent hook={hook} />
          </DialogHeader>

          {hook.step === 'select' && (
            <DialogBody className="overflow-hidden">
              <DefinitionSearchTable
                definitions={hook.definitions}
                isLoading={hook.isLoadingDefinitions}
                onSelect={hook.handleDefinitionSelect}
              />
            </DialogBody>
          )}

          {hook.step === 'configure' && !!hook.selectedDefinition && <ConfigureStep hook={hook} />}

          {hook.step === 'configure' && (
            <FooterActions hook={hook} onOpenChange={onOpenChange} />
          )}
        </DialogContent>
      </Dialog>
    </FormProvider>
  )
}

type HookValues = ReturnType<typeof useIndexerDialog>

function HeaderContent({ hook }: { hook: HookValues }) {
  let titleText: string
  let descriptionText: string

  if (hook.step === 'select') {
    titleText = 'Add Indexer'
    descriptionText = 'Select an indexer from the list below.'
  } else if (hook.isEditing) {
    titleText = 'Edit Indexer'
    descriptionText = 'Configure the indexer settings.'
  } else {
    titleText = 'Configure Indexer'
    descriptionText = 'Configure the indexer settings.'
  }

  return (
    <>
      <DialogTitle className="flex items-center gap-2">
        {hook.step === 'configure' && !hook.isEditing && (
          <Button variant="ghost" size="icon" aria-label="Back" className="size-6" onClick={hook.handleBack}>
            <ArrowLeft className="size-4" />
          </Button>
        )}
        {titleText}
      </DialogTitle>
      <DialogDescription>{descriptionText}</DialogDescription>
    </>
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
    <DialogFooter className="flex-col gap-2 sm:flex-row">
      <LoadingButton loading={hook.isTesting} icon={TestTube} variant="outline" onClick={hook.handleTest}>
        Test
      </LoadingButton>
      <div className="flex gap-2 sm:ml-auto">
        <Button variant="outline" onClick={() => onOpenChange(false)}>
          Cancel
        </Button>
        <LoadingButton loading={hook.isPending} onClick={hook.handleSubmit}>
          {hook.isEditing ? 'Save' : 'Add'}
        </LoadingButton>
      </div>
    </DialogFooter>
  )
}
