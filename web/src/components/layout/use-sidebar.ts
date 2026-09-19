import { useUIStore } from '@/stores'

import { useSessionActions } from './use-session-actions'

export function useSidebarActions() {
  const { sidebarCollapsed, toggleSidebar } = useUIStore()
  const session = useSessionActions()

  return {
    sidebarCollapsed,
    toggleSidebar,
    ...session,
  }
}
