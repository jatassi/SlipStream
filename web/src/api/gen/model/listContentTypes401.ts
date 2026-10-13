import type { ListContentTypes401Code } from "./listContentTypes401Code.ts";
import type { Problem } from "./problem.ts";

export type ListContentTypes401 = Problem & {
  code?: ListContentTypes401Code;
};
