import type { Locator, Page } from '@playwright/test'

import { expect, test } from './fixtures'
import { apiBase } from './helpers/paths'

type PortalRequest = {
  id: number
  title: string
  status: string
}

type Invitation = {
  id: number
  username: string
  token: string
  usedAt: string | null
}

type PortalUser = {
  id: number
  username: string
}

type Movie = {
  id: number
  title: string
}

async function bearerToken(page: Page): Promise<string> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('slipstream-portal-auth')
    if (!raw) {
      throw new Error('missing portal auth')
    }
    const parsed = JSON.parse(raw) as { state?: { token?: string } }
    const token = parsed.state?.token
    if (!token) {
      throw new Error('missing auth token')
    }
    return token
  })
}

async function adminApi<T>(
  page: Page,
  path: string,
  init?: { method?: string; data?: unknown },
): Promise<T> {
  const token = await bearerToken(page)
  const response = await page.request.fetch(`${apiBase}${path}`, {
    method: init?.method ?? 'GET',
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: init?.data,
  })
  if (!response.ok()) {
    throw new Error(`${path} failed: ${response.status()}`)
  }
  return (await response.json()) as T
}

async function adminDelete(page: Page, path: string): Promise<void> {
  const token = await bearerToken(page)
  await page.request.fetch(`${apiBase}${path}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
}

/** A name no other spec or project shares, so the two shells never collide. */
function uniqueName(projectName: string, purpose: string): string {
  return `reqadmin-${purpose}-${projectName}-${Date.now()}`
}

/**
 * A request only reaches the queue when its requester does not auto-approve, so
 * the admin's own auto-approve and per-module profiles are cleared first. Both
 * projects write the same values, so they never fight over them.
 */
async function requestAsAdmin(page: Page): Promise<PortalUser> {
  const users = await adminApi<PortalUser[]>(page, '/admin/requests/users')
  const admin = users.find((user) => user.username === 'Administrator')
  if (!admin) {
    throw new Error('developer mode has no Administrator portal user')
  }
  await adminApi(page, `/admin/requests/users/${admin.id}`, {
    method: 'PUT',
    data: { autoApprove: false, moduleSettings: { movie: null, tv: null } },
  })
  return admin
}

async function createPendingRequest(page: Page, title: string): Promise<PortalRequest> {
  await requestAsAdmin(page)
  const request = await adminApi<PortalRequest>(page, '/requests', {
    method: 'POST',
    data: { mediaType: 'movie', title, year: 2019 },
  })
  expect(request.status, 'the new request waits for approval').toBe('pending')
  return request
}

async function cleanUp(page: Page, title: string): Promise<void> {
  const requests = await adminApi<PortalRequest[]>(page, '/admin/requests')
  const mine = requests.find((request) => request.title === title)
  if (mine) {
    await adminDelete(page, `/admin/requests/${mine.id}`)
  }
  const movies = await adminApi<Movie[]>(page, '/movies')
  const added = movies.find((movie) => movie.title === title)
  if (added) {
    await adminDelete(page, `/movies/${added.id}`)
  }
}

function screen(page: Page, title: string): Locator {
  return page.getByRole('region', { name: title }).first()
}

function requestRow(page: Page, title: string): Locator {
  return screen(page, 'Requests').getByRole('group', { name: title, exact: true })
}

async function openQueue(page: Page): Promise<void> {
  await page.goto('/requests-admin/queue')
  await expect(page.getByRole('heading', { name: 'Requests', level: 1 })).toBeVisible()
}

async function selectSegment(
  page: Page,
  activate: (locator: Locator) => Promise<void>,
  name: string,
): Promise<void> {
  await activate(page.getByRole('radio', { name, exact: true }))
  await expect(page.getByRole('radio', { name, exact: true })).toBeChecked()
}

test('the queue segments the states and lists pending requests as rows', async ({
  page,
  activate,
}, testInfo) => {
  const title = uniqueName(testInfo.project.name, 'listing')
  await page.goto('/requests-admin/queue')
  await createPendingRequest(page, title)

  try {
    await openQueue(page)
    const segments = page.getByRole('radiogroup', { name: 'Request status' })
    await expect(segments).toBeVisible()
    for (const label of ['Pending', 'Approved', 'Downloading', 'Available', 'Denied']) {
      await expect(segments.getByRole('radio', { name: label, exact: true })).toBeVisible()
    }
    await expect(segments.getByRole('radio', { name: 'Pending', exact: true })).toBeChecked()

    const row = requestRow(page, title)
    await expect(row).toBeVisible()
    await expect(row.getByText('Pending', { exact: true })).toBeVisible()
    await expect(row.getByText(/Movie.*by Administrator/)).toBeVisible()
    await expect(row.getByRole('button', { name: `Approve ${title}` })).toBeVisible()
    await expect(row.getByRole('button', { name: `Deny ${title}` })).toBeVisible()

    await selectSegment(page, activate, 'Denied')
    await expect(requestRow(page, title)).toBeHidden()
    await selectSegment(page, activate, 'Pending')
    await expect(requestRow(page, title)).toBeVisible()
  } finally {
    await cleanUp(page, title)
  }
})

test('approving a pending request moves it out of Pending', async ({ page, activate }, testInfo) => {
  const title = uniqueName(testInfo.project.name, 'approve')
  await page.goto('/requests-admin/queue')
  await createPendingRequest(page, title)

  try {
    await openQueue(page)
    await expect(requestRow(page, title)).toBeVisible()
    await activate(requestRow(page, title).getByRole('button', { name: `Approve ${title}` }))

    await expect(requestRow(page, title)).toBeHidden()
    await expect
      .poll(async () => {
        const requests = await adminApi<PortalRequest[]>(page, '/admin/requests')
        return requests.find((request) => request.title === title)?.status ?? 'missing'
      })
      .not.toBe('pending')

    await selectSegment(page, activate, 'Approved')
    if (!(await requestRow(page, title).isVisible())) {
      await selectSegment(page, activate, 'Downloading')
    }
    await expect(requestRow(page, title)).toBeVisible()
  } finally {
    await cleanUp(page, title)
  }
})

test('deny confirms before the request moves to Denied', async ({ page, activate }, testInfo) => {
  const title = uniqueName(testInfo.project.name, 'deny')
  await page.goto('/requests-admin/queue')
  await createPendingRequest(page, title)

  try {
    await openQueue(page)
    await activate(requestRow(page, title).getByRole('button', { name: `Deny ${title}` }))

    await expect(page.getByText(`Deny ${title}?`)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()
    await activate(page.getByRole('button', { name: 'Deny request' }))

    await expect(requestRow(page, title)).toBeHidden()
    await selectSegment(page, activate, 'Denied')
    const denied = requestRow(page, title)
    await expect(denied).toBeVisible()
    await expect(denied.getByText('Denied', { exact: true })).toBeVisible()
  } finally {
    await cleanUp(page, title)
  }
})

test('Users lists portal users and an invited user becomes a row', async ({
  page,
  activate,
}, testInfo) => {
  const username = uniqueName(testInfo.project.name, 'invite')
  await page.goto('/requests-admin/users')
  await expect(page.getByRole('heading', { name: 'Users', level: 1 })).toBeVisible()

  const region = screen(page, 'Users')
  await expect(region.getByRole('heading', { name: 'Users', exact: true, level: 2 })).toBeVisible()
  await expect(region.getByRole('heading', { name: 'Invitations', level: 2 })).toBeVisible()
  await expect(region.getByText('Administrator', { exact: true })).toBeVisible()

  await activate(page.getByRole('button', { name: 'Edit Administrator' }))
  await expect(page.getByText('Configure settings for Administrator')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByText('Configure settings for Administrator')).toBeHidden()

  await activate(page.getByRole('button', { name: 'Invite user' }))
  await expect(page.getByText('The name you enter becomes their portal username.')).toBeVisible()
  await page.getByRole('textbox', { name: 'Name' }).fill(username)
  await activate(page.getByRole('button', { name: 'Create Invitation' }))
  await expect(region.getByText(username, { exact: true })).toBeVisible()

  const invitations = await adminApi<Invitation[]>(page, '/admin/requests/invitations')
  const invitation = invitations.find((item) => item.username === username)
  expect(invitation, 'the invitation the sheet created is listed').toBeDefined()

  try {
    await page.reload()
    const reloaded = screen(page, 'Users')
    await expect(reloaded.getByText(username, { exact: true })).toBeVisible()
    await expect(reloaded.getByText(/Pending · created/).first()).toBeVisible()
  } finally {
    await adminDelete(page, `/admin/requests/invitations/${invitation?.id ?? 0}`)
  }
})

test('Request Settings renders grouped controls and saves', async ({ page, activate }, testInfo) => {
  await page.goto('/requests-admin/settings')
  await expect(page.getByRole('heading', { name: 'Request Settings', level: 1 })).toBeVisible()

  const region = screen(page, 'Request Settings')
  for (const group of ['Portal Access', 'Default Quotas', 'Content', 'Rate Limiting']) {
    await expect(region.getByRole('heading', { name: group, exact: true, level: 2 })).toBeVisible()
  }
  await expect(region.getByRole('switch', { name: 'Enable External Requests Portal' })).toBeChecked()
  await expect(region.getByText(/Weekly limits applied to new users/)).toBeVisible()

  // The two projects write different fields: request settings are global, so
  // sharing one field would let the shells overwrite each other mid-assert.
  const field = testInfo.project.name === 'wide' ? 'Movie per Week' : 'Searches per Minute'
  const next = testInfo.project.name === 'wide' ? '7' : '21'
  const control = region.getByRole('spinbutton', { name: field })
  await expect(control).toBeVisible()
  await control.fill(next)
  await activate(page.getByRole('button', { name: 'Save Changes' }))
  await expect(page.getByText('Settings saved')).toBeVisible()

  await page.reload()
  await expect(
    screen(page, 'Request Settings').getByRole('spinbutton', { name: field }),
  ).toHaveValue(next)
})

test('the requests portal login page still renders', async ({ page }) => {
  await page.goto('/requests/auth/login')
  await expect(page.getByText('Welcome Back')).toBeVisible()
  await expect(page.getByText('Sign in to your SlipStream account')).toBeVisible()
})
