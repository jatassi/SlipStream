import type { GetNotificationConnection401Code } from "./getNotificationConnection401Code.ts";
import type { Problem } from "./problem.ts";

export type GetNotificationConnection401 = Problem & {
  code?: GetNotificationConnection401Code;
};
