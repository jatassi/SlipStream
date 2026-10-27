import type { WorksContentType, WorksFieldValue } from "../../api/gen/model";
import { fieldText } from "../fields";

/** The Work's values of the fields its type places in the facts line, in declared order. */
export function factTexts(type: WorksContentType, fields: readonly WorksFieldValue[]): string[] {
  return type.fields
    .filter((spec) => spec.tier === "work" && spec.in.includes("facts"))
    .flatMap((spec) => {
      const value = fields.find((field) => field.key === spec.key);
      return value === undefined ? [] : [fieldText(value)];
    });
}

/** The hero's facts line, the type's noun after the first fact: "1995 · Movie · 2h 50m · R". */
export function factsLine(type: WorksContentType, fields: readonly WorksFieldValue[]): string {
  const [first, ...rest] = factTexts(type, fields);
  return (first === undefined ? [type.work.one] : [first, type.work.one, ...rest]).join(" · ");
}
