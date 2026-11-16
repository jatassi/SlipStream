import type { Problem } from "./problem.ts";
import type { TestNotificationDraft409Code } from "./testNotificationDraft409Code.ts";

export type TestNotificationDraft409 = Problem & {
  code?: TestNotificationDraft409Code;
};
