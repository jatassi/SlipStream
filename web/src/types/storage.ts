// Storage-related types

export interface RootFolderRef {
  id: number
  name: string
  path: string
  mediaType: 'movie' | 'tv'
}

export interface StorageInfo {
  label: string
  path: string
  freeSpace: number
  totalSpace: number
  usedSpace: number
  usedPercent: number
  type: 'fixed' | 'removable' | 'network'
  rootFolders: RootFolderRef[] | null
}