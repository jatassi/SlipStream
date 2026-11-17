import type { Problem } from "./problem.ts";
import type { UpdateNotificationBell422Code } from "./updateNotificationBell422Code.ts";

export type UpdateNotificationBell422 = Problem & {
  code?: UpdateNotificationBell422Code;
};
