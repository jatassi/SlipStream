import { MobileStateProvider } from '../../shared/state'
import { ConsoleShell } from './console-shell'

export function ConsoleVariant() {
  return (
    <MobileStateProvider>
      <ConsoleShell />
    </MobileStateProvider>
  )
}
