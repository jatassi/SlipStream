import type { GetSystem409Code } from "./getSystem409Code.ts";
import type { Problem } from "./problem.ts";

export type GetSystem409 = Problem & {
  code?: GetSystem409Code;
};
