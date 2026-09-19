import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { apiGet, ensureHealthIssue } from './helpers/dev-data'

type ListPage = {
  path: string
  title: string
  endpoint: string
  add?: string
  addDialog?: string
  /** Trailing detail expected on the first row, or 'switch' when the trailing slot is a toggle. */
  trailing: RegExp | 'switch'
}

const LIST_PAGES: ListPage[] = [
  {
    path: '/settings/media/root-folders',
    title: 'Root Folders',
    endpoint: '/rootfolders',
    add: 'Add root folder',
    addDialog: 'Add Root Folder',
    trailing: /Default|free/,
  },
  {
    path: '/settings/media/quality-profiles',
    title: 'Quality Profiles',
    endpoint: '/qualityprofiles',
    add: 'Add quality profile',
    addDialog: 'Add Quality Profile',
    trailing: /Movie|Series/,
  },
  {
    path: '/settings/media/version-slots',
    title: 'Version Slots',
    endpoint: '/slots',
    trailing: /Active|Disabled/,
  },
  {
    path: '/settings/download-pipeline/indexers',
    title: 'Indexers',
    endpoint: '/indexers',
    add: 'Add indexer',
    addDialog: 'Add Indexer',
    trailing: 'switch',
  },
  {
    path: '/settings/download-pipeline/clients',
    title: 'Download Clients',
    endpoint: '/downloadclients',
    add: 'Add download client',
    addDialog: 'Add Download Client',
    trailing: 'switch',
  },
  {
    path: '/settings/general/notifications',
    title: 'Notifications',
    endpoint: '/notifications',
    add: 'Add notification channel',
    addDialog: 'Add Notification',
    trailing: 'switch',
  },
]

function screen(page: Page, title: string): Locator {
  return page.getByRole('region', { name: title }).first()
}

function bar(page: Page, title: string): Locator {
  return page.getByRole('banner', { name: title })
}

function trailingOf(region: Locator, listPage: ListPage, first: string): Locator {
  return listPage.trailing === 'switch'
    ? region.getByRole('switch', { name: `${first} enabled` })
    : region.getByText(listPage.trailing).first()
}

async function itemNames(page: Page, endpoint: string): Promise<string[]> {
  const items = await apiGet<{ name: string }[]>(page, endpoint)
  return items.map((item) => item.name)
}

test('media settings lists its sub-sections with counts', async ({ page }) => {
  await page.goto('/settings/media')
  await expect(page.getByRole('heading', { name: 'Media' })).toBeVisible()
  const media = screen(page, 'Media')
  for (const name of ['Root Folders', 'Quality Profiles', 'Version Slots', 'Import & Naming', 'Migrate from *arr']) {
    await expect(media.getByRole('link', { name: new RegExp(`^${escapeRegExp(name)}`) })).toBeVisible()
  }
  await expect(media.getByRole('link', { name: /^Root Folders \d/ })).toBeVisible()
  await expect(media.getByRole('link', { name: /^Quality Profiles \d/ })).toBeVisible()
  await expect(media.getByRole('link', { name: /^Version Slots \d/ })).toBeVisible()
})

test('download pipeline and general settings list their sub-sections with counts', async ({ page }) => {
  await page.goto('/settings/download-pipeline')
  await expect(page.getByRole('heading', { name: 'Download Pipeline' })).toBeVisible()
  const pipeline = screen(page, 'Download Pipeline')
  await expect(pipeline.getByRole('link', { name: /^Indexers \d/ })).toBeVisible()
  await expect(pipeline.getByRole('link', { name: /^Download Clients \d/ })).toBeVisible()
  await expect(pipeline.getByRole('link', { name: 'Auto Search' })).toBeVisible()
  await expect(pipeline.getByRole('link', { name: 'RSS Sync' })).toBeVisible()

  await page.goto('/settings/general')
  await expect(page.getByRole('heading', { name: 'General' })).toBeVisible()
  const general = screen(page, 'General')
  await expect(general.getByRole('link', { name: 'Server' })).toBeVisible()
  await expect(general.getByRole('link', { name: 'Authentication' })).toBeVisible()
  await expect(general.getByRole('link', { name: /^Notifications \d/ })).toBeVisible()
})

test('a sub-section with a health issue shows a warning in its trailing detail', async ({ page }) => {
  await page.goto('/settings/media')
  await ensureHealthIssue(page)
  await page.reload()
  await expect(
    screen(page, 'Media').getByRole('link', { name: /Root Folders .*warning/ }),
  ).toBeVisible()
})

