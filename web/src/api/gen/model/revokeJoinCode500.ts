import type { Problem } from "./problem.ts";
import type { RevokeJoinCode500Code } from "./revokeJoinCode500Code.ts";

export type RevokeJoinCode500 = Problem & {
  code?: RevokeJoinCode500Code;
};
