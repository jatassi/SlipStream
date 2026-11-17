import type { Problem } from "./problem.ts";
import type { TestNotificationConnection409Code } from "./testNotificationConnection409Code.ts";

export type TestNotificationConnection409 = Problem & {
  code?: TestNotificationConnection409Code;
};
