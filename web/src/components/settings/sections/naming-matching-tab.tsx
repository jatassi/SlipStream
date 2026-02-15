import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { ImportSettings } from '@/types'

import { MATCH_CONFLICT_OPTIONS, UNKNOWN_MEDIA_OPTIONS } from './file-naming-constants'

function OptionSelect({
  label,
  value,
  onChange,
  options,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  options: readonly { value: string; label: string; description: string }[]
}) {
  return (
    <div className="space-y-3">
      <Label>{label}</Label>
      <Select value={value} onValueChange={(v) => v && onChange(v)}>
        <SelectTrigger>
          {options.find((o) => o.value === value)?.label}
        </SelectTrigger>
        <SelectContent>
          {options.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-muted-foreground text-xs">
        {options.find((o) => o.value === value)?.description}
      </p>
    </div>
  )
}

export function MatchingTab({
  form,
  updateField,
}: {
  form: ImportSettings
  updateField: <K extends keyof ImportSettings>(field: K, value: ImportSettings[K]) => void
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Match Behavior</CardTitle>
        <CardDescription>Configure how files are matched to library items</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <OptionSelect
          label="Match Conflict Behavior"
          value={form.matchConflictBehavior}
          onChange={(v) => updateField('matchConflictBehavior', v as ImportSettings['matchConflictBehavior'])}
          options={MATCH_CONFLICT_OPTIONS}
        />
        <OptionSelect
          label="Unknown Media Handling"
          value={form.unknownMediaBehavior}
          onChange={(v) => updateField('unknownMediaBehavior', v as ImportSettings['unknownMediaBehavior'])}
          options={UNKNOWN_MEDIA_OPTIONS}
        />
      </CardContent>
    </Card>
  )
}
