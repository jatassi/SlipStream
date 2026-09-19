import { MobileStateProvider } from '../../shared/state'
import { CinematicShell } from './cinematic-shell'

export function CinematicVariant() {
  return (
    <MobileStateProvider>
      <CinematicShell />
    </MobileStateProvider>
  )
}
