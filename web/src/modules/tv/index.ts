import { lazy } from 'react'

import {
  ArrowDownCircle,
  ArrowUpCircle,
  Binoculars,
  CheckCircle,
  CircleStop,
  Clock,
  Eye,
  Play,
  Tv,
  XCircle,
} from 'lucide-react'

import { seriesApi } from '@/api/series'
import { SeriesCard } from '@/components/series/series-card'
import { missingKeys } from '@/hooks/use-missing'
import { seriesKeys } from '@/hooks/use-series'

import type { ModuleConfig } from '../types'

const SeriesListPage = lazy(() => import('@/routes/series').then((m) => ({ default: m.SeriesListPage })))

export const tvModuleConfig: ModuleConfig = {
  id: 'tv',
  name: 'Series',
  singularName: 'Series',
  pluralName: 'Series',
  icon: Tv,
  themeColor: 'tv',
  basePath: '/series',

  routes: [
    { path: '/', id: 'series' },
    { path: '/$id', id: 'seriesDetail' },
    { path: '/add', id: 'addSeries' },
  ],

  queryKeys: seriesKeys,

  wsInvalidationRules: [
    {
      pattern: 'series:(added|updated|deleted)',
      queryKeys: [seriesKeys.all],
      alsoInvalidate: [missingKeys.counts()],
    },
  ],

  filterOptions: [
    { value: 'monitored', label: 'Monitored', icon: Eye },
    { value: 'continuing', label: 'Continuing', icon: Play },
    { value: 'ended', label: 'Ended', icon: CircleStop },
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
    { value: 'nextAirDate', label: 'Next Air Date' },
    { value: 'dateAdded', label: 'Date Added' },
    { value: 'rootFolder', label: 'Root Folder' },
    { value: 'sizeOnDisk', label: 'Size on Disk' },
  ],

  tableColumns: { static: [], defaults: [] },

  listComponent: SeriesListPage,
  cardComponent: SeriesCard,
  detailComponent: () => null,

  missingTabValue: 'series',
  missingCountKey: 'episodeCount',

  api: {
    list: (options) => seriesApi.list(options),
    get: (id) => seriesApi.get(id),
    update: (id, data) => seriesApi.update(id, data),
    delete: (id, deleteFiles) => seriesApi.delete(id, deleteFiles),
    bulkDelete: (ids, deleteFiles) =>
      seriesApi.bulkDelete(ids, deleteFiles).then(() => undefined),
    bulkUpdate: (ids, data) => seriesApi.bulkUpdate(ids, data),
    bulkMonitor: (ids, monitored) =>
      seriesApi.bulkMonitorSeries(ids, monitored),
    search: (id) => seriesApi.search(id),
    refresh: (id) => seriesApi.refresh(id),
    refreshAll: () => seriesApi.refreshAll(),
  },
}
