import type { Problem } from "./problem.ts";
import type { RevokeJoinCode404Code } from "./revokeJoinCode404Code.ts";

export type RevokeJoinCode404 = Problem & {
  code?: RevokeJoinCode404Code;
};
