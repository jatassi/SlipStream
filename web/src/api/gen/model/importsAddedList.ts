import type { ImportsOutcome } from "./importsOutcome.ts";

export interface ImportsAddedList {
  /** One per pending import, in request order */
  outcomes: ImportsOutcome[];
}
