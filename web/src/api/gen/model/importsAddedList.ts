import type { ImportsOutcome } from "./importsOutcome.ts";

export interface ImportsAddedList {
  /** One per Import, in request order */
  outcomes: ImportsOutcome[];
}
