import type { Problem } from "./problem.ts";
import type { RevokeJoinCode422Code } from "./revokeJoinCode422Code.ts";

export type RevokeJoinCode422 = Problem & {
  code?: RevokeJoinCode422Code;
};
