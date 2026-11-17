import type { WorksFieldValue, WorksRating } from "../../api/gen/model";
import { dayWords } from "../../format/days";
import { fieldText } from "../fields";

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
