import type { Locator } from '@playwright/test'

export type ShellKind = 'phone' | 'wide'

export function shellKind(projectName: string): ShellKind {
  return projectName === 'wide' ? 'wide' : 'phone'
}

export async function activate(locator: Locator, kind: ShellKind): Promise<void> {
  if (kind === 'phone') {
    await locator.tap()
    return
  }
  await locator.click()
}
