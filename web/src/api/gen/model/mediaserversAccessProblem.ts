import type { MediaserversAccessProblemCode } from "./mediaserversAccessProblemCode.ts";

export interface MediaserversAccessProblem {
  code: MediaserversAccessProblemCode;
  /** What went wrong, as a sentence for an Admin */
  detail?: string;
}
