import type { Problem } from "./problem.ts";
import type { RunRemoteAccess404Code } from "./runRemoteAccess404Code.ts";

export type RunRemoteAccess404 = Problem & {
  code?: RunRemoteAccess404Code;
};
