import { Group } from '@/components/grouped-list'
import { SelectRow, SliderRow, StackedRow } from '@/components/settings/control-row'
import type { ImportSettings } from '@/types'

import { ExtensionManager } from './extension-manager'
import { VALIDATION_LEVELS } from './file-naming-constants'
import { MediaInfoStatus } from './media-info-status'

export function ValidationTab({
  form,
  updateField,
}: {
  form: ImportSettings
  updateField: <K extends keyof ImportSettings>(field: K, value: ImportSettings[K]) => void
}) {
  const level = VALIDATION_LEVELS.find((l) => l.value === form.validationLevel)

  return (
    <>
      <div className="px-screen mb-7">
        <MediaInfoStatus />
      </div>
      <Group header="File Validation" footer={level?.description}>
        <SelectRow
          label="Validation Level"
          value={form.validationLevel}
          onChange={(v) => updateField('validationLevel', v as ImportSettings['validationLevel'])}
          options={VALIDATION_LEVELS}
        />
      </Group>
      <Group footer="Files smaller than this are rejected, which helps filter out sample files.">
        <SliderRow
          label="Minimum File Size"
          value={form.minimumFileSizeMB}
          display={`${form.minimumFileSizeMB} MB`}
          onChange={(v) => updateField('minimumFileSizeMB', v)}
          min={0}
          max={500}
          step={10}
        />
      </Group>
      <Group
        header="Allowed Video Extensions"
        footer="Only files with one of these extensions are considered for import."
      >
        <StackedRow>
          <ExtensionManager
            extensions={form.videoExtensions}
            onChange={(exts) => updateField('videoExtensions', exts)}
          />
        </StackedRow>
      </Group>
    </>
  )
}
