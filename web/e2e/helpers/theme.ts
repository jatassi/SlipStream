import type { Page } from '@playwright/test'

export type Theme = 'light' | 'dark'

export const THEMES: Theme[] = ['dark', 'light']

const UI_STORE_KEY = 'slipstream-ui'

/**
 * The theme is a UI-store preference with no control in the app, so a test picks
 * it by patching the persisted store before the page's own scripts run. Call it
 * before `page.goto`.
 */
export async function useTheme(page: Page, theme: Theme): Promise<void> {
  await page.addInitScript(
    ({ key, value }) => {
      const raw = localStorage.getItem(key)
      const parsed = raw === null ? {} : (JSON.parse(raw) as { state?: Record<string, unknown> })
      localStorage.setItem(
        key,
        JSON.stringify({ ...parsed, state: { ...parsed.state, theme: value } }),
      )
    },
    { key: UI_STORE_KEY, value: theme },
  )
}

export async function readThemeIsDark(page: Page): Promise<boolean> {
  return page.evaluate(() => document.documentElement.classList.contains('dark'))
}
