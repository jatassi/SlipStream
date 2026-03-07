import { missingKeys } from '@/hooks/use-missing'
import { movieKeys } from '@/hooks/use-movies'
import { seriesKeys } from '@/hooks/use-series'

type InvalidationRule = {
  queryKeys: readonly (readonly string[])[]
}

// Maps module type to query keys to invalidate on any entity event from that module.
const entityInvalidationRegistry: Record<string, InvalidationRule> = {
  movie: {
    queryKeys: [movieKeys.all, missingKeys.counts()],
  },
  tv: {
    queryKeys: [seriesKeys.all, missingKeys.counts()],
  },
}

export function getEntityInvalidationKeys(moduleType: string): readonly (readonly string[])[] | undefined {
  const rule = entityInvalidationRegistry[moduleType] as InvalidationRule | undefined
  return rule?.queryKeys
}
