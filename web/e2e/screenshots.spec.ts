import { expect, test } from './fixtures'

const screenshotOptions = {
  animations: 'disabled' as const,
  maxDiffPixelRatio: 0.08,
}

test('dashboard screenshot comparison', async ({ page }) => {
  test.skip(process.env.PLAYWRIGHT_SCREENSHOTS !== '1', 'screenshot comparison is opt-in')
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  await expect(page).toHaveScreenshot('dashboard.png', screenshotOptions)
})

test('movies list screenshot comparison', async ({ page }) => {
  test.skip(process.env.PLAYWRIGHT_SCREENSHOTS !== '1', 'screenshot comparison is opt-in')
  await page.goto('/movies')
  await expect(page.getByRole('heading', { name: 'Movies' })).toBeVisible()
  await expect(page.getByRole('link', { name: /The Matrix/ })).toBeVisible()
  await expect(page).toHaveScreenshot('movies.png', screenshotOptions)
})
