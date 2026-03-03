import { getStatusConfig } from '@/lib/request-status-config'

export type { StatusConfigEntry } from '@/lib/request-status-config'

const base = getStatusConfig('md')

export const STATUS_CONFIG = {
  ...base,
  pending: {
    ...base.pending,
    label: 'Pending Approval',
  },
}
