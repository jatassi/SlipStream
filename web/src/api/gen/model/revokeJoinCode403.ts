import type { Problem } from "./problem.ts";
import type { RevokeJoinCode403Code } from "./revokeJoinCode403Code.ts";

export type RevokeJoinCode403 = Problem & {
  code?: RevokeJoinCode403Code;
};
