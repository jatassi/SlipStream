export interface RootFolder {
  id: number
  path: string
  name: string
  mediaType: 'movie' | 'tv'
  freeSpace: number
  createdAt: string
}

export interface CreateRootFolderInput {
  path: string
  name?: string
  mediaType: 'movie' | 'tv'
}
