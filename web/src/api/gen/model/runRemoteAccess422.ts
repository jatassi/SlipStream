import type { Problem } from "./problem.ts";
import type { RunRemoteAccess422Code } from "./runRemoteAccess422Code.ts";

export type RunRemoteAccess422 = Problem & {
  code?: RunRemoteAccess422Code;
};
