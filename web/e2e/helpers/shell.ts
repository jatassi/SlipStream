import { expect, type Locator, type Page } from '@playwright/test'

export function primaryNav(page: Page): Locator {
  return page.getByRole('navigation', { name: 'Primary' })
}

export function sidebarNav(page: Page): Locator {
  return page.getByRole('navigation', { name: 'Sidebar' })
}

export async function expectPhoneTabs(page: Page): Promise<void> {
  const nav = primaryNav(page)
  await expect(nav).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Dashboard' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Library' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Activity' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Search' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'More' })).toBeVisible()
}

export async function expectWideSidebarGroups(page: Page): Promise<void> {
  const nav = sidebarNav(page)
  await expect(nav).toBeVisible()
  await expect(nav.getByRole('heading', { name: 'Discover' })).toBeVisible()
  await expect(nav.getByRole('button', { name: 'Settings' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Calendar' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Requests' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Missing' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Manual Import' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'History' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Media' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Download Pipeline' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'General' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'System' })).toBeVisible()
}

export async function openActivity(
  page: Page,
  activateControl: (locator: Locator) => Promise<void>,
  kind: 'phone' | 'wide',
): Promise<void> {
  const target =
    kind === 'phone'
      ? primaryNav(page).getByRole('link', { name: 'Activity' })
      : page.getByRole('link', { name: 'Downloads' }).first()
  await activateControl(target)
}

export async function applySafeArea(page: Page): Promise<void> {
  await page.addStyleTag({
    content: ':root { --safe-top: 54px; --safe-bottom: 34px; }',
  })
}