test('every list page renders its items as rows with a trailing detail', async ({ page }) => {
  for (const listPage of LIST_PAGES) {
    await page.goto(listPage.path)
    await expect(page.getByRole('heading', { name: listPage.title })).toBeVisible()
    const region = screen(page, listPage.title)
    const names = await itemNames(page, listPage.endpoint)
    expect(names.length, `${listPage.title} has dev-mode items`).toBeGreaterThan(0)
    for (const name of names) {
      await expect(region.getByText(name, { exact: true }).first()).toBeVisible()
    }
    await expect(trailingOf(region, listPage, names[0])).toBeVisible()
  }
})

test('the add action lives in the screen chrome and opens the create flow', async ({
  page,
  activate,
}, testInfo) => {
  const onPhone = shellKind(testInfo.project.name) === 'phone'
  for (const listPage of LIST_PAGES) {
    if (listPage.add === undefined) {
      continue
    }
    await page.goto(listPage.path)
    await expect(page.getByRole('heading', { name: listPage.title })).toBeVisible()
    const add = page.getByRole('button', { name: listPage.add })
    await expect(add).toHaveCount(1)
    if (onPhone) {
      await expect(bar(page, listPage.title).getByRole('button', { name: listPage.add })).toBeVisible()
    }
    await activate(add)
    await expect(page.getByRole('heading', { name: listPage.addDialog })).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByRole('heading', { name: listPage.addDialog })).toBeHidden()
  }
})

test('a row tap opens the existing edit flow', async ({ page, activate }) => {
  await page.goto('/settings/general/notifications')
  await activate(page.getByRole('button', { name: 'Edit Mock Notification' }))
  await expect(page.getByRole('heading', { name: 'Edit Notification' })).toBeVisible()
  await page.keyboard.press('Escape')

  await page.goto('/settings/download-pipeline/indexers')
  await activate(page.getByRole('button', { name: 'Edit Mock Indexer' }))
  await expect(page.getByRole('heading', { name: 'Edit Indexer' })).toBeVisible()
})

// Toggles the notification channel rather than an indexer or download client: those are
// shared with the autosearch other specs drive in parallel.
test('a switch row toggles and the value survives a reload', async ({ page, activate }) => {
  await page.goto('/settings/general/notifications')
  const toggle = page.getByRole('switch', { name: 'Mock Notification enabled' })
  await expect(toggle).toBeVisible()
  const before = await toggle.isChecked()
  await activate(toggle)
  await expect(toggle).toBeChecked({ checked: !before })
  await page.reload()
  const reloaded = page.getByRole('switch', { name: 'Mock Notification enabled' })
  await expect(reloaded).toBeChecked({ checked: !before })
  await activate(reloaded)
  await expect(reloaded).toBeChecked({ checked: before })
})

test('phone back reads the section name from a list page and More from a section', async ({
  page,
  activate,
}, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/more')
  await activate(screen(page, 'More').getByRole('link', { name: 'Media' }))
  await expect(page.getByRole('heading', { name: 'Media' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'More' })).toBeVisible()
  await activate(screen(page, 'Media').getByRole('link', { name: /^Root Folders/ }))
  await expect(page.getByRole('heading', { name: 'Root Folders' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Media' })).toBeVisible()
  await activate(page.getByRole('button', { name: 'Media' }))
  await expect(page.getByRole('heading', { name: 'Media' })).toBeVisible()
  await activate(page.getByRole('button', { name: 'More' }))
  await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()
})

test('phone back from every section leaf reads its section name', async ({ page }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  const leaves = [
    { path: '/settings/media/version-slots', back: 'Media' },
    { path: '/settings/media/file-naming', back: 'Media' },
    { path: '/settings/download-pipeline/indexers', back: 'Download Pipeline' },
    { path: '/settings/download-pipeline/rss-sync', back: 'Download Pipeline' },
    { path: '/settings/general/notifications', back: 'General' },
    { path: '/settings/general/server', back: 'General' },
  ]
  for (const leaf of leaves) {
    await page.goto(leaf.path)
    await expect(page.getByRole('button', { name: leaf.back })).toBeVisible()
  }
})

function escapeRegExp(value: string): string {
  return value.replaceAll(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)
}
