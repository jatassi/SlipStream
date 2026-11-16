import type { GetNotificationConnection422Code } from "./getNotificationConnection422Code.ts";
import type { Problem } from "./problem.ts";

export type GetNotificationConnection422 = Problem & {
  code?: GetNotificationConnection422Code;
};
