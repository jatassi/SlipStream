import type { Problem } from "./problem.ts";
import type { TestNotificationDraft401Code } from "./testNotificationDraft401Code.ts";

export type TestNotificationDraft401 = Problem & {
  code?: TestNotificationDraft401Code;
};
