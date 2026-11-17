import type { GetNotificationBell401Code } from "./getNotificationBell401Code.ts";
import type { Problem } from "./problem.ts";

export type GetNotificationBell401 = Problem & {
  code?: GetNotificationBell401Code;
};
