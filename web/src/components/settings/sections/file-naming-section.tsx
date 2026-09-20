import { Save } from 'lucide-react'

import { SectionError, SectionLoading } from '@/components/settings/section-state'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getEnabledModules } from '@/modules'

import { MatchingTab } from './naming-matching-tab'
import { MovieNamingTab } from './naming-movie-tab'
import { TokenReferenceTab } from './naming-token-reference-tab'
import { TvNamingTab } from './naming-tv-tab'
import { ValidationTab } from './naming-validation-tab'
import { useFileNamingSection } from './use-file-naming-section'

function SaveStatus({ isSaving, hasChanges }: { isSaving: boolean; hasChanges: boolean }) {
  if (isSaving) {
    return (
      <span className="text-footnote flex items-center gap-2 text-muted-foreground">
        <Save className="size-4 animate-pulse" />
        Saving...
      </span>
    )
  }
  return (
    <span className="text-footnote flex items-center gap-2 text-muted-foreground">
      <Save className="size-4" />
      {hasChanges ? 'Unsaved changes' : 'All changes saved'}
    </span>
  )
}

const MODULE_NAMING_TABS: Partial<Record<string, React.ComponentType>> = {
  movie: MovieNamingTab,
  tv: TvNamingTab,
}

function ModuleNamingTab({ moduleId }: { moduleId: string }) {
  const Tab = MODULE_NAMING_TABS[moduleId]
  if (!Tab) {
    return null
  }
  return <Tab />
}

export function FileNamingSection() {
  const { form, activeTab, setActiveTab, updateField, hasChanges, isLoading, isError, isSaving, refetch } =
    useFileNamingSection()
  const modules = getEnabledModules()

  if (isLoading) {
    return <SectionLoading count={3} />
  }
  if (isError || !form) {
    return <SectionError onRetry={refetch} />
  }

  const isImportTab = activeTab === 'validation' || activeTab === 'matching'

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab}>
      <div className="px-screen mb-4 flex flex-wrap items-center justify-between gap-3">
        <TabsList className="max-w-full overflow-x-auto">
          <TabsTrigger value="validation">Validation</TabsTrigger>
          <TabsTrigger value="matching">Matching</TabsTrigger>
          {modules.map((mod) => (
            <TabsTrigger key={mod.id} value={`${mod.id}-naming`}>
              {mod.singularName} Naming
            </TabsTrigger>
          ))}
          <TabsTrigger value="tokens">Token Reference</TabsTrigger>
        </TabsList>
        {isImportTab ? <SaveStatus isSaving={isSaving} hasChanges={!!hasChanges} /> : null}
      </div>
      <TabsContent value="validation">
        <ValidationTab form={form} updateField={updateField} />
      </TabsContent>
      <TabsContent value="matching">
        <MatchingTab form={form} updateField={updateField} />
      </TabsContent>
      {modules.map((mod) => (
        <TabsContent key={mod.id} value={`${mod.id}-naming`}>
          <ModuleNamingTab moduleId={mod.id} />
        </TabsContent>
      ))}
      <TabsContent value="tokens">
        <TokenReferenceTab />
      </TabsContent>
    </Tabs>
  )
}
