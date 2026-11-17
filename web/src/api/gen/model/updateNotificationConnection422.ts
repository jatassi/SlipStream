import type { Problem } from "./problem.ts";
import type { UpdateNotificationConnection422Code } from "./updateNotificationConnection422Code.ts";

export type UpdateNotificationConnection422 = Problem & {
  code?: UpdateNotificationConnection422Code;
  field?: string;
};
