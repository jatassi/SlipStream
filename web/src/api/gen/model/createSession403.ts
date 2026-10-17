import type { CreateSession403Code } from "./createSession403Code.ts";
import type { Problem } from "./problem.ts";

export type CreateSession403 = Problem & {
  code?: CreateSession403Code;
};
