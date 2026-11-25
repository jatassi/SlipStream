import type { Problem } from "./problem.ts";
import type { RunRemoteAccess401Code } from "./runRemoteAccess401Code.ts";

export type RunRemoteAccess401 = Problem & {
  code?: RunRemoteAccess401Code;
};
