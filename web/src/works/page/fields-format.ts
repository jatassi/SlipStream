import type { WorksFieldValue, WorksRating } from "../../api/gen/model";
import { fieldText } from "../fields";

const fullDate = new Intl.DateTimeFormat(undefined, {
  day: "numeric",
  month: "long",
  year: "numeric",
  timeZone: "UTC",
});
const month = new Intl.DateTimeFormat(undefined, {
  month: "long",
  year: "numeric",
  timeZone: "UTC",
});

/** A stored date at the source's precision, a day, a month or a year, in the browser's words. */
export function dayWords(date: string): string {
  const [year, monthPart, dayPart] = date.split("-").map(Number);
  if (year === undefined || monthPart === undefined) {
    return date;
  }
  const at = new Date(Date.UTC(year, monthPart - 1, dayPart ?? 1));
  return dayPart === undefined ? month.format(at) : fullDate.format(at);
}

/** A field's value as About shows it, with dates in words. */
export const aboutText = (value: WorksFieldValue): string =>
  value.kind === "date" ? dayWords(value.date) : fieldText(value);

/** A rating's value on its own scale: "8.3", "94%" or "3.5/5". */
export function ratingValue({ value, scale }: WorksRating): string {
  if (scale === 100) {
    return `${Math.round(value)}%`;
  }
  const shown = Number.isInteger(value) ? String(value) : value.toFixed(1);
  return scale === 10 ? shown : `${shown}/${scale}`;
}
