import type { Problem } from "./problem.ts";
import type { TestNotificationConnection401Code } from "./testNotificationConnection401Code.ts";

export type TestNotificationConnection401 = Problem & {
  code?: TestNotificationConnection401Code;
};
