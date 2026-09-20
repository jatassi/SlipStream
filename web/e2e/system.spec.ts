import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import type { ShellKind } from './helpers/activate'
import { shellKind } from './helpers/activate'
import { ensureHealthIssue } from './helpers/dev-data'
import { primaryNav, sidebarNav } from './helpers/shell'

const RUNNABLE_TASK = 'History Cleanup'

function confirmSurface(page: Page): Locator {
  return page.getByRole('alertdialog').or(page.getByRole('dialog'))
}

function taskDetail(page: Page, name: string): Locator {
  return page
    .getByRole('region', { name: 'Scheduled Tasks' })
    .getByText(name, { exact: true })
    .locator('xpath=..')
}

async function openSystem(page: Page): Promise<void> {
  await page.goto('/system/health')
  await expect(page.getByRole('heading', { name: 'System' })).toBeVisible()
}

async function stubRunningTask(page: Page, name: string): Promise<void> {
  await page.route('**/api/v1/scheduler/tasks', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          id: 'stub-running',
          name,
          description: 'Stubbed task held in the running state',
          cron: '@every 1h',
          lastRun: new Date(Date.now() - 3_600_000).toISOString(),
          nextRun: new Date(Date.now() + 3_600_000).toISOString(),
          running: true,
        },
      ]),
    })
  })
}

async function stubUpdateCheck(page: Page): Promise<void> {
  await page.route('**/api/v1/update/check', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: { currentVersion: 'dev', state: 'up-to-date', progress: 0 },
        updateAvailable: false,
      }),
    })
  })
}

async function openSessionShell(page: Page, kind: ShellKind): Promise<void> {
  if (kind === 'phone') {
    await page.goto('/more')
    await expect(page.getByRole('heading', { name: 'More' })).toBeVisible()
    return
  }
  await page.goto('/')
  await expect(sidebarNav(page)).toBeVisible()
}

function sessionTrigger(page: Page, kind: ShellKind, name: string): Locator {
  const scope = kind === 'phone' ? page.getByRole('region', { name: 'More' }) : sidebarNav(page)
  return scope.getByRole('button', { name, exact: true })
}

test('system health lists issues as rows', async ({ page }) => {
  await openSystem(page)
  const issue = await ensureHealthIssue(page)
  await page.reload()
  await expect(page.getByRole('heading', { name: 'System' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Download Clients' })).toBeVisible()
  const region = page.getByRole('region', { name: 'System' })
  await expect(region.getByText(issue.name, { exact: true }).first()).toBeVisible()
  await expect(page.getByRole('button', { name: /^Test / }).first()).toBeVisible()
})

test('scheduled tasks list last and next run', async ({ page, activate }) => {
  await openSystem(page)
  await activate(page.getByRole('link', { name: 'Scheduled Tasks' }))
  await expect(page.getByRole('heading', { name: 'Scheduled Tasks' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'System' })).toBeVisible()
  await expect(taskDetail(page, RUNNABLE_TASK).getByText(/Last .+ · Next .+/)).toBeVisible()
})

test('a running task shows the running indicator', async ({ page }) => {
  await stubRunningTask(page, 'Stubbed Task')
  await page.goto('/system/tasks')
  await expect(page.getByRole('heading', { name: 'Scheduled Tasks' })).toBeVisible()
  const region = page.getByRole('region', { name: 'Scheduled Tasks' })
  await expect(region.getByRole('status', { name: 'Running' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Run Stubbed Task' })).toHaveCount(0)
})

test('running a task updates its last run', async ({ page, activate }) => {
  await page.goto('/system/tasks')
  await expect(page.getByRole('heading', { name: 'Scheduled Tasks' })).toBeVisible()
  const run = page.getByRole('button', { name: `Run ${RUNNABLE_TASK}` })
  await expect(run).toBeEnabled()
  await activate(run)
  await expect(page.getByText(`Started: ${RUNNABLE_TASK}`)).toBeVisible()
  await expect(taskDetail(page, RUNNABLE_TASK).getByText(/Last just now/)).toBeVisible()
})

test('logs and update open as sub-screens of system', async ({ page, activate }) => {
  await stubUpdateCheck(page)
  await openSystem(page)
  await activate(page.getByRole('link', { name: 'Logs' }))
  await expect(page.getByRole('heading', { name: 'Logs' })).toBeVisible()
  await expect(page.getByRole('log', { name: 'Log output' })).toBeVisible()
  const back = page.getByRole('button', { name: 'System' })
  await expect(back).toBeVisible()
  await activate(back)
  await expect(page.getByRole('heading', { name: 'System' })).toBeVisible()
  await activate(page.getByRole('link', { name: 'Update' }))
  await expect(page.getByRole('heading', { name: 'Update' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'System' })).toBeVisible()
})

test('restart confirms with a countdown and can be cancelled', async ({ page, activate }, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  await page.route('**/api/v1/system/restart', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ message: 'restarting' }),
    })
  })
  await openSessionShell(page, kind)

  await activate(sessionTrigger(page, kind, 'Restart'))
  await expect(confirmSurface(page).getByText('Restart SlipStream')).toBeVisible()
  await activate(confirmSurface(page).getByRole('button', { name: 'Cancel' }))
  await expect(confirmSurface(page)).toBeHidden()

  await activate(sessionTrigger(page, kind, 'Restart'))
  await expect(confirmSurface(page).getByText('Restart SlipStream')).toBeVisible()
  await activate(confirmSurface(page).getByRole('button', { name: 'Restart', exact: true }))
  await expect(confirmSurface(page).getByText(/Restarting \(\d+s\)/)).toBeVisible()
  await page.goto('/')
})

test('log out confirms and signs out', async ({ page, activate }, testInfo) => {
  const kind = shellKind(testInfo.project.name)
  await openSessionShell(page, kind)
  await activate(sessionTrigger(page, kind, 'Log out'))
  await expect(confirmSurface(page).getByText('You will need to sign in again.')).toBeVisible()
  await activate(confirmSurface(page).getByRole('button', { name: 'Log out', exact: true }))
  await page.waitForURL(/\/requests\/auth\/login/)
})

test('phone developer tools toggle the global loading state', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'phone', 'phone profile only')
  await page.goto('/')
  await activate(primaryNav(page).getByRole('link', { name: 'More' }))
  await expect(page.getByRole('heading', { name: 'Developer Tools' })).toBeVisible()
  await expectDeveloperToolsToggle(page, activate)
})

test('wide developer tools toggle the global loading state', async ({ page, activate }, testInfo) => {
  test.skip(shellKind(testInfo.project.name) !== 'wide', 'wide profile only')
  await page.goto('/')
  await activate(page.getByRole('button', { name: 'Developer Tools' }))
  await expectDeveloperToolsToggle(page, activate)
})

async function expectDeveloperToolsToggle(
  page: Page,
  activate: (locator: Locator) => Promise<void>,
): Promise<void> {
  const devMode = page.getByRole('switch', { name: 'Developer mode' })
  await expect(devMode).toBeVisible()
  await expect(devMode).toHaveAttribute('aria-checked', 'true')
  const label = page.getByText('Force Loading', { exact: true })
  await expect(label).toBeVisible()
  const loading = page.getByRole('switch', { name: 'Force Loading' })
  const before = await loading.isChecked()
  await activate(label)
  await expect(loading).toHaveAttribute('aria-checked', before ? 'false' : 'true')
  await activate(label)
  await expect(loading).toHaveAttribute('aria-checked', before ? 'true' : 'false')
}
