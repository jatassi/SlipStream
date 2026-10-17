import type { GetEmailLinkByToken500Code } from "./getEmailLinkByToken500Code.ts";
import type { Problem } from "./problem.ts";

export type GetEmailLinkByToken500 = Problem & {
  code?: GetEmailLinkByToken500Code;
};
