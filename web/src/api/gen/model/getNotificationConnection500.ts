import type { GetNotificationConnection500Code } from "./getNotificationConnection500Code.ts";
import type { Problem } from "./problem.ts";

export type GetNotificationConnection500 = Problem & {
  code?: GetNotificationConnection500Code;
};
