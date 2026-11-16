import type { Problem } from "./problem.ts";
import type { UpdateNotificationConnection401Code } from "./updateNotificationConnection401Code.ts";

export type UpdateNotificationConnection401 = Problem & {
  code?: UpdateNotificationConnection401Code;
};
