import type { Problem } from "./problem.ts";
import type { TestNotificationConnection422Code } from "./testNotificationConnection422Code.ts";

export type TestNotificationConnection422 = Problem & {
  code?: TestNotificationConnection422Code;
  field?: string;
};
