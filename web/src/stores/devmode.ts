import { create } from 'zustand'

type DevModeState = {
  enabled: boolean
  switching: boolean
  setEnabled: (enabled: boolean) => void
  setSwitching: (switching: boolean) => void
}

export const useDevModeStore = create<DevModeState>((set) => ({
  enabled: false,
  switching: false,
  setEnabled: (enabled) => set({ enabled }),
  setSwitching: (switching) => set({ switching }),
}))
