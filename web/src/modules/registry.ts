import type { ModuleConfig } from './types'

const modules = new Map<string, ModuleConfig>()
const disabledModules = new Set<string>()

export function registerModule(config: ModuleConfig): void {
  if (modules.has(config.id)) {
    throw new Error(`Module "${config.id}" is already registered`)
  }
  modules.set(config.id, config)
}

export function getModule(id: string): ModuleConfig | undefined {
  return modules.get(id)
}

export function getModuleOrThrow(id: string): ModuleConfig {
  const mod = modules.get(id)
  if (!mod) {throw new Error(`Module "${id}" not found`)}
  return mod
}

export function getAllModules(): ModuleConfig[] {
  return [...modules.values()]
}

export function getEnabledModules(): ModuleConfig[] {
  return getAllModules().filter((mod) => !disabledModules.has(mod.id))
}

export function setModuleEnabledState(enabledMap: Record<string, boolean>): void {
  disabledModules.clear()
  for (const [id, enabled] of Object.entries(enabledMap)) {
    if (!enabled) {
      disabledModules.add(id)
    }
  }
}
