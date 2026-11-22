import type { HomeArrival } from "./homeArrival.ts";
import type { HomeToday } from "./homeToday.ts";

export interface HomeRecent {
  /** Finished Imports, newest first, at most Rows on Home */
  rows: HomeArrival[];
  today: HomeToday;
}
