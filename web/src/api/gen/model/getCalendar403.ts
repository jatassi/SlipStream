import type { GetCalendar403Code } from "./getCalendar403Code.ts";
import type { Problem } from "./problem.ts";

export type GetCalendar403 = Problem & {
  code?: GetCalendar403Code;
};
