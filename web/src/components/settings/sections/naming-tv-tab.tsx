import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { ImportSettings } from '@/types'

import {
  COLON_REPLACEMENT_OPTIONS,
  MULTI_EPISODE_STYLES,
} from './file-naming-constants'
import { FilenameTester } from './filename-tester'
import { PatternEditor } from './naming-pattern-editor'

type TabProps = {
  form: ImportSettings
  updateField: <K extends keyof ImportSettings>(field: K, value: ImportSettings[K]) => void
}

function ColonReplacementSelect({ form, updateField }: TabProps) {
  return (
    <div className="space-y-3">
      <Label>Colon Replacement</Label>
      <Select
        value={form.colonReplacement}
        onValueChange={(v) => v && updateField('colonReplacement', v)}
      >
        <SelectTrigger>
          {COLON_REPLACEMENT_OPTIONS.find((o) => o.value === form.colonReplacement)?.label}
        </SelectTrigger>
        <SelectContent>
          {COLON_REPLACEMENT_OPTIONS.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-muted-foreground text-xs">
        Example:{' '}
        {COLON_REPLACEMENT_OPTIONS.find((o) => o.value === form.colonReplacement)?.example}
      </p>
      {form.colonReplacement === 'custom' && (
        <Input
          value={form.customColonReplacement ?? ''}
          onChange={(e) => updateField('customColonReplacement', e.target.value)}
          placeholder="Enter custom replacement character"
        />
      )}
    </div>
  )
}

function MultiEpisodeStyleSelect({ form, updateField }: TabProps) {
  return (
    <div className="space-y-3">
      <Label>Multi-Episode Style</Label>
      <Select
        value={form.multiEpisodeStyle}
        onValueChange={(v) => v && updateField('multiEpisodeStyle', v)}
      >
        <SelectTrigger>
          {MULTI_EPISODE_STYLES.find((s) => s.value === form.multiEpisodeStyle)?.label}
        </SelectTrigger>
        <SelectContent>
          {MULTI_EPISODE_STYLES.map((style) => (
            <SelectItem key={style.value} value={style.value}>
              {style.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-muted-foreground font-mono text-xs">
        Example:{' '}
        {MULTI_EPISODE_STYLES.find((s) => s.value === form.multiEpisodeStyle)?.example}
      </p>
    </div>
  )
}

function EpisodeRenamingCard({ form, updateField }: TabProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Episode Renaming</CardTitle>
        <CardDescription>Configure how TV episodes are renamed during import</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label>Rename Episodes</Label>
            <p className="text-muted-foreground text-sm">
              Rename files according to format patterns
            </p>
          </div>
          <Switch
            checked={form.renameEpisodes}
            onCheckedChange={(v) => updateField('renameEpisodes', v)}
          />
        </div>

        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label>Replace Illegal Characters</Label>
            <p className="text-muted-foreground text-sm">
              Replace filesystem-illegal characters with safe alternatives
            </p>
          </div>
          <Switch
            checked={form.replaceIllegalCharacters}
            onCheckedChange={(v) => updateField('replaceIllegalCharacters', v)}
          />
        </div>

        <ColonReplacementSelect form={form} updateField={updateField} />
        <MultiEpisodeStyleSelect form={form} updateField={updateField} />
      </CardContent>
    </Card>
  )
}

function EpisodeFormatCard({ form, updateField }: TabProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Episode Format Patterns</CardTitle>
        <CardDescription>Define naming patterns for different episode types</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <PatternEditor
          label="Standard Episode Format"
          value={form.standardEpisodeFormat}
          onChange={(v) => updateField('standardEpisodeFormat', v)}
          description="For regular TV series"
          mediaType="episode"
          tokenContext="episode"
        />
        <PatternEditor
          label="Daily Episode Format"
          value={form.dailyEpisodeFormat}
          onChange={(v) => updateField('dailyEpisodeFormat', v)}
          description="For daily/date-based shows"
          mediaType="episode"
          tokenContext="episode"
        />
        <PatternEditor
          label="Anime Episode Format"
          value={form.animeEpisodeFormat}
          onChange={(v) => updateField('animeEpisodeFormat', v)}
          description="For anime series"
          mediaType="episode"
          tokenContext="episode"
        />
      </CardContent>
    </Card>
  )
}

function FolderFormatCard({ form, updateField }: TabProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Folder Format Patterns</CardTitle>
        <CardDescription>
          Define folder naming patterns for series organization
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <PatternEditor
          label="Series Folder Format"
          value={form.seriesFolderFormat}
          onChange={(v) => updateField('seriesFolderFormat', v)}
          description="Root folder for each series"
          mediaType="folder"
          tokenContext="series-folder"
        />
        <PatternEditor
          label="Season Folder Format"
          value={form.seasonFolderFormat}
          onChange={(v) => updateField('seasonFolderFormat', v)}
          description="Subfolder for each season"
          mediaType="folder"
          tokenContext="season-folder"
        />
        <PatternEditor
          label="Specials Folder Format"
          value={form.specialsFolderFormat}
          onChange={(v) => updateField('specialsFolderFormat', v)}
          description="Folder for specials (Season 0)"
          mediaType="folder"
          tokenContext="series-folder"
        />
      </CardContent>
    </Card>
  )
}

export function TvNamingTab({ form, updateField }: TabProps) {
  return (
    <>
      <FilenameTester mediaType="tv" />
      <EpisodeRenamingCard form={form} updateField={updateField} />
      <EpisodeFormatCard form={form} updateField={updateField} />
      <FolderFormatCard form={form} updateField={updateField} />
    </>
  )
}
