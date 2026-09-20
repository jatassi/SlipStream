import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import type { Activate, ListForm, Named } from './helpers/presenter'
import {
  api,
  boxOf,
  dragDown,
  openAdd,
  pickOption,
  removeNamed,
  stubSlotSetup,
  surface,
  tag,
  undersizedFields,
  viewportOf,
} from './helpers/presenter'

const INDEXERS: ListForm = {
  path: '/settings/download-pipeline/indexers',
  screen: 'Indexers',
  add: 'Add indexer',
  title: 'Add Indexer',
}
const CLIENTS: ListForm = {
  path: '/settings/download-pipeline/clients',
  screen: 'Download Clients',
  add: 'Add download client',
  title: 'Add Download Client',
}
const PROFILES: ListForm = {
  path: '/settings/media/quality-profiles',
  screen: 'Quality Profiles',
  add: 'Add quality profile',
  title: 'Add Quality Profile',
}
const CHANNELS: ListForm = {
  path: '/settings/general/notifications',
  screen: 'Notifications',
  add: 'Add notification channel',
  title: 'Add Notification',
}
const FOLDERS: ListForm = {
  path: '/settings/media/root-folders',
  screen: 'Root Folders',
  add: 'Add root folder',
  title: 'Add Root Folder',
}

/** The sheet slides up, so its resting place is what an anchoring check is about. */
async function expectAnchored(form: Locator, viewportHeight: number): Promise<void> {
  await expect
    .poll(async () => {
      const shape = await boxOf(form)
      return Math.abs(shape.y + shape.height - viewportHeight)
    })
    .toBeLessThan(2)
}

test('the add form is a bottom sheet on phones and a centred dialog on wide screens', async ({
  page,
  activate,
}, testInfo) => {
  const form = await openAdd(page, INDEXERS, activate)
  await expect(form).toBeVisible()

  const viewport = viewportOf(page)
  if (shellKind(testInfo.project.name) === 'phone') {
    await expectAnchored(form, viewport.height)
    const shape = await boxOf(form)
    expect(shape.x).toBeLessThan(2)
  } else {
    const shape = await boxOf(form)
    expect(shape.y).toBeGreaterThan(0)
    expect(shape.y + shape.height).toBeLessThan(viewport.height)
    expect(shape.x).toBeGreaterThan(0)
  }

  await page.keyboard.press('Escape')
  await expect(form).toBeHidden()
})

test('an indexer created through the presenter appears as a row', async ({ page, activate }, testInfo) => {
  const name = `${tag(shellKind(testInfo.project.name))} Indexer`
  const picker = await openAdd(page, INDEXERS, activate)

  try {
    await picker.getByPlaceholder('Search definitions...').fill('Generic RSS')
    await activate(picker.getByRole('row', { name: /Generic RSS/ }).first())

    const form = surface(page, 'Configure Indexer')
    await expect(form).toBeVisible()
    await form.getByRole('textbox', { name: 'Name' }).fill(name)
    await activate(form.getByRole('button', { name: 'Add', exact: true }))

    await expect(form).toBeHidden()
    await expect(
      page.getByRole('region', { name: 'Indexers' }).getByText(name, { exact: true }),
    ).toBeVisible()
  } finally {
    await removeNamed(page, '/indexers', name)
  }
})

test('dragging the add sheet down dismisses it without saving', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  const name = `${tag('phone')} Dragged Indexer`
  const picker = await openAdd(page, INDEXERS, activate)
  await picker.getByPlaceholder('Search definitions...').fill('Generic RSS')
  await activate(picker.getByRole('row', { name: /Generic RSS/ }).first())

  const form = surface(page, 'Configure Indexer')
  await expect(form).toBeVisible()
  await form.getByRole('textbox', { name: 'Name' }).fill(name)

  const shape = await boxOf(form)
  await dragDown(page, { x: shape.x + shape.width / 2, y: shape.y + 14 }, 460)

  await expect(form).toBeHidden()
  const indexers = await api<Named[]>(page, '/indexers')
  expect(indexers.map((indexer) => indexer.name)).not.toContain(name)
})

