import type { Problem } from "./problem.ts";
import type { UpdateNotificationBell403Code } from "./updateNotificationBell403Code.ts";

export type UpdateNotificationBell403 = Problem & {
  code?: UpdateNotificationBell403Code;
};
