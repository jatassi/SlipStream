import type { Problem } from "./problem.ts";
import type { UpdateContentTypeSettings401Code } from "./updateContentTypeSettings401Code.ts";

export type UpdateContentTypeSettings401 = Problem & {
  code?: UpdateContentTypeSettings401Code;
};
