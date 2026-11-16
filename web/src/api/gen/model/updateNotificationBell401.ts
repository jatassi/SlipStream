import type { Problem } from "./problem.ts";
import type { UpdateNotificationBell401Code } from "./updateNotificationBell401Code.ts";

export type UpdateNotificationBell401 = Problem & {
  code?: UpdateNotificationBell401Code;
};
