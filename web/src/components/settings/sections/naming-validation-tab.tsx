import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import type { ImportSettings } from '@/types'

import { ExtensionManager } from './extension-manager'
import { VALIDATION_LEVELS } from './file-naming-constants'
import { MediaInfoStatus } from './media-info-status'

function ValidationLevelSelect({
  value,
  onChange,
}: {
  value: string
  onChange: (v: string) => void
}) {
  return (
    <div className="space-y-3">
      <Label>Validation Level</Label>
      <Select value={value} onValueChange={(v) => v && onChange(v)}>
        <SelectTrigger>
          {VALIDATION_LEVELS.find((l) => l.value === value)?.label}
        </SelectTrigger>
        <SelectContent>
          {VALIDATION_LEVELS.map((level) => (
            <SelectItem key={level.value} value={level.value}>
              {level.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-muted-foreground text-xs">
        {VALIDATION_LEVELS.find((l) => l.value === value)?.description}
      </p>
    </div>
  )
}

function MinFileSizeSlider({
  value,
  onChange,
}: {
  value: number
  onChange: (v: number) => void
}) {
  return (
    <div className="space-y-3">
      <div className="flex justify-between">
        <Label>Minimum File Size</Label>
        <span className="text-muted-foreground text-sm">{value} MB</span>
      </div>
      <Slider
        value={[value]}
        onValueChange={(v) =>
          onChange(Array.isArray(v) && typeof v[0] === 'number' ? v[0] : value)
        }
        min={0}
        max={500}
        step={10}
      />
      <p className="text-muted-foreground text-xs">
        Files smaller than this will be rejected (helps filter sample files)
      </p>
    </div>
  )
}

export function ValidationTab({
  form,
  updateField,
}: {
  form: ImportSettings
  updateField: <K extends keyof ImportSettings>(field: K, value: ImportSettings[K]) => void
}) {
  return (
    <>
      <MediaInfoStatus />
      <Card>
        <CardHeader>
          <CardTitle>File Validation</CardTitle>
          <CardDescription>Configure how files are validated before import</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <ValidationLevelSelect
            value={form.validationLevel}
            onChange={(v) => updateField('validationLevel', v as ImportSettings['validationLevel'])}
          />
          <MinFileSizeSlider
            value={form.minimumFileSizeMB}
            onChange={(v) => updateField('minimumFileSizeMB', v)}
          />
          <ExtensionManager
            extensions={form.videoExtensions}
            onChange={(exts) => updateField('videoExtensions', exts)}
          />
        </CardContent>
      </Card>
    </>
  )
}
