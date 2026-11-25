import type { GetRemoteAccess401Code } from "./getRemoteAccess401Code.ts";
import type { Problem } from "./problem.ts";

export type GetRemoteAccess401 = Problem & {
  code?: GetRemoteAccess401Code;
};