test('a download client created through the presenter appears as a row', async ({ page, activate }, testInfo) => {
  const name = `${tag(shellKind(testInfo.project.name))} Client`
  const form = await openAdd(page, CLIENTS, activate)

  try {
    await expect(form).toBeVisible()
    await pickOption(page, { form, label: 'Client Type', option: 'Mock' }, activate)
    await form.getByRole('textbox', { name: 'Name' }).fill(name)
    await activate(form.getByRole('button', { name: 'Add', exact: true }))

    await expect(form).toBeHidden()
    await expect(
      page.getByRole('region', { name: CLIENTS.screen }).getByText(name, { exact: true }),
    ).toBeVisible()
  } finally {
    await removeNamed(page, '/downloadclients', name)
  }
})

test('a quality profile created through the presenter appears as a row', async ({ page, activate }, testInfo) => {
  const name = `${tag(shellKind(testInfo.project.name))} Profile`
  const form = await openAdd(page, PROFILES, activate)

  try {
    await expect(form).toBeVisible()
    await form.getByRole('textbox', { name: 'Name' }).fill(name)
    await pickOption(page, { form, label: 'Module Type', option: 'Movies' }, activate)
    await activate(form.getByRole('button', { name: 'Create' }))

    await expect(form).toBeHidden()
    await expect(
      page.getByRole('region', { name: PROFILES.screen }).getByText(name, { exact: true }),
    ).toBeVisible()
  } finally {
    await removeNamed(page, '/qualityprofiles', name)
  }
})

test('a notification created through the presenter appears as a row', async ({ page, activate }, testInfo) => {
  const name = `${tag(shellKind(testInfo.project.name))} Channel`
  const form = await openAdd(page, CHANNELS, activate)

  try {
    await expect(form).toBeVisible()
    await pickOption(page, { form, label: 'Type', option: 'Mock (Dev Mode)' }, activate)
    await form.getByRole('textbox', { name: 'Name' }).fill(name)
    await activate(form.getByRole('button', { name: 'Add', exact: true }))

    await expect(form).toBeHidden()
    await expect(
      page.getByRole('region', { name: CHANNELS.screen }).getByText(name, { exact: true }),
    ).toBeVisible()
  } finally {
    await removeNamed(page, '/notifications', name)
  }
})

type Entry = { name: string; path: string; isDir: boolean }

test('a root folder created with the folder browser inside the presenter appears as a row', async ({
  page,
  activate,
}, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  const name = `${tag(kind)} Folder`
  await page.goto(FOLDERS.path)
  const listing = await api<{ entries?: Entry[] }>(page, '/filesystem/browse?path=%2Fmock%2Fmovies')
  const target = (listing.entries ?? []).filter((entry) => entry.isDir).at(kind === 'phone' ? 0 : 1)
  if (!target) {
    throw new Error('the developer filesystem has too few folders under /mock/movies')
  }

  try {
    const form = await openAdd(page, FOLDERS, activate)
    await expect(form).toBeVisible()
    await browseTo(page, { form, target }, activate)

    await expect(form.getByRole('textbox', { name: 'Path' })).toHaveValue(target.path)
    await form.getByRole('textbox', { name: 'Name' }).fill(name)
    await activate(form.getByRole('button', { name: 'Add', exact: true }))

    await expect(form).toBeHidden()
    await expect(
      page.getByRole('region', { name: FOLDERS.screen }).getByText(name, { exact: true }),
    ).toBeVisible()
  } finally {
    await removeNamed(page, '/rootfolders', name)
  }
})

async function browseTo(
  page: Page,
  pick: { form: Locator; target: Entry },
  activate: Activate,
): Promise<void> {
  const { form, target } = pick
  await activate(form.getByRole('button', { name: 'Browse folders' }))
  const browser = surface(page, 'Browse Folders')
  await expect(browser).toBeVisible()
  await browser.getByRole('textbox', { name: 'Folder path' }).fill('/mock/movies')
  await activate(browser.getByRole('button', { name: 'Go' }))
  await activate(browser.getByRole('button', { name: target.name, exact: true }))
  await activate(browser.getByRole('button', { name: 'Select Folder' }))
  await expect(browser).toBeHidden()
}

