const dayMs = 86_400_000;

const keyFormats = new Map<string | undefined, Intl.DateTimeFormat>();
const wordFormats = new Map<string | undefined, Intl.DateTimeFormat>();

function formatIn(
  cache: Map<string | undefined, Intl.DateTimeFormat>,
  zone: string | undefined,
  locale: string | undefined,
  options: Intl.DateTimeFormatOptions,
): Intl.DateTimeFormat {
  const known = cache.get(zone);
  if (known !== undefined) {
    return known;
  }
  const format = new Intl.DateTimeFormat(locale, {
    ...options,
    ...(zone !== undefined && { timeZone: zone }),
  });
  cache.set(zone, format);
  return format;
}

/** "2026-11-17": the day an instant falls on in zone, the browser's when undefined. */
export const localDayKey = (at: string | Date, zone?: string): string =>
  formatIn(keyFormats, zone, "en-CA", { year: "numeric", month: "2-digit", day: "2-digit" }).format(
    new Date(at),
  );

const dayNumber = (key: string): number => Date.parse(`${key}T00:00:00Z`) / dayMs;

/** "Today", "Yesterday", "Tuesday 10 November", with the year when it isn't this year. */
export function dayHeading(at: string, now: Date, zone?: string): string {
  const key = localDayKey(at, zone);
  const today = localDayKey(now, zone);
  const days = dayNumber(today) - dayNumber(key);
  if (days === 0) {
    return "Today";
  }
  if (days === 1) {
    return "Yesterday";
  }
  const parts = new Map(
    formatIn(wordFormats, zone, undefined, {
      weekday: "long",
      day: "numeric",
      month: "long",
      year: "numeric",
    })
      .formatToParts(new Date(at))
      .map((part) => [part.type, part.value]),
  );
  const words = `${parts.get("weekday")} ${parts.get("day")} ${parts.get("month")}`;
  return key.slice(0, 4) === today.slice(0, 4) ? words : `${words} ${parts.get("year")}`;
}

/** A date chip: today, the last 7 days or the last 30, each counting today. */
export type SinceChip = "today" | "7d" | "30d";

const chipDays: Record<SinceChip, number> = { today: 0, "7d": 6, "30d": 29 };

/** The browser's midnight that starts chip's window, as an instant. */
export function sinceOf(chip: SinceChip, now: Date): string {
  const midnight = new Date(now);
  midnight.setHours(0, 0, 0, 0);
  midnight.setDate(midnight.getDate() - chipDays[chip]);
  return midnight.toISOString();
}
