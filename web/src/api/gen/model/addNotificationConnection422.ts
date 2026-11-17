import type { AddNotificationConnection422Code } from "./addNotificationConnection422Code.ts";
import type { Problem } from "./problem.ts";

export type AddNotificationConnection422 = Problem & {
  code?: AddNotificationConnection422Code;
  field?: string;
};