test('activating a row opens the edit form pre-filled and saving updates the row', async ({
  page,
  activate,
}, testInfo) => {
  const before = `${tag(shellKind(testInfo.project.name))} Edit Channel`
  const after = `${before} Renamed`
  await page.goto(CHANNELS.path)
  await api(page, '/notifications', {
    method: 'POST',
    data: { name: before, type: 'mock', enabled: true, settings: {}, eventToggles: { grab: true } },
  })

  try {
    await page.reload()
    await activate(page.getByRole('button', { name: `Edit ${before}` }))
    const form = surface(page, 'Edit Notification')
    await expect(form).toBeVisible()
    const nameField = form.getByRole('textbox', { name: 'Name' })
    await expect(nameField).toHaveValue(before)

    await nameField.fill(after)
    await activate(form.getByRole('button', { name: 'Save' }))
    await expect(form).toBeHidden()

    const region = page.getByRole('region', { name: CHANNELS.screen })
    await expect(region.getByText(after, { exact: true })).toBeVisible()
    await expect(region.getByText(before, { exact: true })).toHaveCount(0)
  } finally {
    await removeNamed(page, '/notifications', before)
    await removeNamed(page, '/notifications', after)
  }
})

test('the version-slot dry run, the resolve flows and the token builder open through the presenter', async ({
  page,
  activate,
}) => {
  await page.goto('/settings/media/version-slots')
  const profiles = await api<Named[]>(page, '/qualityprofiles')
  if (profiles.length === 0) {
    throw new Error('developer mode has no quality profile')
  }
  await stubSlotSetup(page, profiles[0].id)
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Version Slots', level: 1 })).toBeVisible()

  await activate(page.getByRole('button', { name: 'Begin' }))
  const dryRun = surface(page, 'Migration Dry Run Preview')
  await expect(dryRun).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(dryRun).toBeHidden()

  const profileAlert = page.getByRole('alert').filter({ hasText: 'Quality Profiles' })
  await activate(profileAlert.getByRole('button', { name: 'Validate' }))
  await activate(profileAlert.getByRole('button', { name: 'Resolve...' }))
  const conflicts = surface(page, 'Resolve Profile Conflicts')
  await expect(conflicts).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(conflicts).toBeHidden()

  const namingAlert = page.getByRole('alert').filter({ hasText: 'File Naming' })
  await activate(namingAlert.getByRole('button', { name: 'Validate' }))
  await activate(namingAlert.getByRole('button', { name: 'Resolve...' }))
  const naming = surface(page, 'Resolve Naming Format Issues')
  await expect(naming).toBeVisible()

  await activate(naming.getByRole('button', { name: 'Standard Episode Format' }))
  await expect(surface(page, 'Edit Pattern')).toBeVisible()
})

test('every field in a presented form is 16px and at least 44px tall', async ({ page, activate }) => {
  for (const entry of [CLIENTS, CHANNELS, PROFILES, FOLDERS]) {
    const form = await openAdd(page, entry, activate)
    await expect(form).toBeVisible()
    // Polled: the surface scales as it opens, and a field measured mid-entrance
    // is a rounding error short of its resting height.
    await expect.poll(async () => undersizedFields(form), { message: `${entry.title} fields` }).toEqual([])
    await page.keyboard.press('Escape')
    await expect(form).toBeHidden()
  }
})

test('focusing a field inside a sheet leaves the viewport scale unchanged and keeps the sheet anchored', async ({
  page,
  activate,
}, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  const form = await openAdd(page, CLIENTS, activate)
  await expect(form).toBeVisible()

  const viewport = viewportOf(page)
  const before = await page.evaluate(() => globalThis.visualViewport?.scale ?? 1)

  const host = form.getByRole('textbox', { name: 'Host' })
  await activate(host)
  await host.focus()
  expect(await page.evaluate(() => globalThis.visualViewport?.scale ?? 1)).toBe(before)

  await expectAnchored(form, viewport.height)
  const field = await boxOf(host)
  const shape = await boxOf(form)
  expect(field.y).toBeGreaterThanOrEqual(shape.y)
  expect(field.y + field.height).toBeLessThanOrEqual(shape.y + shape.height + 1)
})
