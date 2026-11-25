import type { Problem } from "./problem.ts";
import type { ShareMediaServerAccess422Code } from "./shareMediaServerAccess422Code.ts";

export type ShareMediaServerAccess422 = Problem & {
  code?: ShareMediaServerAccess422Code;
};
