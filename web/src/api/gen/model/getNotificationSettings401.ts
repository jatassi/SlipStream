import type { GetNotificationSettings401Code } from "./getNotificationSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetNotificationSettings401 = Problem & {
  code?: GetNotificationSettings401Code;
};
