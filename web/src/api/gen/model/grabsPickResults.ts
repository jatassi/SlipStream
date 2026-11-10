import type { GrabsPickResult } from "./grabsPickResult.ts";

export interface GrabsPickResults {
  /** One for each pick, in the order sent */
  results: GrabsPickResult[];
}
