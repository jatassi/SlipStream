import type { GetNotificationBell500Code } from "./getNotificationBell500Code.ts";
import type { Problem } from "./problem.ts";

export type GetNotificationBell500 = Problem & {
  code?: GetNotificationBell500Code;
};
