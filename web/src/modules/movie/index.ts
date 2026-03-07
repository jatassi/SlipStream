import {
  ArrowDownCircle,
  ArrowUpCircle,
  Binoculars,
  CheckCircle,
  Clock,
  Eye,
  Film,
  XCircle,
} from 'lucide-react'

import { moviesApi } from '@/api/movies'
import { MovieCard } from '@/components/movies/movie-card'
import { missingKeys } from '@/hooks/use-missing'
import { movieKeys } from '@/hooks/use-movies'

import type { ModuleConfig } from '../types'

export const movieModuleConfig: ModuleConfig = {
  id: 'movie',
  name: 'Movies',
  singularName: 'Movie',
  pluralName: 'Movies',
  icon: Film,
  themeColor: 'movie',
  basePath: '/movies',

  routes: [
    { path: '/', id: 'movies' },
    { path: '/$id', id: 'movieDetail' },
    { path: '/add', id: 'addMovie' },
  ],

  queryKeys: movieKeys,

  wsInvalidationRules: [
    {
      pattern: 'movie:(added|updated|deleted)',
      queryKeys: [movieKeys.all],
      alsoInvalidate: [missingKeys.counts()],
    },
  ],

  filterOptions: [
    { value: 'monitored', label: 'Monitored', icon: Eye },
    { value: 'unreleased', label: 'Unreleased', icon: Clock },
    { value: 'missing', label: 'Missing', icon: Binoculars },
    { value: 'downloading', label: 'Downloading', icon: ArrowDownCircle },
    { value: 'failed', label: 'Failed', icon: XCircle },
    { value: 'upgradable', label: 'Upgradable', icon: ArrowUpCircle },
    { value: 'available', label: 'Available', icon: CheckCircle },
  ],

  sortOptions: [
    { value: 'title', label: 'Title' },
    { value: 'monitored', label: 'Monitored' },
    { value: 'qualityProfile', label: 'Quality Profile' },
    { value: 'releaseDate', label: 'Release Date' },
    { value: 'dateAdded', label: 'Date Added' },
    { value: 'rootFolder', label: 'Root Folder' },
    { value: 'sizeOnDisk', label: 'Size on Disk' },
  ],

  tableColumns: { static: [], defaults: [] },

  cardComponent: MovieCard,
  detailComponent: () => null,

  api: {
    list: (options) => moviesApi.list(options),
    get: (id) => moviesApi.get(id),
    update: (id, data) => moviesApi.update(id, data),
    delete: (id, deleteFiles) => moviesApi.delete(id, deleteFiles),
    bulkDelete: (ids, deleteFiles) =>
      moviesApi.bulkDelete(ids, deleteFiles).then(() => undefined),
    bulkUpdate: (ids, data) => moviesApi.bulkUpdate(ids, data),
    bulkMonitor: (ids, monitored) => moviesApi.bulkMonitor(ids, monitored),
    search: (id) => moviesApi.search(id),
    refresh: (id) => moviesApi.refresh(id),
    refreshAll: () => moviesApi.refreshAll(),
  },
}
