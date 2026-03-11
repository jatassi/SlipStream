import { Suspense } from 'react'

import { LoadingState } from '@/components/data/loading-state'
import { PageHeader } from '@/components/layout/page-header'
import { getModuleOrThrow } from '@/modules'

function ModulePageContent({ moduleId }: { moduleId: string }) {
  const mod = getModuleOrThrow(moduleId)
  const ListComponent = mod.listComponent
  return <ListComponent />
}

export function ModuleListPage({ moduleId }: { moduleId: string }) {
  const mod = getModuleOrThrow(moduleId)

  return (
    <Suspense fallback={<div><PageHeader title={mod.name} /><LoadingState /></div>}>
      <ModulePageContent moduleId={moduleId} />
    </Suspense>
  )
}
