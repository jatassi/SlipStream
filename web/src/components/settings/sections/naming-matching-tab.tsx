import { Group } from '@/components/grouped-list'
import { SelectRow } from '@/components/settings/control-row'
import type { ImportSettings } from '@/types'

import { MATCH_CONFLICT_OPTIONS, UNKNOWN_MEDIA_OPTIONS } from './file-naming-constants'

function OptionGroup({
  header,
  label,
  value,
  onChange,
  options,
}: {
  header: string
  label: string
  value: string
  onChange: (v: string) => void
  options: readonly { value: string; label: string; description: string }[]
}) {
  return (
    <Group header={header} footer={options.find((o) => o.value === value)?.description}>
      <SelectRow label={label} value={value} onChange={onChange} options={options} />
    </Group>
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
    <>
      <OptionGroup
        header="Match Behavior"
        label="Match Conflict Behavior"
        value={form.matchConflictBehavior}
        onChange={(v) =>
          updateField('matchConflictBehavior', v as ImportSettings['matchConflictBehavior'])
        }
        options={MATCH_CONFLICT_OPTIONS}
      />
      <OptionGroup
        header="Unknown Media"
        label="Unknown Media Handling"
        value={form.unknownMediaBehavior}
        onChange={(v) =>
          updateField('unknownMediaBehavior', v as ImportSettings['unknownMediaBehavior'])
        }
        options={UNKNOWN_MEDIA_OPTIONS}
      />
    </>
  )
}
