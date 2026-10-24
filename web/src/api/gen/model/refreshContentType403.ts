import type { Problem } from "./problem.ts";
import type { RefreshContentType403Code } from "./refreshContentType403Code.ts";

export type RefreshContentType403 = Problem & {
  code?: RefreshContentType403Code;
};
