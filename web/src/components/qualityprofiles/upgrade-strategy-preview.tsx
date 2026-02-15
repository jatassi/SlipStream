import { useState } from 'react'

import { ArrowRight, Check, ChevronDown, TrendingUp, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import type { QualityItem, UpgradeStrategy } from '@/types'

import type { UpgradeScenario } from './upgrade-scenarios'
import { generateScenarios } from './upgrade-scenarios'

type UpgradeStrategyPreviewProps = {
  allowedQualities: QualityItem[]
  strategy: UpgradeStrategy
  cutoffId: number
  cutoffOverridesStrategy: boolean
}

export function UpgradeStrategyPreview({
  allowedQualities,
  strategy,
  cutoffId,
  cutoffOverridesStrategy,
}: UpgradeStrategyPreviewProps) {
  const [isOpen, setIsOpen] = useState(false)
  const scenarios = generateScenarios({
    allowedItems: allowedQualities,
    strategy,
    cutoffId,
    cutoffOverridesStrategy,
  })
  if (scenarios.length === 0) {
    return null
  }

  const allowedCount = scenarios.filter((s) => s.allowed).length
  const blockedCount = scenarios.length - allowedCount

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <div className="border-border/60 bg-muted/20 rounded-lg border px-3 py-2.5">
        <CollapsibleTrigger className="flex w-full items-center justify-between">
          <div className="text-muted-foreground flex items-center gap-1.5 text-xs font-medium">
            <TrendingUp className="size-3" />
            Upgrade Preview
          </div>
          <div className="flex items-center gap-2">
            {allowedCount > 0 && (
              <span className="text-[10px] text-green-500">{allowedCount} allowed</span>
            )}
            {blockedCount > 0 && (
              <span className="text-[10px] text-red-500">{blockedCount} blocked</span>
            )}
            <ChevronDown
              className={`text-muted-foreground size-3.5 transition-transform ${isOpen ? 'rotate-180' : ''}`}
            />
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-1.5 pt-1.5">
          {scenarios.map((s) => (
            <ScenarioRow key={`${s.from.id}-${s.to.id}-${s.reason}`} scenario={s} />
          ))}
        </CollapsibleContent>
      </div>
    </Collapsible>
  )
}

function ScenarioRow({ scenario }: { scenario: UpgradeScenario }) {
  const Icon = scenario.allowed ? Check : X
  const iconColor = scenario.allowed ? 'text-green-500' : 'text-red-500'

  return (
    <div className="flex items-center gap-2 text-sm">
      <Icon className={`size-3.5 shrink-0 ${iconColor}`} />
      <Badge
        variant="secondary"
        className={`px-1.5 py-0 text-xs font-normal ${scenario.allowed ? '' : 'opacity-60'}`}
      >
        {scenario.from.name}
        <ArrowRight className="mx-1 inline size-3" />
        {scenario.to.name}
      </Badge>
      <span className="text-muted-foreground text-[10px]">{scenario.reason}</span>
    </div>
  )
}
