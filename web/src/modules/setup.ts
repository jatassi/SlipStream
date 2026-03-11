import { movieModuleConfig } from './movie'
import { registerModule } from './registry'
import { tvModuleConfig } from './tv'

export function setupModules(): void {
  registerModule(movieModuleConfig)
  registerModule(tvModuleConfig)
}
