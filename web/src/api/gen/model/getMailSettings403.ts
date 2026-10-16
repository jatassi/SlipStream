import type { GetMailSettings403Code } from "./getMailSettings403Code.ts";
import type { Problem } from "./problem.ts";

export type GetMailSettings403 = Problem & {
  code?: GetMailSettings403Code;
};
