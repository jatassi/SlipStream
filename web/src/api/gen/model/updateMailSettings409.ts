import type { Problem } from "./problem.ts";
import type { UpdateMailSettings409Code } from "./updateMailSettings409Code.ts";

export type UpdateMailSettings409 = Problem & {
  code?: UpdateMailSettings409Code;
};
