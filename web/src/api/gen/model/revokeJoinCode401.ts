import type { Problem } from "./problem.ts";
import type { RevokeJoinCode401Code } from "./revokeJoinCode401Code.ts";

export type RevokeJoinCode401 = Problem & {
  code?: RevokeJoinCode401Code;
};
