/** A date without a time or zone, such as "2026-11-17", which no browser offset shifts. */
export type IsoDay = string & { readonly __isoDay: unique symbol };

export type Month = { readonly year: number; readonly month: number };

const dayMs = 86_400_000;
const isoPattern = /^\d{4}-\d{2}-\d{2}$/u;

export function isoDay(text: string): IsoDay {
  if (!isoPattern.test(text)) {
    throw new Error(`${text} is not a day`);
  }
  // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the pattern above is what makes an IsoDay
  return text as IsoDay;
}

const msOf = (day: IsoDay) => Date.parse(`${day}T00:00:00Z`);
const dayAt = (ms: number) => isoDay(new Date(ms).toISOString().slice(0, 10));

export const addDays = (day: IsoDay, days: number): IsoDay => dayAt(msOf(day) + days * dayMs);

export const daysBetween = (from: IsoDay, to: IsoDay): number =>
  Math.round((msOf(to) - msOf(from)) / dayMs);

export function monthOf(day: IsoDay): Month {
  const at = new Date(msOf(day));
  return { year: at.getUTCFullYear(), month: at.getUTCMonth() + 1 };
}

export const addMonths = ({ year, month }: Month, months: number): Month => {
  const at = new Date(Date.UTC(year, month - 1 + months, 1));
  return { year: at.getUTCFullYear(), month: at.getUTCMonth() + 1 };
};

/** "2026-11", as a month's address names it. */
export const monthKey = ({ year, month }: Month): string =>
  `${String(year)}-${String(month).padStart(2, "0")}`;

/** The day's weekday, 0 for Sunday to 6 for Saturday. */
const weekdayOf = (day: IsoDay): number => new Date(msOf(day)).getUTCDay();

/** The month grid's six weeks: the first day shown, and the day after the last, 42 days on. */
export function monthSpan(
  { year, month }: Month,
  firstWeekday: number,
): { from: IsoDay; to: IsoDay } {
  const first = dayAt(Date.UTC(year, month - 1, 1));
  const from = addDays(first, -((weekdayOf(first) - firstWeekday + 7) % 7));
  return { from, to: addDays(from, 42) };
}

/** The browser's own day, the Calendar's guess until the Controller names its day. */
export function browserToday(now = new Date()): IsoDay {
  return dayAt(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate()));
}

const formatter = (options: Intl.DateTimeFormatOptions) =>
  new Intl.DateTimeFormat(undefined, { ...options, timeZone: "UTC" });

const fullDate = formatter({ day: "numeric", month: "long", year: "numeric" });
const monthYear = formatter({ month: "long", year: "numeric" });
const dayMonth = formatter({ day: "numeric", month: "long" });
const longDay = formatter({ weekday: "long", day: "numeric", month: "long" });
const shortWeekday = formatter({ weekday: "short" });
const longWeekday = formatter({ weekday: "long" });

/** A stored date at the source's precision, a day, a month or a year, in the browser's words. */
export function dayWords(date: string): string {
  const [year, monthPart, dayPart] = date.split("-").map(Number);
  if (year === undefined || monthPart === undefined) {
    return date;
  }
  const at = new Date(Date.UTC(year, monthPart - 1, dayPart ?? 1));
  return dayPart === undefined ? monthYear.format(at) : fullDate.format(at);
}

export const monthWords = ({ year, month }: Month): string =>
  monthYear.format(new Date(Date.UTC(year, month - 1, 1)));

/** "Tuesday 17 November". */
export const calendarDayWords = (day: IsoDay): string => longDay.format(new Date(msOf(day)));

/** "14 November and 26 December", with each year when the two days' years differ. */
export function betweenWords(first: IsoDay, last: IsoDay): string {
  const words = monthOf(first).year === monthOf(last).year ? dayMonth : fullDate;
  return `${words.format(new Date(msOf(first)))} and ${words.format(new Date(msOf(last)))}`;
}

/** The weekday's short and long names, "Tue" and "Tuesday". */
export function weekdayNames(day: IsoDay): { short: string; long: string } {
  const at = new Date(msOf(day));
  return { short: shortWeekday.format(at), long: longWeekday.format(at) };
}

type WeekInfo = { readonly firstDay: number };
type WithWeekInfo = Intl.Locale & {
  readonly getWeekInfo?: () => WeekInfo;
  readonly weekInfo?: WeekInfo;
};

const monday = 1;

/** The locale's first day of the week, 0 for Sunday to 6, and Monday when the browser can't say. */
export function weekStart(locale = navigator.language): number {
  const tagged: WithWeekInfo = new Intl.Locale(locale);
  const info = tagged.getWeekInfo?.() ?? tagged.weekInfo;
  return info === undefined ? monday : info.firstDay % 7;
}
