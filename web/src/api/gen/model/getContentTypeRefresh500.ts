import type { GetContentTypeRefresh500Code } from "./getContentTypeRefresh500Code.ts";
import type { Problem } from "./problem.ts";

export type GetContentTypeRefresh500 = Problem & {
  code?: GetContentTypeRefresh500Code;
};
