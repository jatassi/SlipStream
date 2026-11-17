import type { GetNotificationBell409Code } from "./getNotificationBell409Code.ts";
import type { Problem } from "./problem.ts";

export type GetNotificationBell409 = Problem & {
  code?: GetNotificationBell409Code;
};
