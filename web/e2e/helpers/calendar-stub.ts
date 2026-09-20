import type { Page } from '@playwright/test'

// The developer-mode library carries no release or air dates, so the calendar
// API answers with an empty list. These events stand in for it so the grouping
// and the month grid can be exercised against known dates.
export type StubEvent = {
  id: number
  title: string
  mediaType: string
  moduleType: string
  eventType: string
  date: string
  status: string
  monitored: boolean
  extra?: Record<string, unknown>
}

function isoDay(offset: number): string {
  const day = new Date()
  day.setDate(day.getDate() + offset)
  return [
    day.getFullYear().toString().padStart(4, '0'),
    (day.getMonth() + 1).toString().padStart(2, '0'),
    day.getDate().toString().padStart(2, '0'),
  ].join('-')
}

export const calendarEvents: StubEvent[] = [
  {
    id: 9001,
    title: 'Stub Feature',
    mediaType: 'movie',
    moduleType: 'movie',
    eventType: 'digital',
    date: isoDay(0),
    status: 'missing',
    monitored: true,
    extra: { year: 2026 },
  },
  {
    id: 9002,
    title: 'Stub Pilot',
    mediaType: 'episode',
    moduleType: 'tv',
    eventType: 'airDate',
    date: isoDay(0),
    status: 'available',
    monitored: true,
    extra: { seriesId: 1, seriesTitle: 'Stub Series', seasonNumber: 1, episodeNumber: 1 },
  },
  {
    id: 9003,
    title: 'Stub Sequel',
    mediaType: 'movie',
    moduleType: 'movie',
    eventType: 'physical',
    date: isoDay(1),
    status: 'missing',
    monitored: true,
    extra: { year: 2026 },
  },
]

export const todayEventTitles = ['Stub Feature', 'Stub Series']
export const tomorrowEventTitle = 'Stub Sequel'

export async function stubCalendar(page: Page): Promise<void> {
  await page.route('**/api/v1/calendar*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(calendarEvents),
    })
  })
}
