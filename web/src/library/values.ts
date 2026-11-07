import type { WorksContentType } from "../api/gen/model";
import type { LibraryWork, ListValue, PartialDate } from "./rows";

const full = new Intl.DateTimeFormat(undefined, {
  day: "numeric",
  month: "short",
  year: "numeric",
});
const monthly = new Intl.DateTimeFormat(undefined, { month: "short", year: "numeric" });

/** A date as precise as it is known: "9 Apr 2021", "Apr 2021" or "2021". */
export function partialDateText(date: PartialDate): string {
  const [year = 0, month, day] = date.split("-").map(Number);
  if (month === undefined) {
    return String(year);
  }
  return day === undefined
    ? monthly.format(new Date(year, month - 1))
    : full.format(new Date(year, month - 1, day));
}

export const dateText = (date: Date): string => full.format(date);

const minuteMs = 60_000;

export function valueText(value: ListValue): string {
  switch (value.kind) {
    case "text":
      return value.text;
    case "number":
      return String(value.number);
    case "date":
      return partialDateText(value.date);
    case "texts":
      return value.texts.join(", ");
    case "duration": {
      const minutes = Math.round(value.ms / minuteMs);
      return minutes < 60 ? `${minutes}m` : `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
    }
    default:
      return value satisfies never;
  }
}

const firstFacts = new WeakMap<WorksContentType, readonly string[]>();

/** The first of its type's facts the Work has, such as its year, for the line under a poster. */
export function leadFact(work: LibraryWork): string | undefined {
  let keys = firstFacts.get(work.type);
  if (keys === undefined) {
    keys = work.type.fields.filter((field) => field.in.includes("facts")).map(({ key }) => key);
    firstFacts.set(work.type, keys);
  }
  for (const key of keys) {
    const value = work.values.get(key);
    if (value !== undefined) {
      return valueText(value);
    }
  }
  return undefined;
}
