import type { Problem } from "./problem.ts";
import type { UpdateNotificationBell409Code } from "./updateNotificationBell409Code.ts";

export type UpdateNotificationBell409 = Problem & {
  code?: UpdateNotificationBell409Code;
};
