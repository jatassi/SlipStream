import type { Problem } from "./problem.ts";
import type { TestNotificationConnection404Code } from "./testNotificationConnection404Code.ts";

export type TestNotificationConnection404 = Problem & {
  code?: TestNotificationConnection404Code;
};
