import type { Problem } from "./problem.ts";
import type { UpdateMailSettings401Code } from "./updateMailSettings401Code.ts";

export type UpdateMailSettings401 = Problem & {
  code?: UpdateMailSettings401Code;
};
