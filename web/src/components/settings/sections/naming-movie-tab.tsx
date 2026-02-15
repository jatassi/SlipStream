import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import type { ImportSettings } from '@/types'

import { FilenameTester } from './filename-tester'
import { PatternEditor } from './naming-pattern-editor'

export function MovieNamingTab({
  form,
  updateField,
}: {
  form: ImportSettings
  updateField: <K extends keyof ImportSettings>(field: K, value: ImportSettings[K]) => void
}) {
  return (
    <>
      <FilenameTester mediaType="movie" />

      <Card>
        <CardHeader>
          <CardTitle>Movie Renaming</CardTitle>
          <CardDescription>Configure how movies are renamed during import</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Rename Movies</Label>
              <p className="text-muted-foreground text-sm">
                Rename files according to format patterns
              </p>
            </div>
            <Switch
              checked={form.renameMovies}
              onCheckedChange={(v) => updateField('renameMovies', v)}
            />
          </div>

          <PatternEditor
            label="Movie Folder Format"
            value={form.movieFolderFormat}
            onChange={(v) => updateField('movieFolderFormat', v)}
            description="Folder name for each movie"
            mediaType="folder"
            tokenContext="movie-folder"
          />

          <PatternEditor
            label="Movie File Format"
            value={form.movieFileFormat}
            onChange={(v) => updateField('movieFileFormat', v)}
            description="Filename pattern for movie files"
            mediaType="movie"
            tokenContext="movie"
          />
        </CardContent>
      </Card>
    </>
  )
}
