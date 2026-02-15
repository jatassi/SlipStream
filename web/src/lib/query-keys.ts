export function createQueryKeys<const T extends readonly string[]>(...scope: T) {
  return {
    all: scope,
    lists: () => [...scope, 'list'] as const,
    list: () => [...scope, 'list'] as const,
    details: () => [...scope, 'detail'] as const,
    detail: (id: number) => [...scope, 'detail', id] as const,
  }
}
