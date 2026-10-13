import type { ListContentTypes500Code } from "./listContentTypes500Code.ts";
import type { Problem } from "./problem.ts";

export type ListContentTypes500 = Problem & {
  code?: ListContentTypes500Code;
};
