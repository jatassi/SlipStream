import { expect, test } from './fixtures'

test('admin shell stays operable', async ({ page, activate }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  await activate(page.getByRole('link', { name: 'Movies' }))
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(page.getByRole('link', { name: /The Matrix/ })).toBeVisible()
})
