import type { GetContentTypeRefresh401Code } from "./getContentTypeRefresh401Code.ts";
import type { Problem } from "./problem.ts";

export type GetContentTypeRefresh401 = Problem & {
  code?: GetContentTypeRefresh401Code;
};
