import type { GetEmailLinkByToken422Code } from "./getEmailLinkByToken422Code.ts";
import type { Problem } from "./problem.ts";

export type GetEmailLinkByToken422 = Problem & {
  code?: GetEmailLinkByToken422Code;
};
