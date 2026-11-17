import type { Problem } from "./problem.ts";
import type { UpdateNotificationSettings403Code } from "./updateNotificationSettings403Code.ts";

export type UpdateNotificationSettings403 = Problem & {
  code?: UpdateNotificationSettings403Code;
};
