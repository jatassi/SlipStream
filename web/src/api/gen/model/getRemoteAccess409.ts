import type { GetRemoteAccess409Code } from "./getRemoteAccess409Code.ts";
import type { Problem } from "./problem.ts";

export type GetRemoteAccess409 = Problem & {
  code?: GetRemoteAccess409Code;
};
