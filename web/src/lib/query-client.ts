import { MutationCache, QueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import { ApiError } from '@/types'

const mutationCache = new MutationCache({
  // eslint-disable-next-line max-params
  onError(error, _variables, _context, mutation) {
    if (mutation.options.onError) {
      return
    }
    if (error instanceof ApiError) {
      toast.error(error.data?.message ?? error.data?.error ?? error.message)
    } else {
      toast.error(error instanceof Error ? error.message : 'Something went wrong')
    }
  },
})

export const queryClient = new QueryClient({
  mutationCache,
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5,
      gcTime: 1000 * 60 * 10,
      retry: 1,
    },
  },
})
