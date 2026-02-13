import { useUIStore } from '@/stores'

export function useGlobalLoading() {
  return useUIStore((state) => state.globalLoading)
}
