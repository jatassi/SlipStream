import type { Problem } from "./problem.ts";
import type { SetContentTypeEnabled404Code } from "./setContentTypeEnabled404Code.ts";

export type SetContentTypeEnabled404 = Problem & {
  code?: SetContentTypeEnabled404Code;
};
