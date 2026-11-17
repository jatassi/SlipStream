import type { GetCalendar401Code } from "./getCalendar401Code.ts";
import type { Problem } from "./problem.ts";

export type GetCalendar401 = Problem & {
  code?: GetCalendar401Code;
};
