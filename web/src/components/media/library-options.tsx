import { useRef, useState } from 'react'

import { Ellipsis } from 'lucide-react'

import type { ActionItem } from '@/components/presenter'
import { ActionPresenter } from '@/components/presenter'
import { useViewport } from '@/hooks/use-viewport'
import type { ColumnDef } from '@/lib/table-columns'

type Menu = 'root' | 'sort' | 'view' | 'size' | 'columns'

type Nav = {
  open: (menu: Menu) => void
  close: () => void
}

const POSTER_SIZE_PRESETS = [
  { label: 'Small', value: 120 },
  { label: 'Medium', value: 160 },
  { label: 'Large', value: 240 },
]

const MENU_TITLES: Record<Menu, string> = {
  root: 'Options',
  sort: 'Sort by',
  view: 'View',
  size: 'Poster size',
  columns: 'Columns',
}

export type LibraryOptionsProps<T> = {
  pluralMediaLabel: string
  sortOptions: { value: string; label: string }[]
  sortField: string
  sortDirection: 'asc' | 'desc'
  onSortFieldChange: (value: string) => void
  onToggleSortDirection: () => void
  view: 'grid' | 'table'
  onViewChange: (view: 'grid' | 'table') => void
  posterSize: number
  onPosterSizeChange: (size: number) => void
  columns: ColumnDef<T>[]
  visibleColumnIds: string[]
  onTableColumnsChange: (ids: string[]) => void
  onEnterEdit: () => void
  isRefreshing: boolean
  onRefreshAll: () => void
}

function mark(label: string, active: boolean): string {
  return active ? `${label} ✓` : label
}

function sortActions<T>(props: LibraryOptionsProps<T>, nav: Nav): ActionItem[] {
  return props.sortOptions.map((option) => ({
    label: mark(option.label, option.value === props.sortField),
    onClick: () => {
      props.onSortFieldChange(option.value)
      nav.close()
    },
  }))
}

function viewActions<T>(props: LibraryOptionsProps<T>, nav: Nav): ActionItem[] {
  return (['grid', 'table'] as const).map((view) => ({
    label: mark(view === 'grid' ? 'Grid view' : 'Table view', view === props.view),
    onClick: () => {
      props.onViewChange(view)
      nav.close()
    },
  }))
}

function sizeActions<T>(props: LibraryOptionsProps<T>, nav: Nav): ActionItem[] {
  return POSTER_SIZE_PRESETS.map((preset) => ({
    label: mark(preset.label, preset.value === props.posterSize),
    onClick: () => {
      props.onPosterSizeChange(preset.value)
      nav.close()
    },
  }))
}

function columnActions<T>(props: LibraryOptionsProps<T>): ActionItem[] {
  return props.columns
    .filter((column) => column.hideable)
    .map((column) => {
      const visible = props.visibleColumnIds.includes(column.id)
      return {
        label: mark(column.label, visible),
        onClick: () => {
          props.onTableColumnsChange(
            visible
              ? props.visibleColumnIds.filter((id) => id !== column.id)
              : [...props.visibleColumnIds, column.id],
          )
        },
      }
    })
}

function wideActions<T>(props: LibraryOptionsProps<T>, nav: Nav): ActionItem[] {
  return [
    { label: 'View', onClick: () => nav.open('view') },
    props.view === 'grid'
      ? { label: 'Poster size', onClick: () => nav.open('size') }
      : { label: 'Columns', onClick: () => nav.open('columns') },
  ]
}

function rootActions<T>(props: LibraryOptionsProps<T>, wide: boolean, nav: Nav): ActionItem[] {
  const sortLabel =
    props.sortOptions.find((option) => option.value === props.sortField)?.label ?? 'Title'
  return [
    { label: `Sort by ${sortLabel}`, onClick: () => nav.open('sort') },
    {
      label: mark('Reverse order', props.sortDirection === 'desc'),
      onClick: () => {
        props.onToggleSortDirection()
        nav.close()
      },
    },
    ...(wide ? wideActions(props, nav) : []),
    {
      label: `Select ${props.pluralMediaLabel.toLowerCase()}`,
      onClick: () => {
        props.onEnterEdit()
        nav.close()
      },
    },
    {
      label: props.isRefreshing ? 'Refreshing…' : 'Refresh all',
      onClick: () => {
        props.onRefreshAll()
        nav.close()
      },
    },
  ]
}

export function LibraryOptions<T>(props: LibraryOptionsProps<T>) {
  const anchor = useRef<HTMLButtonElement>(null)
  const [menu, setMenu] = useState<Menu | null>(null)
  const wide = useViewport() === 'wide'
  const nav: Nav = {
    open: (next) => {
      setMenu(next)
    },
    close: () => {
      setMenu(null)
    },
  }

  const actionsFor: Record<Menu, ActionItem[]> = {
    root: rootActions(props, wide, nav),
    sort: sortActions(props, nav),
    view: viewActions(props, nav),
    size: sizeActions(props, nav),
    columns: columnActions(props),
  }

  return (
    <>
      <button
        ref={anchor}
        type="button"
        aria-label="Options"
        onClick={() => {
          nav.open('root')
        }}
        className="press-dim focus-visible:ring-ring text-primary flex size-11 items-center justify-center rounded-full outline-none focus-visible:ring-[3px]"
      >
        <Ellipsis className="size-6" />
      </button>
      {(Object.keys(actionsFor) as Menu[]).map((id) => (
        <ActionPresenter
          key={id}
          open={menu === id}
          onOpenChange={(next) => {
            if (!next) {
              setMenu((current) => (current === id ? null : current))
            }
          }}
          title={MENU_TITLES[id]}
          actions={actionsFor[id]}
          anchor={anchor}
        />
      ))}
    </>
  )
}
