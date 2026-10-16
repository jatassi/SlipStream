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

export const countWords = (count: number, unit: string): string =>
  unit === "%" ? `${count}%` : `${count} ${unit}`;

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

const timeFormat = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
});
const weekdayFormat = new Intl.DateTimeFormat(undefined, { weekday: "short" });
const dateFormat = new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" });

const midnight = (date: Date) => new Date(date.getFullYear(), date.getMonth(), date.getDate());

// Rounding absorbs the hour a daylight saving change adds or removes.
const daysFrom = (from: Date, to: Date) =>
  Math.round((midnight(to).getTime() - midnight(from).getTime()) / (day * 1000));

// The evening a time belongs to, counted in days from now's date: a night
// runs from 18:00 to 06:00 the next morning.
function eveningOf(date: Date, now: Date): number | undefined {
  const hours = date.getHours();
  if (hours >= 18) {
    return daysFrom(now, date);
  }
  return hours < 6 ? daysFrom(now, date) - 1 : undefined;
}

export function atWords(iso: string, now: Date): string {
  const at = new Date(iso);
  if (at <= now) {
    return "Due now";
  }
  const time = timeFormat.format(at);
  const days = daysFrom(now, at);
  if (eveningOf(at, now) === (eveningOf(now, now) ?? 0)) {
    return `Tonight at ${time}`;
  }
  if (days === 0) {
    return `Today at ${time}`;
  }
  if (days === 1) {
    return `Tomorrow at ${time}`;
  }
  return `${days < 7 ? weekdayFormat.format(at) : dateFormat.format(at)} at ${time}`;
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

export function sinceWords(iso: string, now: Date): string {
  const at = new Date(iso);
  const time = timeFormat.format(at);
  return daysFrom(now, at) === 0 ? time : `${dateFormat.format(at)}, ${time}`;
}
