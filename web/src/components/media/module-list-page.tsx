import { lazy, Suspense } from 'react'

import { LoadingState } from '@/components/data/loading-state'
import { PageHeader } from '@/components/layout/page-header'
import { getModuleOrThrow } from '@/modules'

const MovieListPage = lazy(() => import('@/routes/movies').then((m) => ({ default: m.MoviesPage })))
const SeriesListPage = lazy(() => import('@/routes/series').then((m) => ({ default: m.SeriesListPage })))

function ModulePageContent({ moduleId }: { moduleId: string }) {
  if (moduleId === 'movie') { return <MovieListPage /> }
  if (moduleId === 'tv') { return <SeriesListPage /> }
  throw new Error(`No list page registered for module "${moduleId}"`)
}

export function ModuleListPage({ moduleId }: { moduleId: string }) {
  const mod = getModuleOrThrow(moduleId)

  return (
    <Suspense fallback={<div><PageHeader title={mod.name} /><LoadingState /></div>}>
      <ModulePageContent moduleId={moduleId} />
    </Suspense>
  )
}
