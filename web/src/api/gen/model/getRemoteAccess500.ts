import type { GetRemoteAccess500Code } from "./getRemoteAccess500Code.ts";
import type { Problem } from "./problem.ts";

export type GetRemoteAccess500 = Problem & {
  code?: GetRemoteAccess500Code;
};
