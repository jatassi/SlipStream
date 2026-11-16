import type { GetNotificationBell403Code } from "./getNotificationBell403Code.ts";
import type { Problem } from "./problem.ts";

export type GetNotificationBell403 = Problem & {
  code?: GetNotificationBell403Code;
};
