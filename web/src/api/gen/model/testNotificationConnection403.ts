import type { Problem } from "./problem.ts";
import type { TestNotificationConnection403Code } from "./testNotificationConnection403Code.ts";

export type TestNotificationConnection403 = Problem & {
  code?: TestNotificationConnection403Code;
};
