import type { Problem } from "./problem.ts";
import type { UpdateNotificationConnection500Code } from "./updateNotificationConnection500Code.ts";

export type UpdateNotificationConnection500 = Problem & {
  code?: UpdateNotificationConnection500Code;
};
