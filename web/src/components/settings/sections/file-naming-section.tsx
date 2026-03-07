import { Save } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getEnabledModules } from '@/modules'
import type { ImportSettings } from '@/types'

import { MatchingTab } from './naming-matching-tab'
import { MovieNamingTab } from './naming-movie-tab'
import { TokenReferenceTab } from './naming-token-reference-tab'
import { TvNamingTab } from './naming-tv-tab'
import { ValidationTab } from './naming-validation-tab'
import { useFileNamingSection } from './use-file-naming-section'

function SaveStatus({ isSaving, hasChanges }: { isSaving: boolean; hasChanges: boolean }) {
  if (isSaving) {
    return (
      <span className="text-muted-foreground flex items-center gap-2 text-sm">
        <Save className="size-4 animate-pulse" />
        Saving...
      </span>
    )
  }
  return (
    <span className="text-muted-foreground flex items-center gap-2 text-sm">
      <Save className="size-4" />
      {hasChanges ? 'Unsaved changes' : 'All changes saved'}
    </span>
  )
}

type NamingTabProps = {
  form: ImportSettings
  updateField: <K extends keyof ImportSettings>(field: K, value: ImportSettings[K]) => void
}

const MODULE_NAMING_TABS: Partial<Record<string, React.ComponentType<NamingTabProps>>> = {
  movie: MovieNamingTab,
  tv: TvNamingTab,
}

function ModuleNamingTab({ moduleId, form, updateField }: NamingTabProps & { moduleId: string }) {
  const Tab = MODULE_NAMING_TABS[moduleId]
  if (!Tab) {
    return null
  }
  return <Tab form={form} updateField={updateField} />
}

export function FileNamingSection() {
  const { form, activeTab, setActiveTab, updateField, hasChanges, isLoading, isError, isSaving, refetch } =
    useFileNamingSection()
  const modules = getEnabledModules()

  if (isLoading) {
    return <LoadingState variant="list" count={3} />
  }
  if (isError || !form) {
    return <ErrorState onRetry={refetch} />
  }

  return (
    <div className="space-y-6">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <div className="flex items-center justify-between gap-4">
          <TabsList>
            <TabsTrigger value="validation">Validation</TabsTrigger>
            <TabsTrigger value="matching">Matching</TabsTrigger>
            {modules.map((mod) => (
              <TabsTrigger key={mod.id} value={`${mod.id}-naming`}>
                {mod.singularName} Naming
              </TabsTrigger>
            ))}
            <TabsTrigger value="tokens">Token Reference</TabsTrigger>
          </TabsList>
          <SaveStatus isSaving={isSaving} hasChanges={!!hasChanges} />
        </div>
        <TabsContent value="validation" className="mt-6 max-w-2xl space-y-6">
          <ValidationTab form={form} updateField={updateField} />
        </TabsContent>
        <TabsContent value="matching" className="mt-6 max-w-2xl space-y-6">
          <MatchingTab form={form} updateField={updateField} />
        </TabsContent>
        {modules.map((mod) => (
          <TabsContent key={mod.id} value={`${mod.id}-naming`} className="mt-6 max-w-3xl space-y-6">
            <ModuleNamingTab moduleId={mod.id} form={form} updateField={updateField} />
          </TabsContent>
        ))}
        <TabsContent value="tokens" className="mt-6 max-w-4xl space-y-6">
          <TokenReferenceTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}
