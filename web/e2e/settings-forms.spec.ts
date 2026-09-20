import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'

type FormPage = {
  path: string
  title: string
  /** A group header rendered on the page. */
  group: string
  /** The explanatory text that group carries in its footer. */
  footer: RegExp
  /** A control inside that group, located by role and accessible name. */
  control: { role: 'switch' | 'combobox' | 'spinbutton' | 'textbox'; name: string }
}

const FORM_PAGES: FormPage[] = [
  {
    path: '/settings/media/file-naming',
    title: 'Import & Naming',
    group: 'Allowed Video Extensions',
    footer: /Only files with one of these extensions/,
    control: { role: 'textbox', name: 'New video extension' },
  },
  {
    path: '/settings/download-pipeline/auto-search',
    title: 'Auto Search',
    group: 'Automatic Search',
    footer: /Scheduled task that periodically searches/,
    control: { role: 'switch', name: 'Enable Automatic Search' },
  },
  {
    path: '/settings/download-pipeline/rss-sync',
    title: 'RSS Sync',
    group: 'Feed Schedule',
    footer: /Periodically fetch RSS feeds/,
    control: { role: 'switch', name: 'Enable RSS Sync' },
  },
  {
    path: '/settings/general/server',
    title: 'Server',
    group: 'Log Rotation',
    footer: /Rotate when a file exceeds/,
    control: { role: 'switch', name: 'Compress Old Logs' },
  },
  {
    path: '/settings/general/authentication',
    title: 'Authentication',
    group: 'Display Name',
    footer: /Shown to users by their authenticator/,
    control: { role: 'textbox', name: 'Display Name' },
  },
]

function screen(page: Page, title: string): Locator {
  return page.getByRole('region', { name: title }).first()
}

async function bottomOf(locator: Locator): Promise<number> {
  const box = await locator.boundingBox()
  if (!box) {
    throw new Error('element has no box')
  }
  return box.y + box.height
}

async function topOf(locator: Locator): Promise<number> {
  const box = await locator.boundingBox()
  if (!box) {
    throw new Error('element has no box')
  }
  return box.y
}

test('every long-form settings page renders grouped control rows with footers', async ({
  page,
}) => {
  for (const formPage of FORM_PAGES) {
    await page.goto(formPage.path)
    await expect(page.getByRole('heading', { name: formPage.title, level: 1 })).toBeVisible()
    const region = screen(page, formPage.title)
    const header = region.getByRole('heading', { name: formPage.group, exact: true, level: 2 })
    await expect(header).toBeVisible()
    const control = region.getByRole(formPage.control.role, { name: formPage.control.name })
    await expect(control).toBeVisible()
    const footer = region.getByText(formPage.footer).first()
    await expect(footer).toBeVisible()
    expect(
      await topOf(footer),
      `${formPage.title}: the explanatory text sits below its group, not between controls`,
    ).toBeGreaterThanOrEqual(await bottomOf(control))
  }
})

test('an auto search switch and its threshold persist across a save and a reload', async ({
  page,
  activate,
}) => {
  await page.goto('/settings/download-pipeline/auto-search')
  const toggle = page.getByRole('switch', { name: 'Enable Automatic Search' })
  await expect(toggle).toBeVisible()
  const before = await toggle.isChecked()

  await activate(toggle)
  await expect(toggle).toBeChecked({ checked: !before })
  await activate(page.getByRole('button', { name: 'Save Changes' }))
  await expect(page.getByText('Settings saved')).toBeVisible()

  await page.reload()
  const reloaded = page.getByRole('switch', { name: 'Enable Automatic Search' })
  await expect(reloaded).toBeChecked({ checked: !before })

  await activate(reloaded)
  await expect(reloaded).toBeChecked({ checked: before })
  await activate(page.getByRole('button', { name: 'Save Changes' }))
  await expect(reloaded).toBeChecked({ checked: before })
})

test('changing a naming option updates the example and the filename preview', async ({
  page,
  activate,
}) => {
  await page.goto('/settings/media/file-naming')
  await expect(page.getByRole('heading', { name: 'Import & Naming' })).toBeVisible()
  await activate(page.getByRole('tab', { name: /Series Naming/ }))

  const region = screen(page, 'Import & Naming')
  const style = region.getByRole('combobox', { name: 'Multi-Episode Style' })
  await expect(style).toBeVisible()
  const example = region.getByText(/^Example: S01E/).first()
  const before = await example.textContent()

  await activate(style)
  await activate(page.getByRole('option', { name: 'Repeat', exact: true }))
  await expect.poll(async () => example.textContent()).not.toBe(before)

  const tester = region.getByRole('textbox', { name: 'Test filename' })
  await tester.fill('Breaking.Bad.S01E02.720p.BluRay.x264-DEMAND.mkv')
  await expect(region.getByText('TV Show')).toBeVisible()
})

