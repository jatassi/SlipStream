import type { GetCalendar422Code } from "./getCalendar422Code.ts";
import type { Problem } from "./problem.ts";

export type GetCalendar422 = Problem & {
  code?: GetCalendar422Code;
};
