import type { Problem } from "./problem.ts";
import type { UpdateContentTypeSettings500Code } from "./updateContentTypeSettings500Code.ts";

export type UpdateContentTypeSettings500 = Problem & {
  code?: UpdateContentTypeSettings500Code;
};
