import type { AttributeMode, AttributeSettings, Quality, QualityItem, UpgradeStrategy } from '@/types'

export type UpgradeScenario = {
  from: Quality
  to: Quality
  allowed: boolean
  reason: string
}

type ScenarioCollector = {
  scenarios: UpgradeScenario[]
  addedKeys: Set<string>
}

type ScenarioContext = {
  sorted: Quality[]
  cutoffWeight: number
  strategy: UpgradeStrategy
  collector: ScenarioCollector
}

const DISC_SOURCES = new Set(['bluray', 'remux'])

function isUpgradeByStrategy(
  current: Quality,
  candidate: Quality,
  strategy: UpgradeStrategy,
): boolean {
  switch (strategy) {
    case 'resolution_only': {
      return candidate.resolution > current.resolution
    }
    case 'balanced': {
      if (candidate.resolution > current.resolution) {
        return true
      }
      if (candidate.resolution === current.resolution) {
        return DISC_SOURCES.has(candidate.source) && !DISC_SOURCES.has(current.source)
      }
      return false
    }
    default: {
      return candidate.weight > current.weight
    }
  }
}

function addScenario(collector: ScenarioCollector, scenario: UpgradeScenario): void {
  const key = `${scenario.from.id}-${scenario.to.id}`
  if (collector.addedKeys.has(key)) {
    return
  }
  collector.addedKeys.add(key)
  collector.scenarios.push(scenario)
}

function addSourceUpgrade(from: Quality, to: Quality, ctx: ScenarioContext): void {
  const passes = isUpgradeByStrategy(from, to, ctx.strategy)
  addScenario(ctx.collector, {
    from,
    to,
    allowed: passes,
    reason: passes ? 'Better source' : 'Same tier',
  })
}

function addDiscCrossover(from: Quality, to: Quality, ctx: ScenarioContext): void {
  const passes = isUpgradeByStrategy(from, to, ctx.strategy)
  addScenario(ctx.collector, {
    from,
    to,
    allowed: passes,
    reason: passes ? 'Non-disc to disc' : 'Same resolution',
  })
}

function addResolutionScenarios(ctx: ScenarioContext): void {
  const { sorted, cutoffWeight, collector } = ctx
  const resolutions = [...new Set(sorted.map((q) => q.resolution))].toSorted((a, b) => a - b)

  for (const res of resolutions) {
    addScenariosForResolution(res, ctx)
  }

  for (let i = 0; i < resolutions.length - 1; i++) {
    const fromQ = sorted.filter((q) => q.resolution === resolutions[i])
    const toQ = sorted.find((q) => q.resolution === resolutions[i + 1])
    const from = fromQ.at(-1)
    if (from && toQ && from.weight < cutoffWeight) {
      addScenario(collector, { from, to: toQ, allowed: true, reason: 'Higher resolution' })
    }
  }
}

function addScenariosForResolution(res: number, ctx: ScenarioContext): void {
  const { sorted, cutoffWeight } = ctx
  const atRes = sorted.filter((q) => q.resolution === res)
  const belowCutoffAtRes = atRes.filter((q) => q.weight < cutoffWeight)
  if (belowCutoffAtRes.length === 0) {
    return
  }

  const nonDisc = belowCutoffAtRes.filter((q) => !DISC_SOURCES.has(q.source))
  const disc = atRes.filter((q) => DISC_SOURCES.has(q.source))

  const lastNonDisc = nonDisc.at(-1)
  if (nonDisc.length >= 2 && lastNonDisc) {
    addSourceUpgrade(nonDisc[0], lastNonDisc, ctx)
  }

  if (lastNonDisc && disc.length > 0) {
    addDiscCrossover(lastNonDisc, disc[0], ctx)
  }

  const lastDisc = disc.at(-1)
  if (disc.length >= 2 && disc[0].weight < cutoffWeight && lastDisc) {
    addSourceUpgrade(disc[0], lastDisc, ctx)
  }
}

function addCutoffScenarios(ctx: ScenarioContext, cutoffQ: Quality | undefined): void {
  const { sorted, cutoffWeight, collector } = ctx

  const atCutoff = sorted.filter((q) => q.weight >= cutoffWeight)
  if (atCutoff.length === 0) {
    return
  }
  const from = atCutoff[0]
  const higher = sorted.find((q) => q.weight > from.weight)
  if (higher) {
    addScenario(collector, { from, to: higher, allowed: false, reason: 'At cutoff' })
  }

  if (!cutoffQ) {
    return
  }
  const belowCutoff = sorted.filter((q) => q.weight < cutoffWeight)
  const overrideFrom = belowCutoff
    .toReversed()
    .find((q) => !isUpgradeByStrategy(q, cutoffQ, ctx.strategy))
  if (overrideFrom) {
    addScenario(collector, { from: overrideFrom, to: cutoffQ, allowed: true, reason: 'Cutoff override' })
  }
}

type GenerateParams = {
  allowedItems: QualityItem[]
  strategy: UpgradeStrategy
  cutoffId: number
  cutoffOverridesStrategy: boolean
}

export function generateScenarios(params: GenerateParams): UpgradeScenario[] {
  const { allowedItems, strategy, cutoffId, cutoffOverridesStrategy } = params
  const allowed = allowedItems.filter((i) => i.allowed).map((i) => i.quality)
  if (allowed.length < 2) {
    return []
  }

  const sorted = allowed.toSorted((a, b) => a.weight - b.weight)
  const cutoffQ = sorted.find((q) => q.id === cutoffId)
  const lastWeight = sorted.at(-1)?.weight ?? 0
  const cutoffWeight = cutoffQ?.weight ?? lastWeight

  const collector: ScenarioCollector = { scenarios: [], addedKeys: new Set<string>() }
  const ctx: ScenarioContext = { sorted, cutoffWeight, strategy, collector }

  addResolutionScenarios(ctx)

  if (cutoffOverridesStrategy) {
    addCutoffScenarios(ctx, cutoffQ)
  } else {
    addCutoffScenarios(ctx, undefined)
  }

  return collector.scenarios.toSorted((a, b) => {
    if (a.allowed === b.allowed) {
      return 0
    }
    return a.allowed ? -1 : 1
  })
}

export function validateAttributeGroup(
  settings: AttributeSettings,
  options: string[],
): string | null {
  if (options.length === 0) {
    return null
  }

  const modes = options.map((opt) => settings.items[opt] ?? 'acceptable')
  const nonAcceptableModes = modes.filter((m) => m !== 'acceptable')

  if (nonAcceptableModes.length !== options.length) {
    return null
  }

  const firstMode = nonAcceptableModes[0]
  const allSame = nonAcceptableModes.every((m) => m === firstMode)

  if (allSame) {
    const messages: Partial<Record<AttributeMode, string>> = {
      required: 'All items set to Required - no release can match all requirements',
      preferred: 'All items set to Preferred - this is equivalent to Acceptable',
      notAllowed: 'All items set to Not Allowed - no release can match',
    }
    return messages[firstMode] ?? null
  }
  return null
}
