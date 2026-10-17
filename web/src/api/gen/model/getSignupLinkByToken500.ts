import type { GetSignupLinkByToken500Code } from "./getSignupLinkByToken500Code.ts";
import type { Problem } from "./problem.ts";

export type GetSignupLinkByToken500 = Problem & {
  code?: GetSignupLinkByToken500Code;
};
