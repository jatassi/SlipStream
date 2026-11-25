import type { Problem } from "./problem.ts";
import type { ShareMediaServerAccess404Code } from "./shareMediaServerAccess404Code.ts";

export type ShareMediaServerAccess404 = Problem & {
  code?: ShareMediaServerAccess404Code;
};
