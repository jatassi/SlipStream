import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { ensureDownloading } from './helpers/dev-data'
import { applySafeArea, expectPhoneTabs, expectWideSidebarGroups, openActivity, primaryNav, sidebarNav } from './helpers/shell'

test('admin shell stays operable', async ({ page, activate }, testInfo) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  await openActivity(page, activate, shellKind(testInfo.project.name))
  await expect(page.getByRole('heading', { name: 'Downloads', exact: true })).toBeVisible()
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(page.getByRole('link', { name: /The Matrix/ })).toBeVisible()
})

test('phone tabs open destinations and mark the current tab', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/')
  await expectPhoneTabs(page)
  const nav = primaryNav(page)
  await expect(nav.getByRole('link', { name: 'Dashboard' })).toHaveAttribute('aria-current', 'page')
  await expect(
    page.getByText('Nothing downloading').or(nav.getByRole('link', { name: 'Activity' }).getByText(/[1-9]/)),
  ).toBeVisible()
  await activate(nav.getByRole('link', { name: 'Library' }))
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Library' })).toHaveAttribute('aria-current', 'page')
  await activate(nav.getByRole('link', { name: 'Activity' }))
  await expect(page.getByRole('heading', { name: 'Downloads', exact: true })).toBeVisible()
  await activate(nav.getByRole('link', { name: 'Search' }))
  await expect(page.getByRole('searchbox', { name: 'Search' })).toBeVisible()
  await activate(nav.getByRole('link', { name: 'More' }))
  await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'More' })).toHaveAttribute('aria-current', 'page')
})

test('the Activity tab badge counts active downloads', async ({ page }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/')
  await ensureDownloading(page)
  await page.reload()
  await expectPhoneTabs(page)
  const badge = primaryNav(page)
    .getByRole('link', { name: 'Activity', exact: false })
    .getByText(/^\d+$/)
  await expect(badge).toBeVisible()
  expect(Number(await badge.textContent())).toBeGreaterThanOrEqual(1)
})

test('phone tab scroll is restored after switching away', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  const heading = page.getByRole('heading', { name: 'Dashboard' })
  await scrollNearestScroller(heading, 600)
  await expect(heading).not.toBeInViewport()
  await activate(primaryNav(page).getByRole('link', { name: 'More' }))
  await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()
  await activate(primaryNav(page).getByRole('link', { name: 'Dashboard' }))
  await expect(page.getByRole('heading', { name: 'Dashboard' })).not.toBeInViewport()
})

test('phone tab bar stays visible with keyboard and safe-area insets', async ({ page }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/')
  await applySafeArea(page)
  await expectPhoneTabs(page)
  const tabBar = primaryNav(page)
  const before = await tabBar.boundingBox()
  await page.setViewportSize({ width: 390, height: 500 })
  await expect(tabBar).toBeVisible()
  const after = await tabBar.boundingBox()
  expect(before).not.toBeNull()
  expect(after).not.toBeNull()
  if (after === null) {
    return
  }
  expect(after.y + after.height).toBeLessThanOrEqual(500)
  expect(after.y + after.height).toBeGreaterThan(500 - 120)
})

test('phone more lists destinations and missing counts', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/more')
  await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Calendar' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Requests' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Missing' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Manual Import' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'History' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Media' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Download Pipeline' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'General' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'System' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Developer Tools' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Log out' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Restart' })).toBeVisible()
  await expectMissingCounts(page)
  await activate(page.getByRole('link', { name: 'Calendar' }))
  await expect(page.getByRole('heading', { name: /Calendar/ })).toBeVisible()
})

