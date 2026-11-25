import type { Problem } from "./problem.ts";
import type { RunRemoteAccess403Code } from "./runRemoteAccess403Code.ts";

export type RunRemoteAccess403 = Problem & {
  code?: RunRemoteAccess403Code;
};
