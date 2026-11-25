import type { MediaserversAccessProblemCode } from "./mediaserversAccessProblemCode.ts";

export interface MediaserversAccessProblem {
  code: MediaserversAccessProblemCode;
  /** The reply of plex.tv or the server, when there was one */
  detail?: string;
}