test('the server form saves with its usual confirmation', async ({ page, activate }) => {
  await page.goto('/settings/general/server')
  await expect(page.getByRole('heading', { name: 'Server' })).toBeVisible()
  const region = screen(page, 'Server')
  const backups = region.getByRole('spinbutton', { name: 'Max Backup Files' })
  await expect(backups).toBeVisible()
  const before = await backups.inputValue()
  const next = before === '5' ? '6' : '5'

  await backups.fill(next)
  await activate(page.getByRole('button', { name: 'Save', exact: true }))
  await expect(page.getByText('Settings saved')).toBeVisible()

  await page.reload()
  await expect(
    screen(page, 'Server').getByRole('spinbutton', { name: 'Max Backup Files' }),
  ).toHaveValue(next)
})

test('the authentication form saves the relying party and opens the PIN dialog', async ({
  page,
  activate,
}) => {
  await page.goto('/settings/general/authentication')
  await expect(page.getByRole('heading', { name: 'Authentication' })).toBeVisible()
  const region = screen(page, 'Authentication')

  const displayName = region.getByRole('textbox', { name: 'Display Name' })
  await expect(displayName).toBeVisible()
  const before = await displayName.inputValue()
  const next = before === 'SlipStream' ? 'SlipStream E2E' : 'SlipStream'

  await displayName.fill(next)
  await expect(region.getByText('Restart required')).toBeVisible()
  await activate(region.getByRole('button', { name: 'Save' }))
  await expect(page.getByText(/Passkey relying party settings saved/)).toBeVisible()

  await activate(region.getByRole('button', { name: 'Change PIN' }))
  await expect(page.getByRole('heading', { name: 'Enter Current PIN' })).toBeVisible()
  await page.keyboard.press('Escape')
})

test('migrate from *arr opens from Media settings on its first step with a Media back label', async ({
  page,
  activate,
}) => {
  await page.goto('/settings/media')
  await expect(page.getByRole('heading', { name: 'Media' })).toBeVisible()
  await activate(screen(page, 'Media').getByRole('link', { name: /^Migrate from \*arr/ }))
  await expect(page.getByRole('heading', { name: 'Migrate from *arr' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Media' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Connect' }).last()).toBeVisible()
  await activate(page.getByRole('button', { name: 'Media' }))
  await expect(page.getByRole('heading', { name: 'Media' })).toBeVisible()
})

const TAPPABLE_INPUT =
  'input:not([type="range"]):not([type="hidden"]):not([type="checkbox"]):not([type="radio"]), textarea'

async function undersizedInputs(page: Page): Promise<string[]> {
  return page.locator(TAPPABLE_INPUT).evaluateAll((nodes) =>
    nodes
      .filter((node) => node instanceof HTMLElement && node.offsetParent !== null)
      .map((node) => {
        const style = globalThis.getComputedStyle(node)
        const height = node.getBoundingClientRect().height
        const fontSize = Number.parseFloat(style.fontSize)
        const name = node.getAttribute('aria-label') ?? node.id
        return { name, height, fontSize }
      })
      .filter((input) => input.fontSize < 16 || input.height < 44)
      .map((input) => `${input.name}: ${input.fontSize}px / ${input.height}px`),
  )
}

test('phone inputs are 16px and at least 44px tall on every long-form page', async ({
  page,
  activate,
}, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  for (const formPage of FORM_PAGES) {
    await page.goto(formPage.path)
    await expect(page.getByRole('heading', { name: formPage.title, level: 1 })).toBeVisible()
    await expect(
      screen(page, formPage.title).getByRole('heading', {
        name: formPage.group,
        exact: true,
        level: 2,
      }),
    ).toBeVisible()
    expect(await undersizedInputs(page), `${formPage.title} inputs`).toEqual([])
  }

  await page.goto('/settings/media/file-naming')
  for (const tab of [/Matching/, /Movie Naming/, /Series Naming/]) {
    await activate(page.getByRole('tab', { name: tab }))
    await expect(page.getByRole('tab', { name: tab })).toHaveAttribute('aria-selected', 'true')
    expect(await undersizedInputs(page), `Import & Naming ${String(tab)} inputs`).toEqual([])
  }
})
