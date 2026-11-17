import type { Problem } from "./problem.ts";
import type { TestNotificationConnection500Code } from "./testNotificationConnection500Code.ts";

export type TestNotificationConnection500 = Problem & {
  code?: TestNotificationConnection500Code;
};
