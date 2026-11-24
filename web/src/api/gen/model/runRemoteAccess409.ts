import type { Problem } from "./problem.ts";
import type { RunRemoteAccess409Code } from "./runRemoteAccess409Code.ts";

export type RunRemoteAccess409 = Problem & {
  code?: RunRemoteAccess409Code;
};
