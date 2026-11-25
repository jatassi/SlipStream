import type { GetRemoteAccess403Code } from "./getRemoteAccess403Code.ts";
import type { Problem } from "./problem.ts";

export type GetRemoteAccess403 = Problem & {
  code?: GetRemoteAccess403Code;
};
