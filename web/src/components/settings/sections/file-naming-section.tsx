import { Save } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

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

export function FileNamingSection() {
  const {
    form,
    activeTab,
    setActiveTab,
    updateField,
    hasChanges,
    isLoading,
    isError,
    isSaving,
    refetch,
  } = useFileNamingSection()

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
            <TabsTrigger value="tv-naming">TV Naming</TabsTrigger>
            <TabsTrigger value="movie-naming">Movie Naming</TabsTrigger>
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

        <TabsContent value="tv-naming" className="mt-6 max-w-3xl space-y-6">
          <TvNamingTab form={form} updateField={updateField} />
        </TabsContent>

        <TabsContent value="movie-naming" className="mt-6 max-w-3xl space-y-6">
          <MovieNamingTab form={form} updateField={updateField} />
        </TabsContent>

        <TabsContent value="tokens" className="mt-6 max-w-4xl space-y-6">
          <TokenReferenceTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}
