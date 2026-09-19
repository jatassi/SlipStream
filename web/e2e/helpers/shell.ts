import type { Locator, Page } from '@playwright/test'

import type { ShellKind } from './activate'

export async function collapsePhoneSidebar(
  page: Page,
  activateControl: (locator: Locator) => Promise<void>,
  kind: ShellKind,
): Promise<void> {
  if (kind !== 'phone') {
    return
  }
  await activateControl(page.getByRole('button', { name: 'Collapse' }))
}
