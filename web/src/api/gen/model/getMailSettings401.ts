import type { GetMailSettings401Code } from "./getMailSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetMailSettings401 = Problem & {
  code?: GetMailSettings401Code;
};
