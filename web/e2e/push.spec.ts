import type { Locator } from '@playwright/test'

import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { sidebarNav } from './helpers/shell'

test('phone library item pushes detail and restores list scroll', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/movies')
  const heading = page.getByRole('heading', { name: 'Movies' })
  await expect(heading).toBeVisible()
  await scrollNearestScroller(heading, 140)
  const scrolled = await scrollTopOf(heading)
  expect(scrolled).toBeGreaterThan(0)
  await activate(page.getByRole('link', { name: /The Matrix/ }))
  await expect(page.getByRole('button', { name: 'Library' })).toBeVisible()
  await expect(page.getByRole('banner', { name: 'The Matrix' })).toBeVisible()
  await activate(page.getByRole('button', { name: 'Library' }))
  await expect(heading).toBeVisible()
  await expect.poll(async () => Math.abs((await scrollTopOf(heading)) - scrolled)).toBeLessThan(8)
})

test('phone settings from More back via button and browser history', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/more')
  await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()
  await activate(page.getByRole('link', { name: 'Media' }))
  await expect(page.getByRole('heading', { name: 'Media Management' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'More' })).toBeVisible()
  await activate(page.getByRole('button', { name: 'More' }))
  await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()
  await activate(page.getByRole('link', { name: 'Media' }))
  await expect(page.getByRole('heading', { name: 'Media Management' })).toBeVisible()
  await page.goBack()
  await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()
})

test('phone detail bar is transparent until the hero is scrolled away', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/movies')
  await activate(page.getByRole('link', { name: /The Matrix/ }))
  const bar = page.getByRole('banner', { name: 'The Matrix' })
  await expect(bar).toBeVisible()
  await expect(page.getByRole('button', { name: 'Library' })).toBeVisible()
  const compact = bar.getByText('The Matrix', { exact: true })
  await expect(compact).toBeHidden()
  const region = page.getByRole('region', { name: 'The Matrix' })
  await expect(region).toBeVisible()
  await scrollNearestScroller(region, 280)
  await expect(compact).toBeVisible()
})

test('wide navigations show a back control instantly', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'wide', 'wide profile only')
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await activate(page.getByRole('link', { name: /The Matrix/ }))
  await expect(page.getByRole('button', { name: 'Library' })).toBeVisible()
  await expect(page.getByRole('banner', { name: 'The Matrix' })).toBeVisible()
  await activate(page.getByRole('button', { name: 'Library' }))
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await activate(sidebarNav(page).getByRole('link', { name: 'Media' }))
  await expect(page.getByRole('button', { name: 'More' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Media Management' })).toBeVisible()
})

async function scrollNearestScroller(locator: Locator, top: number): Promise<void> {
  await locator.evaluate((el, scrollTop) => {
    if (el instanceof HTMLElement && el.scrollHeight > el.clientHeight + 1) {
      el.scrollTop = scrollTop
      el.dispatchEvent(new Event('scroll'))
      return
    }
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

async function scrollTopOf(locator: Locator): Promise<number> {
  return locator.evaluate((el) => {
    if (el instanceof HTMLElement && el.scrollHeight > el.clientHeight + 1) {
      return el.scrollTop
    }
    let node: Element | null = el
    while (node instanceof HTMLElement) {
      if (node.scrollHeight > node.clientHeight + 1) {
        return node.scrollTop
      }
      node = node.parentElement
    }
    return 0
  })
}
