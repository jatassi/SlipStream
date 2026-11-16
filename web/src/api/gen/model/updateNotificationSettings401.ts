import type { Problem } from "./problem.ts";
import type { UpdateNotificationSettings401Code } from "./updateNotificationSettings401Code.ts";

export type UpdateNotificationSettings401 = Problem & {
  code?: UpdateNotificationSettings401Code;
};
