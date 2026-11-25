import type { Problem } from "./problem.ts";
import type { RunRemoteAccess500Code } from "./runRemoteAccess500Code.ts";

export type RunRemoteAccess500 = Problem & {
  code?: RunRemoteAccess500Code;
};
