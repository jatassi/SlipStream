import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { calendarKeys } from '@/hooks/use-calendar'
import { missingKeys } from '@/hooks/use-missing'
import type { ModuleApi, ModuleConfig, ModuleQueryKeys } from '@/modules/types'

function createQueryHooks(keys: ModuleQueryKeys, api: ModuleApi) {
  function useList(options?: Record<string, unknown>) {
    return useQuery({
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
      queryKey: [...keys.list(), options ?? {}],
      queryFn: () => api.list(options),
    })
  }

  function useDetail(id: number) {
    return useQuery({
      queryKey: keys.detail(id),
      queryFn: () => api.get(id),
      enabled: !!id,
    })
  }

  return { useList, useDetail }
}

function createCrudHooks(keys: ModuleQueryKeys, api: ModuleApi) {
  function useUpdate() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ id, data }: { id: number; data: Record<string, unknown> }) =>
        api.update(id, data),
      onSuccess: (entity) => {
        const typed = entity as { id: number }
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: missingKeys.all })
        void queryClient.setQueryData(keys.detail(typed.id), typed)
      },
    })
  }

  function useDelete() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ id, deleteFiles }: { id: number; deleteFiles?: boolean }) =>
        api.delete(id, deleteFiles),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: calendarKeys.all })
      },
    })
  }

  return { useUpdate, useDelete }
}

function createBulkHooks(keys: ModuleQueryKeys, api: ModuleApi) {
  function useBulkDelete() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ ids, deleteFiles }: { ids: number[]; deleteFiles?: boolean }) =>
        api.bulkDelete(ids, deleteFiles),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: calendarKeys.all })
      },
    })
  }

  function useBulkMonitor() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ ids, monitored }: { ids: number[]; monitored: boolean }) =>
        api.bulkMonitor(ids, monitored),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: missingKeys.all })
      },
    })
  }

  function useBulkUpdate() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: ({ ids, data }: { ids: number[]; data: Record<string, unknown> }) =>
        api.bulkUpdate(ids, data),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
        void queryClient.invalidateQueries({ queryKey: missingKeys.all })
      },
    })
  }

  return { useBulkDelete, useBulkMonitor, useBulkUpdate }
}

function createActionHooks(keys: ModuleQueryKeys, api: ModuleApi) {
  function useRefresh() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: (id: number) => api.refresh(id),
      onSuccess: (entity) => {
        const typed = entity as { id: number }
        void queryClient.setQueryData(keys.detail(typed.id), typed)
      },
    })
  }

  function useRefreshAll() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: () => api.refreshAll(),
      onSettled: () => {
        void queryClient.invalidateQueries({ queryKey: keys.all })
      },
    })
  }

  function useSearch() {
    return useMutation({
      mutationFn: (id: number) => api.search(id),
    })
  }

  return { useRefresh, useRefreshAll, useSearch }
}

export function createModuleHooks(mod: ModuleConfig) {
  const { queryKeys, api } = mod
  return {
    ...createQueryHooks(queryKeys, api),
    ...createCrudHooks(queryKeys, api),
    ...createBulkHooks(queryKeys, api),
    ...createActionHooks(queryKeys, api),
  }
}
