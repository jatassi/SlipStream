import { expect, test } from './fixtures'
import { shellKind } from './helpers/activate'
import { collapsePhoneSidebar } from './helpers/shell'

test('admin shell stays operable', async ({ page, activate }, testInfo) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  await collapsePhoneSidebar(page, activate, shellKind(testInfo.project.name))
  await activate(page.getByRole('link', { name: 'View all' }).first())
  await expect(page.getByRole('heading', { name: 'Downloads', exact: true })).toBeVisible()
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(page.getByRole('link', { name: /The Matrix/ })).toBeVisible()
})
