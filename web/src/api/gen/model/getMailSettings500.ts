import type { GetMailSettings500Code } from "./getMailSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetMailSettings500 = Problem & {
  code?: GetMailSettings500Code;
};
