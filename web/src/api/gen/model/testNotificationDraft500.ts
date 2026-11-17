import type { Problem } from "./problem.ts";
import type { TestNotificationDraft500Code } from "./testNotificationDraft500Code.ts";

export type TestNotificationDraft500 = Problem & {
  code?: TestNotificationDraft500Code;
};
