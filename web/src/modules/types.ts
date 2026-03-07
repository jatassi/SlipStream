import type { ComponentType } from 'react'

import type { LucideIcon } from 'lucide-react'

export type ModuleConfig = {
  id: string
  name: string
  singularName: string
  pluralName: string
  icon: LucideIcon
  themeColor: string
  basePath: string
  routes: ModuleRouteConfig[]
  queryKeys: ModuleQueryKeys
  wsInvalidationRules: WSInvalidationRule[]
  filterOptions: ModuleFilterOption[]
  sortOptions: ModuleSortOption[]
  tableColumns: ModuleTableColumns
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  cardComponent: ComponentType<any>
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  detailComponent: ComponentType<any>
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  addConfigFields?: ComponentType<any>
  api: ModuleApi
}

export type ModuleRouteConfig = {
  path: string
  id: string
}

export type ModuleQueryKeys = {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  all: readonly any[]
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  list: (...args: any[]) => readonly any[]
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  detail: (id: number) => readonly any[]
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  [key: string]: any
}

export type WSInvalidationRule = {
  pattern: string
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  queryKeys: readonly (readonly any[])[]
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  alsoInvalidate?: readonly (readonly any[])[]
}

export type ModuleFilterOption = {
  value: string
  label: string
  icon: LucideIcon
}

export type ModuleSortOption = {
  value: string
  label: string
}

export type ModuleTableColumns = {
  static: unknown[]
  defaults: string[]
}

export type ModuleEntity = {
  id: number
  title: string
  sortTitle: string
  status: string
  monitored: boolean
  qualityProfileId: number
  rootFolderId: number
  path: string
  sizeOnDisk?: number
  addedAt: string
}

export type ModuleApi = {
  list: (options?: Record<string, unknown>) => Promise<unknown[]>
  get: (id: number) => Promise<unknown>
  update: (id: number, data: Record<string, unknown>) => Promise<unknown>
  delete: (id: number, deleteFiles?: boolean) => Promise<undefined>
  bulkDelete: (ids: number[], deleteFiles?: boolean) => Promise<void>
  bulkUpdate: (ids: number[], data: Record<string, unknown>) => Promise<unknown>
  bulkMonitor: (ids: number[], monitored: boolean) => Promise<unknown>
  search: (id: number) => Promise<undefined>
  refresh: (id: number) => Promise<unknown>
  refreshAll: () => Promise<unknown>
}
