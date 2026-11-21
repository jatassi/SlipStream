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

const grouped = new Intl.NumberFormat("en");
const minute = 60;
const hour = 60 * minute;
const day = 24 * hour;

export function durationWords(seconds: number): string {
  if (seconds === day) {
    return "1 day";
  }
  if (seconds > 0 && seconds % day === 0) {
    return `${grouped.format(seconds / day)} days`;
  }
  if (seconds > 0 && seconds % hour === 0) {
    return `${grouped.format(seconds / hour)} h`;
  }
  if (seconds > 0 && seconds % minute === 0) {
    return `${grouped.format(seconds / minute)} min`;
  }
  return `${grouped.format(seconds)} s`;
}

type SpanUnit = { readonly seconds: number; readonly words: (count: number) => string };

const spanDays: SpanUnit = {
  seconds: day,
  words: (count) => (count === 1 ? "1 day" : `${count} days`),
};
const spanHours: SpanUnit = { seconds: hour, words: (count) => `${count} h` };
const spanMinutes: SpanUnit = { seconds: minute, words: (count) => `${count} min` };
const spanSeconds: SpanUnit = { seconds: 1, words: (count) => `${count} s` };

const spanPairs = [
  [spanDays, spanHours],
  [spanHours, spanMinutes],
  [spanMinutes, spanSeconds],
] as const;

/**
 * A length in at most two units, rounded up to the smaller one, so a time
 * still needed is never understated: "3 days 4 h", "4 h 12 min", "3 min 27 s".
 */
export function spanWords(seconds: number): string {
  for (const [big, small] of spanPairs) {
    const count = Math.ceil(seconds / small.seconds);
    const per = big.seconds / small.seconds;
    if (count >= per) {
      const rest = count % per;
      const whole = big.words((count - rest) / per);
      return rest === 0 ? whole : `${whole} ${small.words(rest)}`;
    }
  }
  return spanSeconds.words(Math.ceil(seconds));
}

export type Noun = { readonly one: string; readonly many: string };

export const nounOf = (count: number, noun: Noun): string => (count === 1 ? noun.one : noun.many);

export const countWords = (count: number, unit: Noun): string =>
  unit.many.startsWith("%") ? `${count}${unit.many}` : `${count} ${nounOf(count, unit)}`;

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
