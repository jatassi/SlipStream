import type { Problem } from "./problem.ts";
import type { SetNaming403Code } from "./setNaming403Code.ts";

export type SetNaming403 = Problem & {
  code?: SetNaming403Code;
};
