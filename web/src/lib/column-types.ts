import type { ReactNode } from 'react'

export type ColumnDef<T> = {
  id: string
  label: string
  sortField?: string
  defaultVisible: boolean
  hideable: boolean
  minWidth?: string
  render: (item: T, ctx: ColumnRenderContext) => ReactNode
  headerClassName?: string
  cellClassName?: string
}

export type ColumnRenderContext = {
  qualityProfileNames: Map<number, string>
  rootFolderNames: Map<number, string>
}
