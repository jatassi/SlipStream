import type { Problem } from "./problem.ts";
import type { UpdateMailSettings403Code } from "./updateMailSettings403Code.ts";

export type UpdateMailSettings403 = Problem & {
  code?: UpdateMailSettings403Code;
};
