import type { Problem } from "./problem.ts";
import type { UpdateNotificationBell500Code } from "./updateNotificationBell500Code.ts";

export type UpdateNotificationBell500 = Problem & {
  code?: UpdateNotificationBell500Code;
};
