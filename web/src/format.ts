const units = ["bytes", "KB", "MB", "GB", "TB"] as const;

export function bytes(count: number): string {
  let value = count;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(unit > 0 && value < 10 ? 1 : 0)} ${units[unit]}`;
}

const clockFormat = new Intl.DateTimeFormat(undefined, {
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});

export const clock = (iso: string): string => clockFormat.format(new Date(iso));

const minute = 60;
const hour = 60 * minute;
const day = 24 * hour;

export function durationWords(seconds: number): string {
  if (seconds === day) {
    return "1 day";
  }
  if (seconds > 0 && seconds % day === 0) {
    return `${seconds / day} days`;
  }
  if (seconds > 0 && seconds % hour === 0) {
    return `${seconds / hour} h`;
  }
  if (seconds > 0 && seconds % minute === 0) {
    return `${seconds / minute} min`;
  }
  return `${seconds} s`;
}

export type Noun = { readonly one: string; readonly many: string };

export const nounOf = (count: number, noun: Noun): string => (count === 1 ? noun.one : noun.many);

export const countWords = (count: number, unit: Noun): string =>
  unit.many === "%" ? `${count}%` : `${count} ${nounOf(count, unit)}`;

export function lengthWords(milliseconds: number): string {
  const seconds = Math.floor(milliseconds / 1000);
  if (seconds < 1) {
    return "under 1 s";
  }
  if (seconds < minute) {
    return `${seconds} s`;
  }
  if (seconds < hour) {
    const rest = seconds % minute;
    return rest === 0 ? `${seconds / minute} min` : `${Math.floor(seconds / minute)} min ${rest} s`;
  }
  const minutes = Math.floor((seconds % hour) / minute);
  const hours = Math.floor(seconds / hour);
  return minutes === 0 ? `${hours} h` : `${hours} h ${minutes} min`;
}

/** Now, read on a clock in a time zone; undefined is the browser's. */
export type Clock = { readonly now: Date; readonly zone: string | undefined };

/**
 * The zone when the name is an IANA one Intl knows, else undefined for the
 * browser's. Intl reads some abbreviations as an unrelated zone (BST as
 * Asia/Dhaka), so only an Area/Location name or UTC counts.
 */
export function knownZone(name: string): string | undefined {
  if (name !== "UTC" && !name.includes("/")) {
    return undefined;
  }
  try {
    return new Intl.DateTimeFormat("en", { timeZone: name }).resolvedOptions().timeZone;
  } catch {
    return undefined;
  }
}

type Formats = {
  readonly time: Intl.DateTimeFormat;
  readonly weekday: Intl.DateTimeFormat;
  readonly date: Intl.DateTimeFormat;
  readonly wall: Intl.DateTimeFormat;
};

const formatsByZone = new Map<string | undefined, Formats>();

function formatsOf(timeZone: string | undefined): Formats {
  const known = formatsByZone.get(timeZone);
  if (known !== undefined) {
    return known;
  }
  const zoned = timeZone === undefined ? {} : { timeZone };
  const formats: Formats = {
    time: new Intl.DateTimeFormat(undefined, {
      ...zoned,
      hour: "2-digit",
      minute: "2-digit",
      hourCycle: "h23",
    }),
    weekday: new Intl.DateTimeFormat(undefined, { ...zoned, weekday: "short" }),
    date: new Intl.DateTimeFormat(undefined, { ...zoned, month: "short", day: "numeric" }),
    wall: new Intl.DateTimeFormat("en-US", {
      ...zoned,
      year: "numeric",
      month: "numeric",
      day: "numeric",
      hour: "numeric",
      hourCycle: "h23",
    }),
  };
  formatsByZone.set(timeZone, formats);
  return formats;
}

function wallOf(date: Date, zone: string | undefined): { day: number; hour: number } {
  const parts = new Map(
    formatsOf(zone)
      .wall.formatToParts(date)
      .map((part) => [part.type, Number(part.value)]),
  );
  const at = (type: Intl.DateTimeFormatPartTypes) => parts.get(type) ?? 0;
  return {
    day: Date.UTC(at("year"), at("month") - 1, at("day")) / (day * 1000),
    hour: at("hour"),
  };
}

const daysFrom = (from: Date, to: Date, zone: string | undefined) =>
  wallOf(to, zone).day - wallOf(from, zone).day;

function eveningOf(date: Date, { now, zone }: Clock): number | undefined {
  const hours = wallOf(date, zone).hour;
  if (hours >= 18) {
    return daysFrom(now, date, zone);
  }
  return hours < 6 ? daysFrom(now, date, zone) - 1 : undefined;
}

export function atWords(iso: string, reading: Clock): string {
  const { now, zone } = reading;
  const at = new Date(iso);
  if (at <= now) {
    return "Due now";
  }
  const formats = formatsOf(zone);
  const time = formats.time.format(at);
  const days = daysFrom(now, at, zone);
  if (eveningOf(at, reading) === (eveningOf(now, reading) ?? 0)) {
    return `Tonight at ${time}`;
  }
  if (days === 0) {
    return `Today at ${time}`;
  }
  if (days === 1) {
    return `Tomorrow at ${time}`;
  }
  return `${days < 7 ? formats.weekday.format(at) : formats.date.format(at)} at ${time}`;
}

export function inWords(iso: string, now: Date): string {
  const minutes = Math.ceil((Date.parse(iso) - now.getTime()) / (minute * 1000));
  if (minutes <= 0) {
    return "Due now";
  }
  if (minutes < 60) {
    return `In ${minutes} min`;
  }
  const hours = Math.round(minutes / 60);
  return hours < 48 ? `In ${hours} h` : `In ${Math.round(hours / 24)} days`;
}

export function sinceWords(iso: string, { now, zone }: Clock): string {
  const at = new Date(iso);
  const formats = formatsOf(zone);
  const time = formats.time.format(at);
  return daysFrom(now, at, zone) === 0 ? time : `${formats.date.format(at)}, ${time}`;
}
