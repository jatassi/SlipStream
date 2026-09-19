import { MobileStateProvider } from '../../shared/state'
import { NativeShell } from './native-shell'

export function NativeVariant() {
  return (
    <MobileStateProvider>
      <NativeShell />
    </MobileStateProvider>
  )
}
