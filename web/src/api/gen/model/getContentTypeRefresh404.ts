import type { GetContentTypeRefresh404Code } from "./getContentTypeRefresh404Code.ts";
import type { Problem } from "./problem.ts";

export type GetContentTypeRefresh404 = Problem & {
  code?: GetContentTypeRefresh404Code;
};
