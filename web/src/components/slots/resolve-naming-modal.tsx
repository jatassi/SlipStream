import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { ImportSettings, MissingTokenInfo } from '@/types'

import { PatternEditor } from './pattern-editor'
import { TokenReferenceList } from './token-reference-list'
import { useResolveNamingModal } from './use-resolve-naming-modal'

type ResolveNamingModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  missingMovieTokens?: MissingTokenInfo[]
  missingEpisodeTokens?: MissingTokenInfo[]
  onResolved: () => void
}

function MissingTokensHint({ tokens }: { tokens: Set<string> }) {
  if (tokens.size === 0) {
    return null
  }
  return (
    <span className="mt-2 block font-medium text-orange-600 dark:text-orange-400">
      Suggested tokens to add: {[...tokens].join(', ')}
    </span>
  )
}

function EpisodeFormatsColumn({
  form,
  updateField,
  missingInStandard,
  missingInDaily,
  missingInAnime,
}: {
  form: Partial<ImportSettings>
  updateField: (field: keyof ImportSettings, value: string) => void
  missingInStandard: string[]
  missingInDaily: string[]
  missingInAnime: string[]
}) {
  return (
    <div className="space-y-4">
      <h3 className="border-b pb-2 text-sm font-medium">Episode Formats</h3>
      <PatternEditor
        label="Standard Episode Format"
        value={form.standardEpisodeFormat ?? ''}
        onChange={(v) => updateField('standardEpisodeFormat', v)}
        mediaType="episode"
        highlightTokens={missingInStandard}
      />
      <PatternEditor
        label="Daily Episode Format"
        value={form.dailyEpisodeFormat ?? ''}
        onChange={(v) => updateField('dailyEpisodeFormat', v)}
        mediaType="episode"
        highlightTokens={missingInDaily}
      />
      <PatternEditor
        label="Anime Episode Format"
        value={form.animeEpisodeFormat ?? ''}
        onChange={(v) => updateField('animeEpisodeFormat', v)}
        mediaType="episode"
        highlightTokens={missingInAnime}
      />
    </div>
  )
}

function MovieFormatsColumn({
  form,
  updateField,
  missingInMovie,
  stillMissingTokens,
}: {
  form: Partial<ImportSettings>
  updateField: (field: keyof ImportSettings, value: string) => void
  missingInMovie: string[]
  stillMissingTokens: Set<string>
}) {
  return (
    <div className="space-y-4">
      <h3 className="border-b pb-2 text-sm font-medium">Movie Format</h3>
      <PatternEditor
        label="Movie File Format"
        value={form.movieFileFormat ?? ''}
        onChange={(v) => updateField('movieFileFormat', v)}
        mediaType="movie"
        highlightTokens={missingInMovie}
      />
      <TokenReferenceList stillMissingTokens={stillMissingTokens} />
    </div>
  )
}

export function ResolveNamingModal(props: ResolveNamingModalProps) {
  const { open, onOpenChange } = props
  const state = useResolveNamingModal(props)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Resolve Naming Format Issues</DialogTitle>
          <DialogDescription>
            Add the missing tokens to your filename formats to differentiate files in different
            slots.
            <MissingTokensHint tokens={state.stillMissingTokens} />
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-6 py-4 md:grid-cols-2">
          <EpisodeFormatsColumn
            form={state.form}
            updateField={state.updateField}
            missingInStandard={state.missingInStandard}
            missingInDaily={state.missingInDaily}
            missingInAnime={state.missingInAnime}
          />
          <MovieFormatsColumn
            form={state.form}
            updateField={state.updateField}
            missingInMovie={state.missingInMovie}
            stillMissingTokens={state.stillMissingTokens}
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={state.handleSave} disabled={state.saving || !state.allResolved}>
            {state.saving ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
