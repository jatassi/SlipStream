import type { Locator } from '@playwright/test'
import { expect, test as base } from '@playwright/test'

import { activate as activateLocator, shellKind } from './helpers/activate'
import { captureConsoleErrors, unexpectedConsoleErrors } from './helpers/console'

type Fixtures = {
  activate: (locator: Locator) => Promise<void>
}

export const test = base.extend<Fixtures>({
  activate: async ({ context: _context }, use, testInfo) => {
    const kind = shellKind(testInfo.project.name)
    await use(async (locator: Locator) => {
      await activateLocator(locator, kind)
    })
  },
  page: async ({ page }, use) => {
    const errors = captureConsoleErrors(page)
    await use(page)
    const unexpected = unexpectedConsoleErrors(errors)
    expect(unexpected, unexpected.join('\n')).toEqual([])
  },
})

export { expect } from '@playwright/test'
