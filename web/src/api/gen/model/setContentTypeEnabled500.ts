import type { Problem } from "./problem.ts";
import type { SetContentTypeEnabled500Code } from "./setContentTypeEnabled500Code.ts";

export type SetContentTypeEnabled500 = Problem & {
  code?: SetContentTypeEnabled500Code;
};
