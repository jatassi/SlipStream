import type { Problem } from "./problem.ts";
import type { RevokeJoinCode409Code } from "./revokeJoinCode409Code.ts";

export type RevokeJoinCode409 = Problem & {
  code?: RevokeJoinCode409Code;
};
