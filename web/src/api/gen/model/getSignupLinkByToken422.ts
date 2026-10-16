import type { GetSignupLinkByToken422Code } from "./getSignupLinkByToken422Code.ts";
import type { Problem } from "./problem.ts";

export type GetSignupLinkByToken422 = Problem & {
  code?: GetSignupLinkByToken422Code;
};
