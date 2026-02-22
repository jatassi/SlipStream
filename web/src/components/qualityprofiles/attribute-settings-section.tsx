import { useState } from 'react'

import { AlertTriangle, ChevronDown } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { AttributeMode, AttributeSettings } from '@/types'

import { MODE_LABELS, MODE_OPTIONS } from './constants'

const EMPTY_ITEMS: string[] = []

type AttributeSettingsSectionProps = {
  label: string
  settings: AttributeSettings
  options: string[]
  disabledItems?: string[]
  warning?: string | null
  onItemModeChange: (value: string, mode: AttributeMode) => void
}

function countByMode(settings: AttributeSettings, mode: AttributeMode): number {
  return Object.values(settings.items).filter((m) => m === mode).length
}

export function AttributeSettingsSection({
  label,
  settings,
  options,
  disabledItems = EMPTY_ITEMS,
  warning,
  onItemModeChange,
}: AttributeSettingsSectionProps) {
  const [isOpen, setIsOpen] = useState(false)
  const requiredCount = countByMode(settings, 'required')
  const preferredCount = countByMode(settings, 'preferred')
  const notAllowedCount = countByMode(settings, 'notAllowed')
  const hasSettings = requiredCount > 0 || preferredCount > 0 || notAllowedCount > 0

  return (
    <Collapsible
      open={isOpen}
      onOpenChange={setIsOpen}
      className={`rounded-lg border ${warning ? 'border-yellow-500' : ''}`}
    >
      <CollapsibleTrigger className="hover:bg-muted/50 flex w-full items-center justify-between p-3 transition-colors">
        <SectionLabel label={label} warning={warning} />
        <SectionBadges
          requiredCount={requiredCount}
          preferredCount={preferredCount}
          notAllowedCount={notAllowedCount}
          hasSettings={hasSettings}
          isOpen={isOpen}
        />
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="space-y-1.5 border-t px-3 pt-3 pb-3">
          {options.map((value) => (
            <AttributeItemRow
              key={value}
              value={value}
              mode={settings.items[value] ?? 'acceptable'}
              disabled={disabledItems.includes(value)}
              onModeChange={(mode) => onItemModeChange(value, mode)}
            />
          ))}
          {warning ? <WarningBanner message={warning} /> : null}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

function SectionLabel({ label, warning }: { label: string; warning?: string | null }) {
  return (
    <div className="flex items-center gap-2">
      {warning ? <AlertTriangle className="size-4 text-yellow-500" /> : null}
      <span className="text-sm font-medium">{label}</span>
    </div>
  )
}

function SectionBadges({
  requiredCount,
  preferredCount,
  notAllowedCount,
  hasSettings,
  isOpen,
}: {
  requiredCount: number
  preferredCount: number
  notAllowedCount: number
  hasSettings: boolean
  isOpen: boolean
}) {
  return (
    <div className="flex items-center gap-2">
      {requiredCount > 0 && (
        <Badge variant="destructive" className="px-1.5 py-0 text-xs">
          {requiredCount} required
        </Badge>
      )}
      {preferredCount > 0 && (
        <Badge variant="secondary" className="px-1.5 py-0 text-xs">
          {preferredCount} preferred
        </Badge>
      )}
      {notAllowedCount > 0 && (
        <Badge variant="outline" className="px-1.5 py-0 text-xs">
          {notAllowedCount} blocked
        </Badge>
      )}
      {!hasSettings && <span className="text-muted-foreground text-xs">Acceptable</span>}
      <ChevronDown
        className={`text-muted-foreground size-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
      />
    </div>
  )
}

function AttributeItemRow({
  value,
  mode,
  disabled,
  onModeChange,
}: {
  value: string
  mode: AttributeMode
  disabled?: boolean
  onModeChange: (mode: AttributeMode) => void
}) {
  return (
    <div className={`flex items-center justify-between py-1 ${disabled ? 'opacity-50' : ''}`}>
      <span className="text-sm">{value}</span>
      <Select
        value={mode}
        onValueChange={(v) => v && onModeChange(v as AttributeMode)}
        disabled={disabled}
      >
        <SelectTrigger className="h-7 w-28 text-xs">{MODE_LABELS[mode]}</SelectTrigger>
        <SelectContent>
          {MODE_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function WarningBanner({ message }: { message: string }) {
  return (
    <div className="mt-2 flex items-center gap-2 rounded border border-yellow-500/20 bg-yellow-500/10 p-2 text-yellow-600 dark:text-yellow-500">
      <AlertTriangle className="size-4 shrink-0" />
      <span className="text-xs">{message}</span>
    </div>
  )
}
