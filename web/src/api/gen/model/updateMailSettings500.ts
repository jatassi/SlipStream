import type { Problem } from "./problem.ts";
import type { UpdateMailSettings500Code } from "./updateMailSettings500Code.ts";

export type UpdateMailSettings500 = Problem & {
  code?: UpdateMailSettings500Code;
};
