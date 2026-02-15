import { useState } from 'react'

import { AlertTriangle, ChevronDown } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { AttributeMode, AttributeSettings } from '@/types'

import { MODE_LABELS, MODE_OPTIONS } from './resolve-config-constants'

type CompactAttributeSectionProps = {
  label: string
  settings: AttributeSettings
  options: string[]
  isConflicting: boolean
  onItemModeChange: (value: string, mode: AttributeMode) => void
}

export function CompactAttributeSection({
  label,
  settings,
  options,
  isConflicting,
  onItemModeChange,
}: CompactAttributeSectionProps) {
  const [isOpen, setIsOpen] = useState(isConflicting)

  const getItemMode = (value: string): AttributeMode => settings.items[value] ?? 'acceptable'

  const requiredCount = countByMode(settings, 'required')
  const preferredCount = countByMode(settings, 'preferred')
  const notAllowedCount = countByMode(settings, 'notAllowed')
  const hasSettings = requiredCount > 0 || preferredCount > 0 || notAllowedCount > 0

  return (
    <Collapsible
      open={isOpen}
      onOpenChange={setIsOpen}
      className={`rounded-lg border ${isConflicting ? 'border-orange-400 dark:border-orange-500' : ''}`}
    >
      <CollapsibleTrigger className="hover:bg-muted/50 flex w-full items-center justify-between p-2 transition-colors">
        <SectionLabel label={label} isConflicting={isConflicting} />
        <SectionBadges
          requiredCount={requiredCount}
          preferredCount={preferredCount}
          notAllowedCount={notAllowedCount}
          hasSettings={hasSettings}
          isOpen={isOpen}
        />
      </CollapsibleTrigger>

      <CollapsibleContent>
        <AttributeOptionList
          options={options}
          getItemMode={getItemMode}
          onItemModeChange={onItemModeChange}
        />
      </CollapsibleContent>
    </Collapsible>
  )
}

function AttributeOptionList({
  options,
  getItemMode,
  onItemModeChange,
}: {
  options: string[]
  getItemMode: (value: string) => AttributeMode
  onItemModeChange: (value: string, mode: AttributeMode) => void
}) {
  return (
    <div className="space-y-1 border-t px-2 pt-2 pb-2">
      {options.map((value) => (
        <div key={value} className="flex items-center justify-between py-0.5">
          <span className="text-xs">{value}</span>
          <Select
            value={getItemMode(value)}
            onValueChange={(v) => v && onItemModeChange(value, v as AttributeMode)}
          >
            <SelectTrigger className="h-6 w-24 text-[10px]">
              {MODE_LABELS[getItemMode(value)]}
            </SelectTrigger>
            <SelectContent>
              {MODE_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ))}
    </div>
  )
}

function countByMode(settings: AttributeSettings, mode: AttributeMode): number {
  return Object.values(settings.items).filter((m) => m === mode).length
}

function SectionLabel({ label, isConflicting }: { label: string; isConflicting: boolean }) {
  const textClass = isConflicting ? 'text-orange-600 dark:text-orange-400' : ''
  return (
    <div className="flex items-center gap-1.5">
      {isConflicting ? <AlertTriangle className="size-3.5 text-orange-500" /> : null}
      <span className={`text-xs font-medium ${textClass}`}>{label}</span>
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
    <div className="flex items-center gap-1.5">
      {requiredCount > 0 && (
        <Badge variant="destructive" className="h-4 px-1 py-0 text-[10px]">
          {requiredCount}
        </Badge>
      )}
      {preferredCount > 0 && (
        <Badge variant="secondary" className="h-4 px-1 py-0 text-[10px]">
          {preferredCount}
        </Badge>
      )}
      {notAllowedCount > 0 && (
        <Badge variant="outline" className="h-4 px-1 py-0 text-[10px]">
          {notAllowedCount}
        </Badge>
      )}
      {!hasSettings && <span className="text-muted-foreground text-[10px]">Acceptable</span>}
      <ChevronDown
        className={`text-muted-foreground size-3 transition-transform ${isOpen ? 'rotate-180' : ''}`}
      />
    </div>
  )
}
