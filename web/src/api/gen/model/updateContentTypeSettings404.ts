import type { Problem } from "./problem.ts";
import type { UpdateContentTypeSettings404Code } from "./updateContentTypeSettings404Code.ts";

export type UpdateContentTypeSettings404 = Problem & {
  code?: UpdateContentTypeSettings404Code;
};
