export type DefaultEntry = {
  key: string
  entityType: string
  mediaType: string
  entityId: number
}

export type EntityType = 'root_folder' | 'quality_profile' | 'download_client' | 'indexer'
export type MediaType = 'movie' | 'tv'
