import type { GetNotificationSettings500Code } from "./getNotificationSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetNotificationSettings500 = Problem & {
  code?: GetNotificationSettings500Code;
};
