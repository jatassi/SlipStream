import type { Problem } from "./problem.ts";
import type { RefreshWork404Code } from "./refreshWork404Code.ts";

export type RefreshWork404 = Problem & {
  code?: RefreshWork404Code;
};
