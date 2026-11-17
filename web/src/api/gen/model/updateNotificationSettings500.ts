import type { Problem } from "./problem.ts";
import type { UpdateNotificationSettings500Code } from "./updateNotificationSettings500Code.ts";

export type UpdateNotificationSettings500 = Problem & {
  code?: UpdateNotificationSettings500Code;
};
