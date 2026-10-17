import type { Problem } from "./problem.ts";
import type { UpdateMailSettings422Code } from "./updateMailSettings422Code.ts";

export type UpdateMailSettings422 = Problem & {
  code?: UpdateMailSettings422Code;
};
