export type RootFolder = {
  id: number
  path: string
  name: string
  mediaType: 'movie' | 'tv'
  freeSpace: number
  createdAt: string
  isDefault?: boolean
}

export type CreateRootFolderInput = {
  path: string
  name?: string
  mediaType: 'movie' | 'tv'
}
