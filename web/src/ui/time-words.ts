import { type Clock, ownZone } from "./clock";

type Formats = {
  readonly time: Intl.DateTimeFormat;
  readonly stamp: Intl.DateTimeFormat;
  readonly weekday: Intl.DateTimeFormat;
  readonly date: Intl.DateTimeFormat;
  readonly dated: Intl.DateTimeFormat;
  readonly heading: Intl.DateTimeFormat;
  readonly moment: Intl.DateTimeFormat;
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
    time: new Intl.DateTimeFormat(undefined, { ...zoned, hour: "numeric", minute: "2-digit" }),
    // A log reads down a column, so its stamps keep one width on every locale.
    stamp: new Intl.DateTimeFormat("en-GB", {
      ...zoned,
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      fractionalSecondDigits: 3,
      hourCycle: "h23",
    }),
    weekday: new Intl.DateTimeFormat(undefined, { ...zoned, weekday: "short" }),
    date: new Intl.DateTimeFormat(undefined, { ...zoned, month: "short", day: "numeric" }),
    dated: new Intl.DateTimeFormat(undefined, {
      ...zoned,
      month: "short",
      day: "numeric",
      year: "numeric",
    }),
    heading: new Intl.DateTimeFormat(undefined, {
      ...zoned,
      weekday: "long",
      day: "numeric",
      month: "long",
      year: "numeric",
    }),
    moment: new Intl.DateTimeFormat(undefined, { ...zoned, dateStyle: "full", timeStyle: "long" }),
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

type Wall = { readonly day: number; readonly hour: number; readonly year: number };

const dayMs = 86_400_000;

function wallOf(date: Date, formats: Formats): Wall {
  const parts = new Map(formats.wall.formatToParts(date).map((part) => [part.type, part.value]));
  const at = (type: Intl.DateTimeFormatPartTypes) => Number(parts.get(type));
  return {
    day: Date.UTC(at("year"), at("month") - 1, at("day")) / dayMs,
    hour: at("hour"),
    year: at("year"),
  };
}

// The evening runs from 18:00 to 06:00 the next day, named by the day it starts.
const eveningOf = ({ day, hour }: Wall): number | undefined => {
  if (hour >= 18) {
    return day;
  }
  return hour < 6 ? day - 1 : undefined;
};

/** Where an instant falls against now on a clock, which every form reads. */
export type Placed = {
  /** Milliseconds from now to the instant, below zero when it has passed. */
  readonly ahead: number;
  /** Calendar days from now's day to the instant's, below zero before today. */
  readonly days: number;
  /** A coming instant in the evening now is in, or the one ahead of a day. */
  readonly tonight: boolean;
  readonly time: string;
  readonly stamp: string;
  readonly weekday: string;
  /** "Nov 19", with the year when it isn't now's. */
  readonly date: string;
  /** "Tuesday 10 November", with the year when it isn't now's. */
  readonly heading: string;
};

export function placeOf(at: string, { now, zone }: Clock): Placed {
  const instant = new Date(at);
  const formats = formatsOf(zone);
  const there = wallOf(instant, formats);
  const here = wallOf(now, formats);
  const ahead = instant.getTime() - now.getTime();
  const thisYear = there.year === here.year;
  const heading = new Map(
    formats.heading.formatToParts(instant).map((part) => [part.type, part.value]),
  );
  const named = `${heading.get("weekday")} ${heading.get("day")} ${heading.get("month")}`;
  return {
    ahead,
    days: there.day - here.day,
    tonight: ahead > 0 && eveningOf(there) === (eveningOf(here) ?? here.day),
    time: formats.time.format(instant),
    stamp: formats.stamp.format(instant),
    weekday: formats.weekday.format(instant),
    date: (thisYear ? formats.date : formats.dated).format(instant),
    heading: thisYear ? named : `${named} ${heading.get("year")}`,
  };
}

/** The day an instant falls on, by name when it has one: "tonight", "tomorrow", "Mon", "Nov 19". */
function dayOf(placed: Placed): { readonly words: string; readonly named: boolean } {
  if (placed.tonight) {
    return { words: "tonight", named: true };
  }
  if (Math.abs(placed.days) === 1) {
    return { words: placed.days === 1 ? "tomorrow" : "yesterday", named: true };
  }
  return { words: Math.abs(placed.days) < 7 ? placed.weekday : placed.date, named: false };
}

const onDay = (placed: Placed): string => {
  const { words, named } = dayOf(placed);
  return named ? words : `on ${words}`;
};

const capital = (words: string) => `${words.charAt(0).toUpperCase()}${words.slice(1)}`;

const ago = new Intl.RelativeTimeFormat("en", { numeric: "always" });
const minuteMs = 60_000;

function ageWords({ ahead }: Placed): string {
  const minutes = Math.floor(-ahead / minuteMs);
  if (minutes < 1) {
    return "just now";
  }
  if (minutes < 60) {
    return ago.format(-minutes, "minute");
  }
  const hours = Math.floor(minutes / 60);
  return hours < 24 ? ago.format(-hours, "hour") : ago.format(-Math.floor(hours / 24), "day");
}

/** How long until a coming instant: "12 min", "3 h", "2 days", or undefined once it has come. */
function leftWords({ ahead }: Placed): string | undefined {
  const minutes = Math.ceil(ahead / minuteMs);
  if (minutes <= 0) {
    return undefined;
  }
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.round(minutes / 60);
  return hours < 48 ? `${hours} h` : `${Math.round(hours / 24)} days`;
}

const lead = (word: string, placed: Placed): string =>
  placed.days === 0 ? `${word} ${placed.time}` : `${word} ${dayOf(placed).words} at ${placed.time}`;

const atWords = (placed: Placed): string =>
  placed.days === 0 ? `at ${placed.time}` : `${onDay(placed)} at ${placed.time}`;

/**
 * Every way a SlipStream screen says an instant. The first six stand on their
 * own and read only through RelativeTime or PlainTime, so each carries its
 * whole moment. The rest sit inside a sentence or a heading, through whenWords.
 */
export const timeForms = {
  clock: (placed) => (placed.days === 0 ? placed.time : `${placed.date}, ${placed.time}`),
  time: (placed) => placed.time,
  stamp: (placed) => placed.stamp,
  due: (placed) => {
    if (placed.ahead <= 0) {
      return "Due now";
    }
    const day = placed.days === 0 && !placed.tonight ? "today" : dayOf(placed).words;
    return `${capital(day)} at ${placed.time}`;
  },
  countdown: (placed) => {
    const left = leftWords(placed);
    return left === undefined ? "Due now" : `In ${left}`;
  },
  age: ageWords,
  // An age drawn where no element can carry its moment, as on a Diagram tile.
  ago: ageWords,
  at: atWords,
  next: (placed) => (placed.ahead <= 0 ? "now" : atWords(placed)),
  in: (placed) => {
    const left = leftWords(placed);
    return left === undefined ? "now" : `in ${left}`;
  },
  since: (placed) => lead("since", placed),
  sinceDay: (placed) => `since ${placed.days === 0 ? "today" : dayOf(placed).words}`,
  until: (placed) => lead("until", placed),
  from: (placed) => lead("from", placed),
  on: (placed) => (placed.days === 0 ? "today" : onDay(placed)),
  heading: (placed) => {
    if (placed.days === 0) {
      return "Today";
    }
    return placed.days === -1 ? "Yesterday" : placed.heading;
  },
} as const satisfies Record<string, (placed: Placed) => string>;

export type TimeForm = keyof typeof timeForms;

/** A form that stands on its own as a value, read through RelativeTime or PlainTime. */
export type ValueForm = "clock" | "time" | "stamp" | "due" | "countdown" | "age";

/** A form inside a sentence, which carries its own word before the time, or a day's heading. */
export type WordsForm = Exclude<TimeForm, ValueForm>;

/**
 * An instant inside a sentence, in lower case with its preposition: "at 2:02 PM"
 * today, "yesterday at 2:02 PM", "on Mon at 2:02 PM", "on Nov 19 at 2:02 PM". It has
 * no hover, because the sentence already names the day.
 */
export const whenWords = (at: string, form: WordsForm, clock: Clock): string =>
  timeForms[form](placeOf(at, clock));

/** Whether an instant falls on the clock's today. */
export const isToday = (at: string, clock: Clock): boolean => placeOf(at, clock).days === 0;

/** The whole moment on the clock, then on the person's own when the clock is another zone's. */
export function momentWords(at: string, { zone }: Clock): readonly string[] {
  const instant = new Date(at);
  const there = formatsOf(zone).moment.format(instant);
  return zone === undefined || zone === ownZone()
    ? [there]
    : [there, formatsOf(undefined).moment.format(instant)];
}