test('wide sidebar groups collapse and keep search plus developer tools', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'wide', 'wide profile only')
  await page.goto('/')
  await expectWideSidebarGroups(page)
  await expect(page.getByRole('searchbox', { name: 'Search' })).toBeVisible()
  await expect(page.getByRole('switch', { name: 'Developer mode' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Downloads' })).toBeVisible()
  await expect(primaryNav(page)).toHaveCount(0)
  await activate(page.getByRole('button', { name: 'Collapse sidebar' }))
  await expect(page.getByRole('button', { name: 'Expand sidebar' })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('button', { name: 'Expand sidebar' })).toBeVisible()
})

test('direct urls render inside the matching shell', async ({ page }, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  if (kind === 'phone') {
    await expectPhoneTabs(page)
    await expect(primaryNav(page).getByRole('link', { name: 'Library' })).toHaveAttribute(
      'aria-current',
      'page',
    )
  } else {
    await expect(sidebarNav(page)).toBeVisible()
  }
  await page.goto('/requests/auth/login')
  await expect(primaryNav(page)).toHaveCount(0)
  await expect(sidebarNav(page)).toHaveCount(0)
})

test('dashboard large title collapses after scroll', async ({ page }) => {
  await page.goto('/')
  const heading = page.getByRole('heading', { name: 'Dashboard' })
  await expect(heading).toBeVisible()
  const compact = page.getByRole('banner', { name: 'Dashboard' }).getByText('Dashboard', { exact: true })
  await expect(compact).toBeHidden()
  await scrollNearestScroller(heading, 800)
  await expect(heading).not.toBeInViewport()
  await expect(compact).toBeVisible()
})

test('wide keyboard traversal shows a focus ring on shell controls', async ({ page }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'wide', 'wide profile only')
  await page.goto('/')
  await expect(sidebarNav(page)).toBeVisible()
  await page.keyboard.press('Tab')
  const named = await focusedControlName(page)
  expect(named.length).toBeGreaterThan(0)
  expect(await focusedRingVisible(page)).toBe(true)
})

test('toasts sit above the tab bar on phone and bottom-right on wide', async ({ page, activate }, testInfo) => {
  await page.goto('/system/health')
  await expect(page.getByRole('heading', { name: 'System' })).toBeVisible()
  const testAll = page.getByRole('button', { name: 'Test All' }).first()
  await expect(testAll).toBeEnabled()
  await activate(testAll)
  const toast = page.getByText(/Download Clients:/)
  await expect(toast).toBeVisible()
  const viewport = page.viewportSize()
  expect(viewport).not.toBeNull()
  if (viewport === null) {
    return
  }
  if (shellKind(testInfo.project.name) === 'phone') {
    const tabBar = primaryNav(page)
    await expect.poll(async () => {
      const box = await toast.boundingBox()
      const tabBox = await tabBar.boundingBox()
      if (box === null || tabBox === null) {
        return Number.POSITIVE_INFINITY
      }
      return box.y + box.height - tabBox.y
    }).toBeLessThanOrEqual(8)
    return
  }
  const box = await toast.boundingBox()
  expect(box).not.toBeNull()
  if (box === null) {
    return
  }
  expect(box.x).toBeGreaterThan(viewport.width / 2)
  expect(box.y).toBeGreaterThan(viewport.height / 2)
})

async function scrollNearestScroller(locator: Locator, top: number): Promise<void> {
  await locator.evaluate((el, scrollTop) => {
    let node: Element | null = el
    while (node instanceof HTMLElement) {
      if (node.scrollHeight > node.clientHeight + 1) {
        node.scrollTop = scrollTop
        node.dispatchEvent(new Event('scroll'))
        return
      }
      node = node.parentElement
    }
  }, top)
}

async function expectMissingCounts(page: Page): Promise<void> {
  const missing = page.getByRole('link', { name: 'Missing' })
  await expect(missing).toBeVisible()
  await expect(missing).toContainText('|')
}

async function focusedControlName(page: Page): Promise<string> {
  return page.evaluate(() => {
    const el = document.activeElement
    if (!(el instanceof HTMLElement)) {
      return ''
    }
    const labelled = el.getAttribute('aria-label')
    if (labelled) {
      return labelled
    }
    const text = el.textContent.trim()
    if (text.length > 0) {
      return text
    }
    return el.tagName
  })
}

async function focusedRingVisible(page: Page): Promise<boolean> {
  return page.evaluate(() => {
    const el = document.activeElement
    if (!(el instanceof HTMLElement)) {
      return false
    }
    const style = getComputedStyle(el)
    return style.outlineWidth !== '0px' || style.boxShadow !== 'none'
  })
}
