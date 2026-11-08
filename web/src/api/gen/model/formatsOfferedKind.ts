import type { FormatsKindValue } from "./formatsKindValue.ts";

export interface FormatsOfferedKind {
  /** The fields its conditions take, in upstream's order */
  fields: string[];
  kind: string;
  /** Its value is a .NET regular expression, matched ignoring case */
  pattern: boolean;
  /** What its value field takes, in upstream's order; empty for a kind whose value is a pattern or a number */
  values: FormatsKindValue[];
  words: string;
}
