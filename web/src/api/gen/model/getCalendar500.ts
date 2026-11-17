import type { GetCalendar500Code } from "./getCalendar500Code.ts";
import type { Problem } from "./problem.ts";

export type GetCalendar500 = Problem & {
  code?: GetCalendar500Code;
};
