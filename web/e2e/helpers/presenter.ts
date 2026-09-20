import type { Locator, Page } from '@playwright/test'

import type { ShellKind } from './activate'
import { apiBase } from './paths'

export type Activate = (locator: Locator) => Promise<void>

/** Names are unique per spec and per project so the two shells never collide. */
export function tag(kind: ShellKind): string {
  return `Presenter ${kind === 'phone' ? 'Phone' : 'Wide'}`
}

/** The presented form, whichever surface the shell chose for it. */
export function surface(page: Page, title: string): Locator {
  return page
    .getByRole('dialog')
    .filter({ has: page.getByRole('heading', { name: title, exact: true }) })
}

async function bearer(page: Page): Promise<string> {
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

export async function api<T>(
  page: Page,
  path: string,
  init?: { method?: string; data?: unknown },
): Promise<T> {
  const response = await page.request.fetch(`${apiBase}${path}`, {
    method: init?.method ?? 'GET',
    headers: {
      Authorization: `Bearer ${await bearer(page)}`,
      'Content-Type': 'application/json',
    },
    data: init?.data,
  })
  if (!response.ok()) {
    throw new Error(`${path} failed: ${response.status()}`)
  }
  return (await response.json()) as T
}

export type Named = { id: number; name: string }

export async function removeNamed(page: Page, endpoint: string, name: string): Promise<void> {
  const items = await api<Named[]>(page, endpoint)
  for (const item of items.filter((candidate) => candidate.name === name)) {
    await page.request.fetch(`${apiBase}${endpoint}/${item.id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${await bearer(page)}` },
    })
  }
}

export type Box = { x: number; y: number; width: number; height: number }

export async function boxOf(locator: Locator): Promise<Box> {
  const measured = await locator.boundingBox()
  if (!measured) {
    throw new Error('element has no box')
  }
  return measured
}

export function viewportOf(page: Page): { width: number; height: number } {
  const size = page.viewportSize()
  if (!size) {
    throw new Error('the project has no viewport size')
  }
  return size
}

// vaul ignores a drag that starts within 500ms of the sheet opening, so the
// gesture waits for the entrance to settle before pulling the sheet down.
export async function dragDown(page: Page, from: { x: number; y: number }, distance: number): Promise<void> {
  await page.waitForTimeout(700)
  await page.mouse.move(from.x, from.y)
  await page.mouse.down()
  for (let step = 1; step <= 8; step += 1) {
    await page.mouse.move(from.x, from.y + (distance / 8) * step)
  }
  await page.mouse.up()
}

export type ListForm = { path: string; screen: string; add: string; title: string }

export async function openAdd(page: Page, form: ListForm, activate: Activate): Promise<Locator> {
  await page.goto(form.path)
  const heading = page.getByRole('heading', { name: form.screen, level: 1 })
  await heading.waitFor()
  await activate(page.getByRole('button', { name: form.add }))
  return surface(page, form.title)
}

export async function pickOption(
  page: Page,
  field: { form: Locator; label: string; option: string },
  activate: Activate,
): Promise<void> {
  await activate(field.form.getByRole('combobox', { name: field.label }))
  await activate(page.getByRole('option', { name: field.option, exact: true }))
}

const TAPPABLE =
  'input:not([type="range"]):not([type="hidden"]):not([type="checkbox"]):not([type="radio"]), textarea, [data-slot="select-trigger"]'

/** Every control a presented form shows, reported when it is under 16px or 44px. */
export async function undersizedFields(form: Locator): Promise<string[]> {
  return form.locator(TAPPABLE).evaluateAll((nodes) =>
    nodes
      .filter((node) => node instanceof HTMLElement && node.offsetParent !== null)
      .map((node) => {
        const style = globalThis.getComputedStyle(node)
        const height = node.getBoundingClientRect().height
        const fontSize = Number.parseFloat(style.fontSize)
        const name = node.getAttribute('aria-label') ?? node.id
        return { name, height, fontSize }
      })
      .filter((field) => field.fontSize < 16 || field.height < 44)
      .map((field) => `${field.name}: ${field.fontSize}px / ${field.height}px`),
  )
}

function stubSlot(id: number, name: string, qualityProfileId: number) {
  return {
    id,
    slotNumber: id,
    name,
    enabled: true,
    qualityProfileId,
    displayOrder: id,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
    rootFolders: {},
  }
}

async function stubJson(page: Page, pattern: string, body: unknown): Promise<void> {
  await page.route(pattern, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(body),
    })
  })
}

/**
 * Puts the version-slot screen in the one state where the dry run and both
 * resolve flows are reachable, without writing to the shared developer database.
 */
export async function stubSlotSetup(page: Page, qualityProfileId: number): Promise<void> {
  await stubJson(page, '**/api/v1/slots/settings', {
    enabled: false,
    dryRunCompleted: false,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  })
  await stubJson(page, '**/api/v1/slots', [
    stubSlot(1, 'Standard Slot', qualityProfileId),
    stubSlot(2, '4K Slot', qualityProfileId),
  ])
  await stubJson(page, '**/api/v1/slots/validate', {
    valid: false,
    errors: ['Profile conflict between Standard Slot and 4K Slot'],
    conflicts: [
      {
        slotAId: 1,
        slotAName: 'Standard Slot',
        slotBId: 2,
        slotBName: '4K Slot',
        issues: [{ attribute: 'HDR', message: 'Both slots accept the same HDR formats' }],
      },
    ],
  })
  await stubJson(page, '**/api/v1/slots/validate-naming', {
    canProceed: false,
    noEnabledSlots: false,
    qualityTierExclusive: false,
    requiredAttributes: ['HDR'],
    movieFormatValid: true,
    movieValidation: { valid: true, missingTokens: [] },
    episodeFormatValid: false,
    episodeValidation: {
      valid: false,
      missingTokens: [{ attribute: 'HDR', suggestedToken: '{MediaInfo HDR}' }],
    },
  })
}
