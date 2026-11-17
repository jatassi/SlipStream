import type { GetCalendar409Code } from "./getCalendar409Code.ts";
import type { Problem } from "./problem.ts";

export type GetCalendar409 = Problem & {
  code?: GetCalendar409Code;
};
