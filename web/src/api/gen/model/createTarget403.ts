import type { CreateTarget403Code } from "./createTarget403Code.ts";
import type { Problem } from "./problem.ts";

export type CreateTarget403 = Problem & {
  code?: CreateTarget403Code;
};
