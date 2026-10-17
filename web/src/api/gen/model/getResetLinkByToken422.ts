import type { GetResetLinkByToken422Code } from "./getResetLinkByToken422Code.ts";
import type { Problem } from "./problem.ts";

export type GetResetLinkByToken422 = Problem & {
  code?: GetResetLinkByToken422Code;
};
