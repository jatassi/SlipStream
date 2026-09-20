import { useState } from 'react'

import { Save } from 'lucide-react'
import { toast } from 'sonner'

import { Group } from '@/components/grouped-list'
import { InputRow, SliderRow, SwitchRow } from '@/components/settings/control-row'
import { SectionError, SectionLoading } from '@/components/settings/section-state'
import { Button } from '@/components/ui/button'
import { useAutoSearchSettings, useUpdateAutoSearchSettings } from '@/hooks'

const intervalLabel = (hours: number) => (hours === 1 ? 'Every hour' : `Every ${hours} hours`)

function useAutoSearchForm() {
  const { data: settings, isLoading, isError, refetch } = useAutoSearchSettings()
  const updateMutation = useUpdateAutoSearchSettings()

  const [enabled, setEnabled] = useState(true)
  const [intervalHours, setIntervalHours] = useState(1)
  const [backoffThreshold, setBackoffThreshold] = useState(12)
  const [prevSettings, setPrevSettings] = useState<typeof settings>(undefined)

  if (settings !== prevSettings) {
    setPrevSettings(settings)
    if (settings) {
      setEnabled(settings.enabled)
      setIntervalHours(settings.intervalHours)
      setBackoffThreshold(settings.backoffThreshold)
    }
  }

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({ enabled, intervalHours, backoffThreshold })
      toast.success('Settings saved')
    } catch {
      toast.error('Failed to save settings')
    }
  }

  const hasChanges = Boolean(
    settings &&
      (enabled !== settings.enabled ||
        intervalHours !== settings.intervalHours ||
        backoffThreshold !== settings.backoffThreshold),
  )

  return {
    enabled,
    setEnabled,
    intervalHours,
    setIntervalHours,
    backoffThreshold,
    setBackoffThreshold,
    handleSave,
    hasChanges,
    isSaving: updateMutation.isPending,
    isLoading,
    isError,
    refetch,
  }
}

function AutoSearchGroups({ form }: { form: ReturnType<typeof useAutoSearchForm> }) {
  return (
    <>
      <Group
        header="Automatic Search"
        footer="Scheduled task that periodically searches for missing monitored movies and episodes, every 1 to 24 hours."
      >
        <SwitchRow
          label="Enable Automatic Search"
          checked={form.enabled}
          onCheckedChange={form.setEnabled}
        />
        <SliderRow
          label="Search Interval"
          value={form.intervalHours}
          display={intervalLabel(form.intervalHours)}
          onChange={form.setIntervalHours}
          min={1}
          max={24}
          step={1}
          disabled={!form.enabled}
        />
      </Group>
      <Group
        header="Backoff"
        footer="After this many consecutive failed searches, the item is searched less frequently. Default: 12 failures before backoff."
      >
        <InputRow
          label="Backoff Threshold"
          type="number"
          min={1}
          value={form.backoffThreshold}
          disabled={!form.enabled}
          onChange={(e) =>
            // `||` intentional: parseInt of an emptied/invalid field is NaN, which `??` would not catch
            form.setBackoffThreshold(Math.max(1, Number.parseInt(e.target.value) || 1))
          }
        />
      </Group>
    </>
  )
}

export function AutoSearchSection() {
  const form = useAutoSearchForm()

  if (form.isLoading) {
    return <SectionLoading count={2} />
  }
  if (form.isError) {
    return <SectionError onRetry={form.refetch} />
  }

  return (
    <>
      <AutoSearchGroups form={form} />
      <div className="px-screen flex justify-end">
        <Button
          className="h-11"
          onClick={form.handleSave}
          disabled={form.isSaving || !form.hasChanges}
        >
          <Save className="mr-2 size-4" />
          Save Changes
        </Button>
      </div>
    </>
  )
}
