import type { Problem } from "./problem.ts";
import type { UpdateNotificationSettings422Code } from "./updateNotificationSettings422Code.ts";

export type UpdateNotificationSettings422 = Problem & {
  code?: UpdateNotificationSettings422Code;
};
