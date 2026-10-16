import type { GetResetLinkByToken500Code } from "./getResetLinkByToken500Code.ts";
import type { Problem } from "./problem.ts";

export type GetResetLinkByToken500 = Problem & {
  code?: GetResetLinkByToken500Code;
};
